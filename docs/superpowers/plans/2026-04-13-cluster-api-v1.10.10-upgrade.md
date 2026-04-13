# Cluster API v1.10.10 Upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade CAPI from v1.8.12 to v1.10.10 with all migration guide recommendations, without adopting v1beta2 conditions.

**Architecture:** Single jump from v1.8.12 to v1.10.10. PR 1 covers version bumps and migration guide fixes (predicates, e2e renames, addons import path, CRDMigrator). PR 2 removes deprecated FailureReason/FailureMessage fields.

**Tech Stack:** Go, Cluster API v1.10.10, controller-runtime v0.20.x, k8s.io v0.32.x

**Spec:** `docs/superpowers/specs/2026-04-13-cluster-api-v1.10.10-upgrade-design.md`

**Reference PRs:** #352 (v1.8->v1.9), #353 (v1.9->v1.10) — use `gh pr diff <num>` for exact diffs.

---

## PR 1: Version Bump + Migration Guide Recommendations

### Task 1: Bump go.mod Dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Update direct dependencies in go.mod**

Edit `go.mod` to bump the three CAPI/k8s dependency groups. The exact target versions must align with what CAPI v1.10.10 uses in its own `go.mod`. Check https://github.com/kubernetes-sigs/cluster-api/blob/v1.10.10/go.mod for the exact versions.

Expected changes to the `require` block:

```go
// These will change (approximate targets — verify from CAPI v1.10.10 go.mod):
k8s.io/api v0.32.x
k8s.io/apimachinery v0.32.x
k8s.io/client-go v0.32.x
k8s.io/klog/v2 v2.13x.x
sigs.k8s.io/cluster-api v1.10.10
sigs.k8s.io/cluster-api/test v1.10.10
sigs.k8s.io/controller-runtime v0.20.x
```

Also add a new direct dependency that the CRDMigrator needs:

```go
k8s.io/apiextensions-apiserver v0.32.x
```

- [ ] **Step 2: Run go mod tidy**

```bash
make tidy
```

This will resolve all transitive dependencies and update `go.sum`. Expect significant churn in indirect deps.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: compilation errors in files that use changed APIs (predicates, GetVariable, addons import). This is expected — we fix them in subsequent tasks.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): bump cluster-api v1.8.12 -> v1.10.10 and dependencies"
```

---

### Task 2: Fix Predicate Signatures (v1.8->v1.9 Migration)

**Files:**
- Modify: `internal/controller/ionoscloudcluster_controller.go:264,273`

The CAPI v1.9 predicates `ResourceNotPaused` and `ClusterUnpaused` now require a `runtime.Scheme` as the first argument.

- [ ] **Step 1: Update ResourceNotPaused call**

In `internal/controller/ionoscloudcluster_controller.go`, the `SetupWithManager` method (line 261-275):

Change line 264 from:

```go
WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
```

to:

```go
WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
```

- [ ] **Step 2: Update ClusterUnpaused call**

Change line 273 from:

```go
builder.WithPredicates(predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx))),
```

to:

```go
builder.WithPredicates(predicates.ClusterUnpaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))),
```

- [ ] **Step 3: Commit**

```bash
git add internal/controller/ionoscloudcluster_controller.go
git commit -m "fix: update predicate signatures for CAPI v1.9+ (scheme argument)"
```

---

### Task 3: Fix E2E GetVariable -> MustGetVariable (v1.9->v1.10 Migration)

**Files:**
- Modify: `test/e2e/suite_test.go:222,236`

- [ ] **Step 1: Rename GetVariable to MustGetVariable**

In `test/e2e/suite_test.go`, change line 222 from:

```go
cniPath := config.GetVariable(CNIPath)
```

to:

```go
cniPath := config.MustGetVariable(CNIPath)
```

And change line 236 from:

```go
KubernetesVersion: e2eConfig.GetVariable(KubernetesVersion),
```

to:

```go
KubernetesVersion: e2eConfig.MustGetVariable(KubernetesVersion),
```

- [ ] **Step 2: Commit**

```bash
git add test/e2e/suite_test.go
git commit -m "fix(e2e): rename GetVariable to MustGetVariable for CAPI v1.10"
```

---

### Task 4: Fix Addons Import Path (v1.9->v1.10 Migration)

**Files:**
- Modify: `test/e2e/helpers/finalizers.go:26`
- Modify: `test/e2e/helpers/ownerreference.go:27`

The addons API graduated from experimental in CAPI v1.10. The import path changed from `sigs.k8s.io/cluster-api/exp/addons/api/v1beta1` to `sigs.k8s.io/cluster-api/api/addons/v1beta1`.

- [ ] **Step 1: Update import in finalizers.go**

In `test/e2e/helpers/finalizers.go`, change line 26 from:

```go
addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
```

to:

```go
addonsv1 "sigs.k8s.io/cluster-api/api/addons/v1beta1"
```

- [ ] **Step 2: Update import in ownerreference.go**

In `test/e2e/helpers/ownerreference.go`, change line 27 from:

```go
addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
```

to:

```go
addonsv1 "sigs.k8s.io/cluster-api/api/addons/v1beta1"
```

Note: The `gci` import sorter will move this line to its correct alphabetical position in the import block. Run `make lint-fix` after editing to auto-sort.

- [ ] **Step 3: Commit**

```bash
git add test/e2e/helpers/finalizers.go test/e2e/helpers/ownerreference.go
git commit -m "fix(e2e): update addons import path (graduated from experimental)"
```

---

### Task 5: Add CRDMigrator Controller (v1.9->v1.10 Migration)

**Files:**
- Modify: `cmd/main.go`

This is the most significant change. The CAPI v1.10 migration guide recommends adding a `CRDMigrator` controller so that CRD migration is handled by the provider itself, rather than relying on clusterctl (which is being deprecated for this purpose).

- [ ] **Step 1: Add new imports to cmd/main.go**

Add these imports to the import block in `cmd/main.go`:

```go
"context"
"fmt"

apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
"sigs.k8s.io/cluster-api/controllers/crdmigrator"
"sigs.k8s.io/controller-runtime/pkg/client"
```

The full import block should become:

```go
import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/crdmigrator"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/flags"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	iccontroller "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/controller"
)
```

- [ ] **Step 2: Add skipCRDMigrationPhases variable**

Add to the `var` block:

```go
var (
	scheme                 = runtime.NewScheme()
	setupLog               = ctrl.Log.WithName("setup")
	healthProbeAddr        string
	enableLeaderElection   bool
	managerOptions         = flags.ManagerOptions{}
	skipCRDMigrationPhases []string

	icClusterConcurrency int
	icMachineConcurrency int
)
```

- [ ] **Step 3: Register apiextensionsv1 scheme**

In the `init()` function, add after `clientgoscheme.AddToScheme(scheme)`:

```go
utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
```

- [ ] **Step 4: Add RBAC markers for CRDMigrator**

Add these RBAC markers above the `main()` function, after the existing authorization markers:

```go
// Add RBAC for CRDMigrator controller.
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions/status,verbs=get;patch
```

- [ ] **Step 5: Extract controller setup into setupControllers function**

Replace the inline controller setup in `main()` with a call to a new function. Change:

```go
if err = iccontroller.NewIonosCloudClusterReconciler(mgr).SetupWithManager(
	ctx,
	mgr,
	controller.Options{MaxConcurrentReconciles: icClusterConcurrency},
); err != nil {
	setupLog.Error(err, "unable to create controller", "controller", "IonosCloudCluster")
	os.Exit(1)
}
if err = iccontroller.NewIonosCloudMachineReconciler(mgr).SetupWithManager(
	mgr,
	controller.Options{MaxConcurrentReconciles: icMachineConcurrency},
); err != nil {
	setupLog.Error(err, "unable to create controller", "controller", "IonosCloudMachine")
	os.Exit(1)
}
```

to:

```go
if err = setupControllers(ctx, mgr); err != nil {
	setupLog.Error(err, "unable to setup controllers")
	os.Exit(1)
}
```

- [ ] **Step 6: Add setupControllers function**

Add this function after `main()`, before `initFlags()`:

```go
func setupControllers(ctx context.Context, mgr ctrl.Manager) error {
	skipPhases := make([]crdmigrator.Phase, 0, len(skipCRDMigrationPhases))
	for _, p := range skipCRDMigrationPhases {
		skipPhases = append(skipPhases, crdmigrator.Phase(p))
	}
	if err := (&crdmigrator.CRDMigrator{
		Client:                 mgr.GetClient(),
		APIReader:              mgr.GetAPIReader(),
		SkipCRDMigrationPhases: skipPhases,
		Config: map[client.Object]crdmigrator.ByObjectConfig{
			&infrav1.IonosCloudCluster{}:         {UseCache: true},
			&infrav1.IonosCloudClusterTemplate{}: {UseCache: false},
			&infrav1.IonosCloudMachine{}:         {UseCache: true},
			&infrav1.IonosCloudMachineTemplate{}: {UseCache: false},
		},
	}).SetupWithManager(ctx, mgr, controller.Options{}); err != nil {
		return fmt.Errorf("unable to create CRDMigrator controller: %w", err)
	}

	if err := iccontroller.NewIonosCloudClusterReconciler(mgr).SetupWithManager(
		ctx,
		mgr,
		controller.Options{MaxConcurrentReconciles: icClusterConcurrency},
	); err != nil {
		return fmt.Errorf("unable to create IonosCloudCluster controller: %w", err)
	}
	if err := iccontroller.NewIonosCloudMachineReconciler(mgr).SetupWithManager(
		mgr,
		controller.Options{MaxConcurrentReconciles: icMachineConcurrency},
	); err != nil {
		return fmt.Errorf("unable to create IonosCloudMachine controller: %w", err)
	}
	return nil
}
```

Note: `UseCache: true` is set for types that already have informers (Cluster, Machine), `UseCache: false` for template types that don't.

- [ ] **Step 7: Add --skip-crd-migration-phases flag**

In `initFlags()`, add after the `leader-elect` flag:

```go
pflag.StringArrayVar(&skipCRDMigrationPhases, "skip-crd-migration-phases", []string{},
	"CRD migration phases to skip. Use when the CAPI CRDMigrator handles migration.")
```

- [ ] **Step 8: Commit**

```bash
git add cmd/main.go
git commit -m "feat: add CRDMigrator controller per CAPI v1.10 migration guide"
```

---

### Task 6: Update golangci-lint Configuration

**Files:**
- Modify: `.golangci.yml`

- [ ] **Step 1: Add comment-spacings exception for +list marker**

In `.golangci.yml`, the `comment-spacings` revive rule exception (line 149) needs to also allow `+list` markers used by CAPI v1.10. Change:

```yaml
      source: "//(\\+(kubebuilder|optional|required)|#nosec)"
```

to:

```yaml
      source: "//(\\+(kubebuilder|optional|required|list)|#nosec)"
```

- [ ] **Step 2: Commit**

```bash
git add .golangci.yml
git commit -m "chore: update golangci-lint config for CAPI v1.10 markers"
```

---

### Task 7: Update E2E Config and Add v1.10 Metadata

**Files:**
- Modify: `test/e2e/config/ionoscloud.yaml`
- Create: `test/e2e/data/shared/v1.10/metadata.yaml`

- [ ] **Step 1: Update ionoscloud.yaml component URLs**

In `test/e2e/config/ionoscloud.yaml`, update all three CAPI provider entries:

Replace all occurrences of `name: "v1.8.5"` with `name: "v1.10.10"`.
Replace all component URLs from `v1.8.1` to `v1.10.10`.
Replace all metadata sourcePaths from `v1.8` to `v1.10`.

The providers section should become:

```yaml
providers:
  - name: cluster-api
    type: CoreProvider
    versions:
    - name: "v1.10.10"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/core-components.yaml"
      type: url
      contract: v1beta1
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.10/metadata.yaml"
  - name: kubeadm
    type: BootstrapProvider
    versions:
    - name: "v1.10.10"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/bootstrap-components.yaml"
      type: url
      contract: v1beta1
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.10/metadata.yaml"
  - name: kubeadm
    type: ControlPlaneProvider
    versions:
      - name: "v1.10.10"
        value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/control-plane-components.yaml"
        type: url
        contract: v1beta1
        replacements:
          - old: --metrics-addr=127.0.0.1:8080
            new: --metrics-addr=:8443
        files:
          - sourcePath: "../data/shared/v1.10/metadata.yaml"
```

- [ ] **Step 2: Create v1.10 metadata.yaml**

Create `test/e2e/data/shared/v1.10/metadata.yaml`:

```yaml
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: 1
    minor: 10
    contract: v1beta1
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

- [ ] **Step 3: Commit**

```bash
git add test/e2e/config/ionoscloud.yaml test/e2e/data/shared/v1.10/metadata.yaml
git commit -m "chore(e2e): update CAPI component URLs to v1.10.10"
```

---

### Task 8: Update README Compatibility Table

**Files:**
- Modify: `README.md:50-56`

- [ ] **Step 1: Update the compatibility table**

In `README.md`, replace the current compatibility table (lines 50-56) with an updated version adding the v1.10 column. The new CAPIC release version for v1.10 compatibility should be the next release (check with the user if unsure, but likely v0.7):

```markdown
|                       | Cluster API v1beta1 (v1.7) | Cluster API v1beta1 (v1.8) | Cluster API v1beta1 (v1.10) |
|-----------------------|:--------------------------:|:--------------------------:|:---------------------------:|
| CAPIC v1alpha1 (v0.2) |             ✓              |             ☓              |              ☓              |
| CAPIC v1alpha1 (v0.3) |             ✓              |             ☓              |              ☓              |
| CAPIC v1alpha1 (v0.4) |             ✓              |             ✓              |              ☓              |
| CAPIC v1alpha1 (v0.5) |             ✓              |             ✓              |              ☓              |
| CAPIC v1alpha1 (v0.6) |             ✓              |             ✓              |              ☓              |
| CAPIC v1alpha1 (v0.7) |             ☓              |             ☓              |              ✓              |
```

Note: Verify the next CAPIC release version with the team. The v0.7 row assumes this upgrade ships as v0.7.

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update compatibility table for CAPI v1.10"
```

---

### Task 9: Regenerate Manifests, DeepCopy, and Mocks

**Files:**
- Modified by generation: `config/rbac/role.yaml`, `config/crd/bases/*.yaml`, `api/v1alpha1/zz_generated.deepcopy.go`, `internal/ionoscloud/clienttest/mock_client.go`

- [ ] **Step 1: Regenerate RBAC and CRD manifests**

```bash
make manifests
```

Expected: `config/rbac/role.yaml` will gain the new `apiextensions.k8s.io` rules from the CRDMigrator RBAC markers. CRD files may have minor changes from the controller-gen version.

- [ ] **Step 2: Regenerate deepcopy**

```bash
make generate
```

- [ ] **Step 3: Regenerate mocks**

```bash
make mocks
```

Check if `internal/ionoscloud/clienttest/mock_client.go` changed. If the `Client` interface didn't change, this is a no-op.

- [ ] **Step 4: Run go mod tidy**

```bash
make tidy
```

- [ ] **Step 5: Run lint-fix**

```bash
make lint-fix
```

This runs `gci` (import sorting) and `gofumpt` (formatting) across all files.

- [ ] **Step 6: Commit all generated changes**

```bash
git add -A
git commit -m "chore: regenerate manifests, deepcopy, and mocks for CAPI v1.10.10"
```

---

### Task 10: Verify — Tests, Lint, and Generated Files

- [ ] **Step 1: Run unit tests**

```bash
make unit-test
```

Expected: All tests pass. If any fail, fix them before proceeding.

- [ ] **Step 2: Run integration tests**

```bash
make integration-test
```

Expected: All tests pass.

- [ ] **Step 3: Run linter**

```bash
make lint
```

Expected: No lint errors.

- [ ] **Step 4: Run verify**

```bash
make verify
```

Expected: Generated files and go.mod/go.sum are up-to-date.

- [ ] **Step 5: Run build**

```bash
make build
```

Expected: Clean build with no errors.

---

## PR 2: Remove Deprecated FailureReason/FailureMessage

This PR should be based on the branch from PR 1 (or on main after PR 1 is merged).

### Task 11: Remove FailureReason/FailureMessage from Machine Types

**Files:**
- Modify: `api/v1alpha1/ionoscloudmachine_types.go:24,316,335`

- [ ] **Step 1: Remove the errors import**

In `api/v1alpha1/ionoscloudmachine_types.go`, remove the import (line 24):

```go
"sigs.k8s.io/cluster-api/errors"
```

- [ ] **Step 2: Remove FailureReason field**

Remove the `FailureReason` field from `IonosCloudMachineStatus` (around line 316):

```go
// FailureReason will be set in the event that there is a terminal problem
// reconciling the Machine and will contain a succinct value suitable
// for machine interpretation.
//
// This field should not be set for transitive errors that a controller
// faces that are expected to be fixed automatically over
// time (like service outages), but instead indicate that something is
// fundamentally wrong with the Machine's spec or the configuration of
// the controller, and that manual intervention is required. Examples
// of terminal errors would be invalid combinations of settings in the
// spec, values that are unsupported by the controller, or the
// responsible controller itself being critically misconfigured.
//
// Any transient errors that occur during the reconciliation of IonosCloudMachines
// can be added as events to the IonosCloudMachine object and/or logged in the
// controller's output.
// +optional
FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`
```

- [ ] **Step 3: Remove FailureMessage field**

Remove the `FailureMessage` field (around line 335):

```go
// FailureMessage will be set in the event that there is a terminal problem
// reconciling the Machine and will contain a more verbose string suitable
// for logging and human consumption.
//
// This field should not be set for transitive errors that a controller
// faces that are expected to be fixed automatically over
// time (like service outages), but instead indicate that something is
// fundamentally wrong with the Machine's spec or the configuration of
// the controller, and that manual intervention is required. Examples
// of terminal errors would be invalid combinations of settings in the
// spec, values that are unsupported by the controller, or the
// responsible controller itself being critically misconfigured.
//
// Any transient errors that occur during the reconciliation of IonosCloudMachines
// can be added as events to the IonosCloudMachine object and/or logged in the
// controller's output.
// +optional
FailureMessage *string `json:"failureMessage,omitempty"`
```

- [ ] **Step 4: Commit**

```bash
git add api/v1alpha1/ionoscloudmachine_types.go
git commit -m "feat(api): remove deprecated FailureReason/FailureMessage fields"
```

---

### Task 12: Remove HasFailed Method from Machine Scope

**Files:**
- Modify: `scope/machine.go:172-175`

- [ ] **Step 1: Remove HasFailed method**

In `scope/machine.go`, remove the `HasFailed` method (lines 172-175):

```go
// HasFailed checks if the IonosCloudMachine has a failure reason or message.
func (m *Machine) HasFailed() bool {
	return m.IonosMachine.Status.FailureReason != nil || m.IonosMachine.Status.FailureMessage != nil
}
```

- [ ] **Step 2: Commit**

```bash
git add scope/machine.go
git commit -m "refactor(scope): remove HasFailed method (deprecated fields removed)"
```

---

### Task 13: Update Tests

**Files:**
- Modify: `api/v1alpha1/ionoscloudmachine_types_test.go`
- Modify: `scope/machine_test.go`

- [ ] **Step 1: Remove FailureReason/FailureMessage test references in types test**

In `api/v1alpha1/ionoscloudmachine_types_test.go`, remove any test cases that set or assert on `FailureReason` or `FailureMessage`. Search for these fields and remove the relevant lines/blocks. Also remove the `"sigs.k8s.io/cluster-api/errors"` import if it's no longer used.

- [ ] **Step 2: Remove HasFailed tests in scope test**

In `scope/machine_test.go`, remove any test cases for the `HasFailed()` method (if they exist — they may have already been removed based on the PR #352 diff showing 16 deletions in this file).

- [ ] **Step 3: Commit**

```bash
git add api/v1alpha1/ionoscloudmachine_types_test.go scope/machine_test.go
git commit -m "test: remove tests for deprecated FailureReason/FailureMessage"
```

---

### Task 14: Regenerate for PR 2

**Files:**
- Modified by generation: `api/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases/infrastructure.cluster.x-k8s.io_ionoscloudmachines.yaml`

- [ ] **Step 1: Regenerate deepcopy and CRDs**

```bash
make generate && make manifests
```

Expected: `zz_generated.deepcopy.go` will lose the `FailureReason` deep copy code. The machine CRD YAML will lose the `failureReason` and `failureMessage` fields from the status schema.

- [ ] **Step 2: Run lint-fix**

```bash
make lint-fix
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: regenerate deepcopy and CRDs after FailureReason removal"
```

---

### Task 15: Verify PR 2

- [ ] **Step 1: Run all tests**

```bash
make test
```

Expected: All tests pass.

- [ ] **Step 2: Run linter**

```bash
make lint
```

Expected: No lint errors.

- [ ] **Step 3: Run verify**

```bash
make verify
```

Expected: Generated files are up-to-date.
