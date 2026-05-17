# CRDMigrator Controller Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire up a CRDMigrator controller in `cmd/main.go` that uses the already-parsed `--skip-crd-migration-phases` flag, enabling storage version migration and managed fields cleanup for this provider's CRDs.

**Architecture:** The CRDMigrator is a controller provided by `sigs.k8s.io/cluster-api/controllers/crdmigrator`. It watches CRDs owned by this provider and performs two phases: StorageVersionMigration (no-op patches to migrate stored versions) and CleanupManagedFields (removes stale managed field entries). The provider registers its own CRDs (IonosCloudCluster, IonosCloudMachine, IonosCloudClusterTemplate, IonosCloudMachineTemplate) in the migrator config, then sets up the controller with concurrency 1 to avoid overwhelming the API server.

**Tech Stack:** Go, controller-runtime v0.21, cluster-api v1.11.7 `controllers/crdmigrator` package

**Reference:** https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.10-to-v1.11#suggested-changes-for-providers and https://github.com/kubernetes-sigs/cluster-api/pull/11889

---

### Task 1: Add RBAC markers for CRD access and template CRs

**Files:**
- Modify: `cmd/main.go`

- [ ] **Step 1: Add RBAC markers for CRD read/write access**

Add these RBAC markers in `cmd/main.go` (near the existing RBAC markers for `tokenreviews`):

```go
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions;customresourcedefinitions/status,verbs=update;patch,resourceNames=ionoscloudclusters.infrastructure.cluster.x-k8s.io;ionoscloudclustertemplates.infrastructure.cluster.x-k8s.io;ionoscloudmachines.infrastructure.cluster.x-k8s.io;ionoscloudmachinetemplates.infrastructure.cluster.x-k8s.io
```

The first marker allows reading all CRDs (needed to discover which ones need migration). The second restricts write access to only this provider's CRDs.

- [ ] **Step 2: Add RBAC markers for template CR access**

The CRDMigrator needs `get;list;watch;patch;update` on all CRs it migrates. The existing controller RBAC already covers `ionoscloudclusters` and `ionoscloudmachines` with these verbs. However, **templates have no RBAC at all**. Add:

```go
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudclustertemplates;ionoscloudmachinetemplates,verbs=get;list;watch;patch;update
```

Note: Templates do not have status subresources, so no `/status` RBAC is needed for them. For `ionoscloudclusters/status` and `ionoscloudmachines/status`, the existing RBAC already includes `update;patch`.

- [ ] **Step 3: Verify compilation**

Run: `go build ./cmd/...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add cmd/main.go
git commit -m "chore: add RBAC markers for CRDMigrator and template CRs"
```

---

### Task 2: Add imports, scheme registration, and wire up CRDMigrator

**Files:**
- Modify: `cmd/main.go`

- [ ] **Step 1: Add imports**

Add to the import block:

```go
apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
"sigs.k8s.io/cluster-api/controllers/crdmigrator"
```

- [ ] **Step 2: Register apiextensions scheme**

Add to `init()`:

```go
utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
```

- [ ] **Step 3: Wire up CRDMigrator controller**

After the existing reconciler setup blocks (after the `IonosCloudMachineReconciler` setup) and before `//+kubebuilder:scaffold:builder`, add:

```go
crdMigratorConfig := map[client.Object]crdmigrator.ByObjectConfig{
    &infrav1.IonosCloudCluster{}:         {UseCache: true, UseStatusForStorageVersionMigration: true},
    &infrav1.IonosCloudMachine{}:         {UseCache: true, UseStatusForStorageVersionMigration: true},
    &infrav1.IonosCloudClusterTemplate{}: {UseCache: false},
    &infrav1.IonosCloudMachineTemplate{}: {UseCache: false},
}
crdMigratorSkipPhases := make([]crdmigrator.Phase, 0, len(skipCRDMigrationPhases))
for _, p := range skipCRDMigrationPhases {
    crdMigratorSkipPhases = append(crdMigratorSkipPhases, crdmigrator.Phase(p))
}
if err := (&crdmigrator.CRDMigrator{
    Client:                 mgr.GetClient(),
    APIReader:              mgr.GetAPIReader(),
    SkipCRDMigrationPhases: crdMigratorSkipPhases,
    Config:                 crdMigratorConfig,
    // Run with concurrency 1 to avoid overwhelming the apiserver.
}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: 1}); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "CRDMigrator")
    os.Exit(1)
}
```

Notes on config choices:
- `UseCache: true` for `IonosCloudCluster` and `IonosCloudMachine` — the reconcilers already watch these types, so informers exist.
- `UseCache: false` for `IonosCloudClusterTemplate` and `IonosCloudMachineTemplate` — no existing informer (templates are not reconciled), so the migrator should use direct API reads.
- `UseStatusForStorageVersionMigration: true` for types with status subresources (`IonosCloudCluster`, `IonosCloudMachine`) — the no-op patch targets the status to avoid triggering reconciliation. Templates don't have status subresources, so this is `false` (the default).
- Concurrency is 1 per CAPI's recommendation to avoid overwhelming the API server.

- [ ] **Step 4: Verify compilation**

Run: `go build ./cmd/...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add cmd/main.go
git commit -m "feat: wire up CRDMigrator controller with skip-phases support"
```

---

### Task 3: Regenerate manifests and verify

**Files:**
- Modify: `config/rbac/role.yaml` (generated)

- [ ] **Step 1: Regenerate manifests**

Run: `make manifests`
Expected: RBAC manifests updated with the new apiextensions and template permissions.

- [ ] **Step 2: Verify generated RBAC contains CRD permissions**

```bash
grep -A 5 "apiextensions" config/rbac/role.yaml
```

Expected: Rules for `customresourcedefinitions` with get;list;watch and scoped update;patch with resourceNames.

- [ ] **Step 3: Verify generated RBAC contains template CR permissions**

```bash
grep "ionoscloudclustertemplates\|ionoscloudmachinetemplates" config/rbac/role.yaml
```

Expected: Template resources with get;list;watch;patch;update verbs.

- [ ] **Step 4: Run unit tests**

Run: `make unit-test`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add config/rbac/
git commit -m "chore: regenerate RBAC manifests for CRDMigrator"
```

---

### Important: GitOps annotation caveat

The CRDMigrator adds a `crd-migration.cluster.x-k8s.io/observed-generation` annotation to CRD objects it manages. If these CRDs are deployed via GitOps tools (kapp, Argo CD, Flux), ensure the annotation is **not** continuously removed by the tool. For example:
- **Argo CD**: Add the annotation to `resource.customizations.ignoreDifferences`
- **Flux**: Use `kustomize.toolkit.fluxcd.io/ssa: IfNotPresent` or equivalent
- **kapp**: Use `kapp.k14s.io/nonce` or annotation exclusion rules

Without this, the CRDMigrator and the GitOps tool will fight over the annotation, causing continuous reconciliation.
