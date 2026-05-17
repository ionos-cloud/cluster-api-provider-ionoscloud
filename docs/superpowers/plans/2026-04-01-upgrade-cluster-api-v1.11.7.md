# Upgrade Cluster API to v1.11.7 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade `sigs.k8s.io/cluster-api` from v1.10.10 to v1.11.7, applying all required import-path reorganization, API method renames, patch-option renames, and e2e config updates that accompany the CAPI v1.11 release.

**Architecture:** CAPI v1.11 reorganized its Go module layout (all APIs now live under `api/<subsystem>/`), promoted v1beta2 conditions to be the default conditions package, and renamed the patch-helper condition-ownership options. The upgrade consists of six mechanical phases: (1) bump go.mod, (2) update import paths across all `.go` files, (3) rename condition getter/setter methods on provider types, (4) rename patch-option structs in scope files, (5) update e2e config, and (6) regenerate and verify.

**Tech Stack:** Go 1.25, cluster-api v1.11.7, controller-runtime v0.21.0, k8s.io v0.33.3, Ginkgo/Gomega for e2e tests.

---

## File Map

| File | Change |
|---|---|
| `go.mod` | Bump cluster-api, cluster-api/test, controller-runtime, k8s.io/* |
| `go.sum` | Auto-updated by `make tidy` |
| `api/v1alpha1/ionoscloudcluster_types.go` | Rename condition methods; update import |
| `api/v1alpha1/ionoscloudmachine_types.go` | Rename condition methods; update import |
| `api/v1alpha1/ionoscloudmachinetemplate_types.go` | Update import |
| `api/v1alpha1/*_test.go` files | Update import |
| `scope/cluster.go` | Update imports; rename patch options |
| `scope/machine.go` | Update imports; rename patch options |
| `internal/controller/ionoscloudcluster_controller.go` | Update imports |
| `internal/controller/ionoscloudmachine_controller.go` | Update imports |
| `internal/service/cloud/server.go` | Update import |
| `internal/service/k8s/ipam.go` | Update import (IPAM moved) |
| `cmd/main.go` | Update imports (IPAM moved) |
| `test/e2e/suite_test.go` | Update imports |
| `test/e2e/helpers/ownerreference.go` | Update imports |
| `test/e2e/helpers/finalizers.go` | No import change needed (addons path unchanged) |
| `test/e2e/config/ionoscloud.yaml` | Update URLs; version names; metadata paths |
| `test/e2e/data/shared/v1.10/metadata.yaml` | Create new file |
| `api/v1alpha1/zz_generated.deepcopy.go` | Regenerated |
| `config/` manifests | Regenerated |
| `internal/ionoscloud/clienttest/` mocks | Regenerated |

---

## Import Path Migration Reference

Use this table throughout the tasks below:

| Old import path | New import path | Alias |
|---|---|---|
| `sigs.k8s.io/cluster-api/api/v1beta1` | `sigs.k8s.io/cluster-api/api/core/v1beta1` | `clusterv1` |
| `sigs.k8s.io/cluster-api/util/conditions/v1beta2` | `sigs.k8s.io/cluster-api/util/conditions` | `conditions` |
| `sigs.k8s.io/cluster-api/util/conditions` *(old v1beta1)* | `sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1` | `conditions` |
| `sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1` | `sigs.k8s.io/cluster-api/api/ipam/v1beta1` | `ipamv1` |
| `sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1` | `sigs.k8s.io/cluster-api/api/bootstrap/kubeadm/v1beta1` | `bootstrapv1` |
| `sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1` | `sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1` | `controlplanev1` |
| `sigs.k8s.io/cluster-api/api/addons/v1beta1` | unchanged | `addonsv1` |

---

### Task 1: Bump cluster-api, controller-runtime, and k8s.io in go.mod

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Edit the direct-dependency versions in go.mod**

In `go.mod`, change these lines in the `require (...)` block:

```
# Before
sigs.k8s.io/cluster-api v1.10.10
sigs.k8s.io/cluster-api/test v1.10.10
sigs.k8s.io/controller-runtime v0.20.4
k8s.io/api v0.32.3
k8s.io/apimachinery v0.32.3
k8s.io/client-go v0.32.3

# After
sigs.k8s.io/cluster-api v1.11.7
sigs.k8s.io/cluster-api/test v1.11.7
sigs.k8s.io/controller-runtime v0.21.0
k8s.io/api v0.33.3
k8s.io/apimachinery v0.33.3
k8s.io/client-go v0.33.3
```

- [ ] **Step 2: Update indirect k8s.io transitive deps in go.mod**

In the `require (...)` indirect block, update the pinned k8s.io entries to v0.33.3:

```
# Before
k8s.io/apiextensions-apiserver v0.32.3 // indirect
k8s.io/apiserver v0.32.3 // indirect
k8s.io/cluster-bootstrap v0.32.3 // indirect
k8s.io/component-base v0.32.3 // indirect

# After
k8s.io/apiextensions-apiserver v0.33.3 // indirect
k8s.io/apiserver v0.33.3 // indirect
k8s.io/cluster-bootstrap v0.33.3 // indirect
k8s.io/component-base v0.33.3 // indirect
```

- [ ] **Step 3: Run make tidy**

```bash
make tidy
```

Expected: exits 0, `go.sum` updated. If it exits non-zero and reports conflicting k8s.io pins, ensure all `k8s.io/*` entries in go.mod are consistently set to v0.33.3.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: bump cluster-api to v1.11.7, controller-runtime to v0.21.0, k8s to v0.33.3"
```

---

### Task 2: Update core API import path (`api/v1beta1` → `api/core/v1beta1`)

**Files:**
- Modify: all `.go` files that import `sigs.k8s.io/cluster-api/api/v1beta1`

This is a mechanical string substitution. In v1.11, the core API moved from the top-level `api/v1beta1` to `api/core/v1beta1`. The alias `clusterv1` does not change.

- [ ] **Step 1: Confirm affected files**

```bash
grep -rl '"sigs.k8s.io/cluster-api/api/v1beta1"' . --include='*.go'
```

Expected output (approximately 25 files including source and test files):
```
./api/v1alpha1/ionoscloudcluster_types.go
./api/v1alpha1/ionoscloudmachine_types.go
./api/v1alpha1/ionoscloudmachinetemplate_types.go
./scope/cluster.go
./scope/machine.go
./internal/controller/ionoscloudcluster_controller.go
./internal/controller/ionoscloudmachine_controller.go
./internal/service/k8s/ipam.go
./cmd/main.go
./test/e2e/suite_test.go
./test/e2e/helpers/ownerreference.go
... (and test files)
```

- [ ] **Step 2: Apply the substitution**

```bash
find . -name '*.go' -exec sed -i 's|"sigs.k8s.io/cluster-api/api/v1beta1"|"sigs.k8s.io/cluster-api/api/core/v1beta1"|g' {} +
```

- [ ] **Step 3: Also fix the aliased form**

```bash
find . -name '*.go' -exec sed -i 's|clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"|clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"|g' {} +
```

- [ ] **Step 4: Verify no old path remains**

```bash
grep -r '"sigs.k8s.io/cluster-api/api/v1beta1"' . --include='*.go'
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: update core CAPI import path api/v1beta1 → api/core/v1beta1"
```

---

### Task 3: Update conditions import paths

**Files:**
- Modify: `scope/cluster.go`, `scope/machine.go`, `internal/controller/ionoscloudcluster_controller.go`, `internal/controller/ionoscloudmachine_controller.go`, `internal/service/cloud/server.go`
- Modify: `api/v1alpha1/ionoscloudcluster_types_test.go`, `api/v1alpha1/ionoscloudmachine_types_test.go`

Two changes are needed:

**A) The v1beta2 conditions package (the one we use in scopes and controllers) moved from `util/conditions/v1beta2` → `util/conditions`.**

**B) The old v1beta1 conditions package (used only in type tests, for `MarkTrue`/`IsTrue`) moved from `util/conditions` → `util/conditions/deprecated/v1beta1`.**

- [ ] **Step 1: Update v1beta2 → main conditions package (scope, controllers, server)**

```bash
find . -name '*.go' -exec sed -i 's|conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"|conditions "sigs.k8s.io/cluster-api/util/conditions"|g' {} +
```

- [ ] **Step 2: Verify no v1beta2 import remains**

```bash
grep -r 'util/conditions/v1beta2' . --include='*.go'
```

Expected: no output.

- [ ] **Step 3: Update old v1beta1 conditions package in test files**

The test files `api/v1alpha1/ionoscloudcluster_types_test.go` and `api/v1alpha1/ionoscloudmachine_types_test.go` import the bare `util/conditions` package (the old v1beta1 package — it has `MarkTrue`, `IsTrue` etc. for `clusterv1.Conditions`). That package moved to `util/conditions/deprecated/v1beta1`.

Check the exact import form in each test file:

```bash
grep '"sigs.k8s.io/cluster-api/util/conditions"' \
  api/v1alpha1/ionoscloudcluster_types_test.go \
  api/v1alpha1/ionoscloudmachine_types_test.go
```

Expected: both files show an unaliased import of `"sigs.k8s.io/cluster-api/util/conditions"`.

In `api/v1alpha1/ionoscloudcluster_types_test.go`, replace:
```go
"sigs.k8s.io/cluster-api/util/conditions"
```
with:
```go
conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
```

In `api/v1alpha1/ionoscloudmachine_types_test.go`, replace:
```go
"sigs.k8s.io/cluster-api/util/conditions"
```
with:
```go
conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
```

- [ ] **Step 4: Verify**

```bash
grep -r '"sigs.k8s.io/cluster-api/util/conditions"' . --include='*.go'
```

Expected: only the new `util/conditions` entries added in Step 1 (scope/controller files). No bare old v1beta1 usage remains.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: update conditions import paths for CAPI v1.11"
```

---

### Task 4: Update IPAM and bootstrap/controlplane import paths

**Files:**
- Modify: `internal/service/k8s/ipam.go`, `cmd/main.go`, `test/e2e/suite_test.go` (IPAM)
- Modify: `test/e2e/helpers/ownerreference.go` (bootstrap + controlplane)

In v1.11, the IPAM API moved from `exp/ipam/api/v1beta1` to `api/ipam/v1beta1`. The bootstrap and controlplane KubeadmConfig APIs moved from `bootstrap/kubeadm/api/v1beta1` and `controlplane/kubeadm/api/v1beta1` to `api/bootstrap/kubeadm/v1beta1` and `api/controlplane/kubeadm/v1beta1` respectively.

- [ ] **Step 1: Update IPAM import**

```bash
find . -name '*.go' -exec sed -i 's|ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"|ipamv1 "sigs.k8s.io/cluster-api/api/ipam/v1beta1"|g' {} +
```

Also update the unaliased form if it appears:

```bash
find . -name '*.go' -exec sed -i 's|"sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"|"sigs.k8s.io/cluster-api/api/ipam/v1beta1"|g' {} +
```

- [ ] **Step 2: Update bootstrap provider import**

```bash
find . -name '*.go' -exec sed -i 's|bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"|bootstrapv1 "sigs.k8s.io/cluster-api/api/bootstrap/kubeadm/v1beta1"|g' {} +
```

- [ ] **Step 3: Update controlplane provider import**

```bash
find . -name '*.go' -exec sed -i 's|controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"|controlplanev1 "sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1"|g' {} +
```

- [ ] **Step 4: Verify no old paths remain**

```bash
grep -r 'exp/ipam/api/v1beta1\|bootstrap/kubeadm/api/v1beta1\|controlplane/kubeadm/api/v1beta1' . --include='*.go'
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: update IPAM and bootstrap/controlplane import paths for CAPI v1.11"
```

---

### Task 5: Rename condition getter/setter methods on provider types

**Files:**
- Modify: `api/v1alpha1/ionoscloudcluster_types.go`
- Modify: `api/v1alpha1/ionoscloudmachine_types.go`

In v1.11, the `util/conditions` package (the promoted v1beta2 conditions package) expects objects to implement:
```go
GetConditions() []metav1.Condition  // Getter interface
SetConditions([]metav1.Condition)   // Setter interface
```

The deprecated `util/conditions/deprecated/v1beta1` package now expects:
```go
GetV1Beta1Conditions() clusterv1.Conditions  // Getter interface
```

Our types currently implement the v1.10 interface names, which must be renamed as follows:

| Old method | New method |
|---|---|
| `GetConditions() clusterv1.Conditions` | `GetV1Beta1Conditions() clusterv1.Conditions` |
| `SetConditions(conditions clusterv1.Conditions)` | `SetV1Beta1Conditions(conditions clusterv1.Conditions)` |
| `GetV1Beta2Conditions() []metav1.Condition` | `GetConditions() []metav1.Condition` |
| `SetV1Beta2Conditions(conditions []metav1.Condition)` | `SetConditions(conditions []metav1.Condition)` |

- [ ] **Step 1: Update ionoscloudcluster_types.go**

Open `api/v1alpha1/ionoscloudcluster_types.go`. Find the four condition methods at the bottom of the file (currently at approximately lines 130–154) and replace them:

```go
// Before
func (i *IonosCloudCluster) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *IonosCloudCluster) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

func (i *IonosCloudCluster) GetV1Beta2Conditions() []metav1.Condition {
	if i.Status.V1Beta2 == nil {
		return nil
	}
	return i.Status.V1Beta2.Conditions
}

func (i *IonosCloudCluster) SetV1Beta2Conditions(conditions []metav1.Condition) {
	if i.Status.V1Beta2 == nil {
		i.Status.V1Beta2 = &IonosCloudClusterV1Beta2Status{}
	}
	i.Status.V1Beta2.Conditions = conditions
}
```

```go
// After
func (i *IonosCloudCluster) GetV1Beta1Conditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *IonosCloudCluster) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

func (i *IonosCloudCluster) GetConditions() []metav1.Condition {
	if i.Status.V1Beta2 == nil {
		return nil
	}
	return i.Status.V1Beta2.Conditions
}

func (i *IonosCloudCluster) SetConditions(conditions []metav1.Condition) {
	if i.Status.V1Beta2 == nil {
		i.Status.V1Beta2 = &IonosCloudClusterV1Beta2Status{}
	}
	i.Status.V1Beta2.Conditions = conditions
}
```

- [ ] **Step 2: Update ionoscloudmachine_types.go**

Open `api/v1alpha1/ionoscloudmachine_types.go`. Find the four condition methods (currently at approximately lines 383–407) and apply the same renames:

```go
// Before
func (i *IonosCloudMachine) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *IonosCloudMachine) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

func (i *IonosCloudMachine) GetV1Beta2Conditions() []metav1.Condition {
	if i.Status.V1Beta2 == nil {
		return nil
	}
	return i.Status.V1Beta2.Conditions
}

func (i *IonosCloudMachine) SetV1Beta2Conditions(conditions []metav1.Condition) {
	if i.Status.V1Beta2 == nil {
		i.Status.V1Beta2 = &IonosCloudMachineV1Beta2Status{}
	}
	i.Status.V1Beta2.Conditions = conditions
}
```

```go
// After
func (i *IonosCloudMachine) GetV1Beta1Conditions() clusterv1.Conditions {
	return i.Status.Conditions
}

func (i *IonosCloudMachine) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

func (i *IonosCloudMachine) GetConditions() []metav1.Condition {
	if i.Status.V1Beta2 == nil {
		return nil
	}
	return i.Status.V1Beta2.Conditions
}

func (i *IonosCloudMachine) SetConditions(conditions []metav1.Condition) {
	if i.Status.V1Beta2 == nil {
		i.Status.V1Beta2 = &IonosCloudMachineV1Beta2Status{}
	}
	i.Status.V1Beta2.Conditions = conditions
}
```

- [ ] **Step 3: Update any callers of the renamed methods**

Search for call sites of the old method names:

```bash
grep -rn 'GetV1Beta2Conditions\|SetV1Beta2Conditions\|\.GetConditions()\|\.SetConditions(' . --include='*.go'
```

For every occurrence of `GetV1Beta2Conditions` → rename to `GetConditions`.
For every occurrence of `SetV1Beta2Conditions` → rename to `SetConditions`.
For every occurrence of `.GetConditions()` that was previously returning `clusterv1.Conditions` (found in test files or deprecated/v1beta1 usage) → rename to `.GetV1Beta1Conditions()`.
For every occurrence of `.SetConditions(` passing `clusterv1.Conditions` → rename to `.SetV1Beta1Conditions(`.

In practice, direct callers of these methods are almost exclusively the CAPI framework itself (via interfaces), so there should be very few explicit call sites in our codebase. The test files that called `conditions.MarkTrue` (which internally uses the interface) were already updated in Task 3 to use `deprecated/v1beta1`.

- [ ] **Step 4: Verify the build compiles**

```bash
make build
```

Expected: exits 0. If there are errors about interface not satisfied, re-check the method signatures above — the `clusterv1.Conditions` return type in `GetV1Beta1Conditions` must not be confused with `[]metav1.Condition`.

- [ ] **Step 5: Commit**

```bash
git add api/v1alpha1/ionoscloudcluster_types.go api/v1alpha1/ionoscloudmachine_types.go
git commit -m "fix: rename condition getter/setter methods for CAPI v1.11 interface contract"
```

---

### Task 6: Rename patch-option structs in scope files

**Files:**
- Modify: `scope/cluster.go` (PatchObject method, ~lines 221–231)
- Modify: `scope/machine.go` (PatchObject method, ~lines 189–200)

In v1.11, `util/patch` renamed the condition-ownership option types:

| Old struct | New struct | Field type |
|---|---|---|
| `patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{...}}` | `patch.WithOwnedV1Beta1Conditions{Conditions: []clusterv1.ConditionType{...}}` | v1beta1 conditions |
| `patch.WithOwnedV1Beta2Conditions{Conditions: []string{...}}` | `patch.WithOwnedConditions{Conditions: []string{...}}` | metav1 conditions |

- [ ] **Step 1: Update scope/cluster.go PatchObject**

Find the `PatchObject` method in `scope/cluster.go`. The current call to `c.patchHelper.Patch(...)` looks like:

```go
return c.patchHelper.Patch(timeoutCtx, c.IonosCluster,
    patch.WithOwnedConditions{
        Conditions: []clusterv1.ConditionType{clusterv1.ReadyCondition},
    },
    patch.WithOwnedV1Beta2Conditions{
        Conditions: []string{
            string(clusterv1.ReadyCondition),
            string(infrav1.IonosCloudClusterReady),
        },
    },
)
```

Replace with:

```go
return c.patchHelper.Patch(timeoutCtx, c.IonosCluster,
    patch.WithOwnedV1Beta1Conditions{
        Conditions: []clusterv1.ConditionType{clusterv1.ReadyCondition},
    },
    patch.WithOwnedConditions{
        Conditions: []string{
            string(clusterv1.ReadyCondition),
            string(infrav1.IonosCloudClusterReady),
        },
    },
)
```

- [ ] **Step 2: Update scope/machine.go PatchObject**

Find the `PatchObject` method in `scope/machine.go`. The current call looks like:

```go
return m.patchHelper.Patch(
    timeoutCtx,
    m.IonosMachine,
    patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
        clusterv1.ReadyCondition,
        infrav1.MachineProvisionedCondition,
    }},
    patch.WithOwnedV1Beta2Conditions{Conditions: []string{
        string(clusterv1.ReadyCondition),
        string(infrav1.MachineProvisionedCondition),
    }},
)
```

Replace with:

```go
return m.patchHelper.Patch(
    timeoutCtx,
    m.IonosMachine,
    patch.WithOwnedV1Beta1Conditions{Conditions: []clusterv1.ConditionType{
        clusterv1.ReadyCondition,
        infrav1.MachineProvisionedCondition,
    }},
    patch.WithOwnedConditions{Conditions: []string{
        string(clusterv1.ReadyCondition),
        string(infrav1.MachineProvisionedCondition),
    }},
)
```

- [ ] **Step 3: Verify build passes**

```bash
make build
```

Expected: exits 0.

- [ ] **Step 4: Run unit tests**

```bash
make unit-test
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add scope/cluster.go scope/machine.go
git commit -m "fix: rename patch option structs for CAPI v1.11 (WithOwnedConditions → WithOwnedV1Beta1Conditions)"
```

---

### Task 7: Fix any remaining compilation errors

**Files:**
- Modify: any `.go` file that fails to compile

After the mechanical changes above, there may be residual compilation errors from controller-runtime v0.21.0 or other k8s library changes.

- [ ] **Step 1: Attempt a full build**

```bash
make build
```

Expected: exits 0. If not, proceed to Step 2.

- [ ] **Step 2: Triage each compilation error**

For each error, apply the minimal fix. Common breakage patterns for this version delta:

**controller-runtime v0.20 → v0.21 known changes:**
- If you see `undefined: ctrl.Options.LeaderElectionConfig` or similar field changes, check the controller-runtime v0.21 changelog for field renames and update `cmd/main.go`.
- If `builder.WithPredicates(...)` or `handler.EnqueueRequestsFromMapFunc` signatures changed, adapt accordingly.

**k8s.io v0.32 → v0.33 known changes:**
- Generally backward-compatible; no expected breakage in provider code.

**cluster-api v1.10 → v1.11 known changes:**
- `Cluster.GetIPFamily()` was removed (it was deprecated since v1.8). If anywhere in our code calls `.GetIPFamily()`, remove the call entirely and derive the IP family from the cluster network settings directly.
- Index helpers `index.ByClusterClassName` / `index.ClusterByClusterClassName` were renamed to `index.ByClusterClassRef` / `index.ClusterByClusterClassRef`. These are only relevant if the provider registers those indexes, which this provider does not (it does not use ClusterClass). If these appear, update to the new names.

- [ ] **Step 3: After fixing all errors, verify build passes**

```bash
make build
```

Expected: exits 0 with no errors.

- [ ] **Step 4: Run unit tests**

```bash
make unit-test
```

Expected: all tests pass.

- [ ] **Step 5: Commit compilation fixes if any**

```bash
git add -p
git commit -m "fix: resolve compilation errors after cluster-api v1.11 / controller-runtime v0.21 upgrade"
```

---

### Task 8: Update e2e clusterctl config to reference v1.11.7

**Files:**
- Modify: `test/e2e/config/ionoscloud.yaml`

The CAPI component download URLs must point to v1.11.7. The version name in the config represents the "old version" used in upgrade tests; update from `v1.9.0` to `v1.10.0`. The metadata source path moves from `v1.9` to `v1.10`.

- [ ] **Step 1: Update CoreProvider entry**

In `test/e2e/config/ionoscloud.yaml`, find the `CoreProvider` block and apply:

```yaml
# Before
- name: "v1.9.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/core-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.9/metadata.yaml"

# After
- name: "v1.10.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.7/core-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.10/metadata.yaml"
```

- [ ] **Step 2: Update BootstrapProvider entry**

```yaml
# Before
- name: "v1.9.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/bootstrap-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.9/metadata.yaml"

# After
- name: "v1.10.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.7/bootstrap-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.10/metadata.yaml"
```

- [ ] **Step 3: Update ControlPlaneProvider entry**

```yaml
# Before
- name: "v1.9.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/control-plane-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.9/metadata.yaml"

# After
- name: "v1.10.0"
  value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.7/control-plane-components.yaml"
  type: url
  contract: v1beta1
  replacements:
    - old: --metrics-addr=127.0.0.1:8080
      new: --metrics-addr=:8443
  files:
    - sourcePath: "../data/shared/v1.10/metadata.yaml"
```

- [ ] **Step 4: Commit**

```bash
git add test/e2e/config/ionoscloud.yaml
git commit -m "chore(e2e): update CAPI component URLs to v1.11.7"
```

---

### Task 9: Create CAPI v1.10 shared metadata file for e2e

**Files:**
- Create: `test/e2e/data/shared/v1.10/metadata.yaml`

This file tells clusterctl which CAPI release series are compatible when the e2e test downloads CAPI v1.10.0 components during upgrade tests. It is referenced by the `sourcePath` entries updated in Task 8.

- [ ] **Step 1: Create the directory and metadata file**

Create `test/e2e/data/shared/v1.10/metadata.yaml` with this content:

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

- [ ] **Step 2: Commit**

```bash
git add test/e2e/data/shared/v1.10/metadata.yaml
git commit -m "chore(e2e): add CAPI v1.10 shared metadata for upgrade tests"
```

---

### Task 10: Regenerate generated code and mocks

**Files:**
- Modify: `api/v1alpha1/zz_generated.deepcopy.go` (auto-generated)
- Modify: CRD/RBAC manifests under `config/` (auto-generated)
- Modify: mocks under `internal/ionoscloud/clienttest/` (auto-generated)

After renaming methods and updating imports, the generated DeepCopy file and CRD manifests must be regenerated to reflect the updated types.

- [ ] **Step 1: Regenerate DeepCopy methods and CRD manifests**

```bash
make generate
make manifests
```

Expected: both exit 0.

- [ ] **Step 2: Regenerate mocks**

```bash
make mocks
```

Expected: exits 0.

- [ ] **Step 3: Check for unexpected diff**

```bash
git diff --stat
```

Review the diff. Changes to `zz_generated.deepcopy.go`, `config/` manifests, and `internal/ionoscloud/clienttest/` are expected. Changes to non-generated files are a warning sign — investigate before committing.

- [ ] **Step 4: Commit regenerated files**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/ internal/ionoscloud/clienttest/
git commit -m "chore: regenerate mocks and manifests after cluster-api v1.11 upgrade"
```

---

### Task 11: Final verification

- [ ] **Step 1: Run full test suite**

```bash
make test
```

Expected: all unit and integration tests pass.

- [ ] **Step 2: Run linter**

```bash
make lint
```

Expected: exits 0. If there are formatting issues, run `make lint-fix` first, then `make lint`.

- [ ] **Step 3: Verify generated files are up-to-date (CI gate)**

```bash
make verify
```

Expected: exits 0. If it fails with "generated files are not up-to-date", re-run the relevant `make generate`/`make manifests`/`make mocks` target and commit.

- [ ] **Step 4: Final commit if lint-fix produced changes**

```bash
git add -p
git commit -m "style: lint-fix after cluster-api v1.11.7 upgrade"
```
