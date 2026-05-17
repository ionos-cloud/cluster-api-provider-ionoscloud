# Upgrade Cluster API to v1.10.10 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade `sigs.k8s.io/cluster-api` from v1.9.11 to v1.10.10, together with the required controller-runtime and k8s.io dependency bumps, function renames in e2e tests, and e2e config updates.

**Architecture:** Bump the three dependency groups (cluster-api, controller-runtime, k8s.io/*) in `go.mod`, run `make tidy` to resolve the full transitive graph, then fix all resulting compilation errors. Apply the two renamed e2e helper functions, update the e2e clusterctl config URLs/version names, and create the new CAPI v1.9 metadata file used during upgrade tests.

**Tech Stack:** Go 1.25, cluster-api v1.10.10, controller-runtime v0.20.4, k8s.io v0.32.3, Ginkgo/Gomega for e2e tests, clusterctl for provider management.

---

## File Map

| File | Change |
|---|---|
| `go.mod` | Bump cluster-api, cluster-api/test, controller-runtime, k8s.io/* |
| `go.sum` | Auto-updated by `go mod tidy` |
| `test/e2e/suite_test.go` | Rename `GetVariable` → `MustGetVariable` (2 call sites) |
| `test/e2e/config/ionoscloud.yaml` | Update URLs v1.9.11 → v1.10.10; version names v1.8.5 → v1.9.0; metadata paths v1.8 → v1.9 |
| `test/e2e/data/shared/v1.9/metadata.yaml` | New file: CAPI v1.9 release-series metadata for clusterctl |
| Any source file broken by API changes | Compilation fixes discovered during `make build` |

---

### Task 1: Bump cluster-api, controller-runtime, and k8s.io in go.mod

**Files:**
- Modify: `go.mod`

The direct dependencies must be updated in lockstep. `go mod tidy` will update `go.sum` and indirect deps automatically.

- [ ] **Step 1: Edit the direct-dependency versions in go.mod**

In `go.mod`, change the following lines inside the `require (...)` block:

```
# Before
sigs.k8s.io/cluster-api v1.9.11
sigs.k8s.io/cluster-api/test v1.9.11
sigs.k8s.io/controller-runtime v0.19.6
k8s.io/api v0.31.3
k8s.io/apimachinery v0.31.3
k8s.io/client-go v0.31.3

# After
sigs.k8s.io/cluster-api v1.10.10
sigs.k8s.io/cluster-api/test v1.10.10
sigs.k8s.io/controller-runtime v0.20.4
k8s.io/api v0.32.3
k8s.io/apimachinery v0.32.3
k8s.io/client-go v0.32.3
```

- [ ] **Step 2: Update indirect k8s.io transitive deps in go.mod**

In the `require (...)` indirect block, update these entries (they will be pinned to match k8s v0.32.3):

```
# Before
k8s.io/apiextensions-apiserver v0.31.3 // indirect
k8s.io/apiserver v0.31.3 // indirect
k8s.io/cluster-bootstrap v0.31.3 // indirect
k8s.io/component-base v0.31.3 // indirect

# After
k8s.io/apiextensions-apiserver v0.32.3 // indirect
k8s.io/apiserver v0.32.3 // indirect
k8s.io/cluster-bootstrap v0.32.3 // indirect
k8s.io/component-base v0.32.3 // indirect
```

- [ ] **Step 3: Run make tidy to resolve the full module graph**

```bash
make tidy
```

Expected: exits 0, `go.sum` updated, no error output. If tidy reports conflicts, check that all `k8s.io/*` pins are consistently at v0.32.3.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: bump cluster-api to v1.10.10, controller-runtime to v0.20.4, k8s to v0.32.3"
```

---

### Task 2: Fix compilation errors from dependency bumps

**Files:**
- Modify: any `.go` file that fails to compile (discovered during this task)

After bumping dependencies, build the project to surface API breakage.

- [ ] **Step 1: Attempt a full build**

```bash
make build
```

Expected: exits 0. If it does not, proceed to the next step.

- [ ] **Step 2: Triage each compilation error**

For each error produced, apply the minimal fix. Common breakage patterns for this version delta:

**controller-runtime v0.19 → v0.20 known changes:**
- `ctrl.Options` field `Scheme` is unchanged; `MetricsBindAddress` is unchanged.
- If you see `undefined: manager.Options.LeaderElectionConfig` → the field was renamed; check the controller-runtime v0.20 changelog and update the field name in `cmd/main.go`.
- Webhook-related `server.Options` struct changes: if `Port` or `CertDir` fields are missing, they moved under a `Options` sub-struct.

**k8s.io v0.31 → v0.32 known changes:**
- Generally backward-compatible for provider code; no expected breakage in `api/v1alpha1/` or `internal/`.

**cluster-api v1.9 → v1.10 known changes:**
- No provider-facing API removals. If anything in `scope/` or `internal/controller/` breaks, check the import paths are still valid.

- [ ] **Step 3: After fixing all errors, verify build passes**

```bash
make build
```

Expected: exits 0 with no errors.

- [ ] **Step 4: Run unit tests to confirm no regressions**

```bash
make unit-test
```

Expected: all tests pass (PASS on each package).

- [ ] **Step 5: Commit compilation fixes**

```bash
git add -p   # stage only the compilation-fix hunks
git commit -m "fix: resolve compilation errors after cluster-api v1.10 / controller-runtime v0.20 upgrade"
```

---

### Task 3: Fix renamed e2e test-framework functions

**Files:**
- Modify: `test/e2e/suite_test.go` (lines 222 and 236)

`clusterctl.E2EConfig.GetVariable` was renamed to `MustGetVariable` in CAPI v1.10. There are exactly two call sites.

- [ ] **Step 1: Verify the call sites**

Open `test/e2e/suite_test.go` and confirm the two lines:
- Line 222: `cniPath := config.GetVariable(CNIPath)`
- Line 236: `KubernetesVersion: e2eConfig.GetVariable(KubernetesVersion),`

- [ ] **Step 2: Replace both occurrences**

In `test/e2e/suite_test.go`:

Line 222 — change:
```go
cniPath := config.GetVariable(CNIPath)
```
to:
```go
cniPath := config.MustGetVariable(CNIPath)
```

Line 236 — change:
```go
KubernetesVersion: e2eConfig.GetVariable(KubernetesVersion),
```
to:
```go
KubernetesVersion: e2eConfig.MustGetVariable(KubernetesVersion),
```

- [ ] **Step 3: Confirm the build still compiles**

```bash
make build
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add test/e2e/suite_test.go
git commit -m "fix(e2e): rename GetVariable to MustGetVariable per CAPI v1.10 migration"
```

---

### Task 4: Update e2e clusterctl config to reference v1.10.10

**Files:**
- Modify: `test/e2e/config/ionoscloud.yaml`

The CAPI component download URLs must point to v1.10.10. The version name in the config is used by clusterctl upgrade tests to simulate "upgrading from v1.9.0"; update it from `v1.8.5` to `v1.9.0`. The metadata source path moves from `v1.8` to `v1.9`.

- [ ] **Step 1: Update CoreProvider entry**

In `test/e2e/config/ionoscloud.yaml`, find the CoreProvider block and apply:

```yaml
# Before
- name: "v1.8.5"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.9.11/core-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.8/metadata.yaml"

# After
- name: "v1.9.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/core-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.9/metadata.yaml"
```

- [ ] **Step 2: Update BootstrapProvider entry**

Same pattern for the kubeadm bootstrap provider:

```yaml
# Before
- name: "v1.8.5"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.9.11/bootstrap-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.8/metadata.yaml"

# After
- name: "v1.9.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/bootstrap-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.9/metadata.yaml"
```

- [ ] **Step 3: Update ControlPlaneProvider entry**

```yaml
# Before
- name: "v1.8.5"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.9.11/control-plane-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.8/metadata.yaml"

# After
- name: "v1.9.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/control-plane-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.9/metadata.yaml"
```

- [ ] **Step 4: Commit**

```bash
git add test/e2e/config/ionoscloud.yaml
git commit -m "chore(e2e): update CAPI component URLs to v1.10.10"
```

---

### Task 5: Create CAPI v1.9 shared metadata file for e2e

**Files:**
- Create: `test/e2e/data/shared/v1.9/metadata.yaml`

This file tells clusterctl which CAPI release series are compatible when the e2e test downloads CAPI v1.9.0 components. It is referenced by the sourcePath entries updated in Task 4.

- [ ] **Step 1: Create the directory and metadata file**

Create `test/e2e/data/shared/v1.9/metadata.yaml` with this content:

```yaml
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: 1
    minor: 9
    contract: v1beta1
  - major: 1
    minor: 8
    contract: v1beta1
  - major: 1
    minor: 7
    contract: v1beta1
  - major: 1
    minor: 6
    contract: v1beta1
  - major: 1
    minor: 5
    contract: v1beta1
  - major: 1
    minor: 4
    contract: v1beta1
  - major: 1
    minor: 3
    contract: v1beta1
  - major: 1
    minor: 2
    contract: v1beta1
  - major: 1
    minor: 1
    contract: v1beta1
  - major: 1
    minor: 0
    contract: v1beta1
  - major: 0
    minor: 4
    contract: v1alpha4
  - major: 0
    minor: 3
    contract: v1alpha3
```

- [ ] **Step 2: Commit**

```bash
git add test/e2e/data/shared/v1.9/metadata.yaml
git commit -m "chore(e2e): add CAPI v1.9 shared metadata for upgrade tests"
```

---

### Task 6: Regenerate generated code and mocks

**Files:**
- Modify: `api/v1alpha1/zz_generated.deepcopy.go` (auto-generated)
- Modify: any files under `internal/ionoscloud/clienttest/` (auto-generated mocks)
- Modify: any CRD/RBAC manifests under `config/` (auto-generated)

Upgrading cluster-api may cause the code generator to emit different output. Always regenerate after a dependency bump.

- [ ] **Step 1: Regenerate DeepCopy methods and CRD manifests**

```bash
make generate
make manifests
```

Expected: exits 0. If markers reference types removed in CAPI v1.10, fix the markers in `api/v1alpha1/` first.

- [ ] **Step 2: Regenerate mocks**

```bash
make mocks
```

Expected: exits 0.

- [ ] **Step 3: Check for unexpected diff**

```bash
git diff --stat
```

Review the diff. Any changes to generated files are expected. Any changes to non-generated files signal that a generator is modifying source — investigate before committing.

- [ ] **Step 4: Commit regenerated files**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/ internal/ionoscloud/clienttest/
git commit -m "chore: regenerate mocks and manifests after cluster-api v1.10 upgrade"
```

---

### Task 7: Add --skip-crd-migration-phases flag (recommended)

**Files:**
- Modify: `cmd/main.go`
- Modify: RBAC config under `config/rbac/` (if the CRDMigrator requires additional permissions)

CAPI v1.10 introduced a `CRDMigrator` component to replace clusterctl-based CRD storage migration. Providers are recommended to add a `--skip-crd-migration-phases` flag now; this becomes mandatory at v1.13.

The flag allows operators to skip individual CRD migration phases that the new CRDMigrator now owns, preventing double-migration.

- [ ] **Step 1: Read cmd/main.go to understand the current flag setup**

Open `cmd/main.go` and note where `pflag` or the existing flags are defined (look for `flag.StringVar`, `flag.BoolVar`, or `pflag` equivalents near the top of `main()`).

- [ ] **Step 2: Add the flag definition**

In `cmd/main.go`, add the flag alongside the existing flags. Example pattern (adjust variable names to match the existing code style):

```go
var skipCRDMigrationPhases []string
flag.StringArrayVar(&skipCRDMigrationPhases, "skip-crd-migration-phases", []string{},
    "CRD migration phases to skip (e.g. StorageVersionMigration). "+
        "Used when the CAPI CRDMigrator handles migration instead of clusterctl.")
```

- [ ] **Step 3: Verify build still compiles**

```bash
make build
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git add cmd/main.go
git commit -m "feat: add --skip-crd-migration-phases flag per CAPI v1.10 provider recommendation"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run full test suite**

```bash
make test
```

Expected: all unit and integration tests pass.

- [ ] **Step 2: Run linter**

```bash
make lint
```

Expected: exits 0 with no new lint errors. If there are formatting issues, run `make lint-fix` first.

- [ ] **Step 3: Verify generated files are up-to-date (CI gate)**

```bash
make verify
```

Expected: exits 0. If it fails, the generated files are stale — re-run the relevant `make generate` / `make manifests` / `make mocks` target and commit.

- [ ] **Step 4: Final commit if any lint-fix changes remain**

```bash
git add -p
git commit -m "chore: lint-fix and verify after cluster-api v1.10.10 upgrade"
```
