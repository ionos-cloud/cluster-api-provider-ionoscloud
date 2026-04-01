# Cluster API v1.9.11 Upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade `sigs.k8s.io/cluster-api` from v1.8.12 to v1.9.11, adopting the v1beta2 conditions API and removing deprecated packages.

**Architecture:** Bump six direct Go dependencies, add v1beta2 conditions support to both API types (nested `V1Beta2` status struct + interface methods), replace all v1beta1 `util/conditions` call sites with `util/conditions/v1beta2`, fix predicate signatures, remove the deprecated `errors` package usage, and update e2e config URLs.

**Tech Stack:** Go 1.24, `sigs.k8s.io/cluster-api` v1.9.11, `sigs.k8s.io/controller-runtime` v0.19.6, `k8s.io` v0.31.3, `kubebuilder` markers, `golangci-lint`, Ginkgo/Gomega.

---

## File Map

| File | Change |
|---|---|
| `go.mod` | Bump 6 direct deps |
| `.golangci.yml` | Add `importas` alias for `util/conditions/v1beta2` |
| `api/v1alpha1/ionoscloudcluster_types.go` | Add `IonosCloudClusterV1Beta2Status`, `V1Beta2` field, `GetV1Beta2Conditions`/`SetV1Beta2Conditions` |
| `api/v1alpha1/ionoscloudmachine_types.go` | Same + remove `FailureReason`/`FailureMessage`/`errors` import |
| `api/v1alpha1/zz_generated.deepcopy.go` | Regenerated — do not edit manually |
| `config/crd/bases/` (CRD YAMLs) | Regenerated — do not edit manually |
| `scope/cluster.go` | Replace `SetSummary` → `SetSummaryCondition`, add `WithOwnedV1Beta2Conditions` to patch |
| `scope/machine.go` | Same + remove `HasFailed` |
| `internal/controller/ionoscloudcluster_controller.go` | Fix predicate signatures, replace `MarkTrue` |
| `internal/controller/ionoscloudmachine_controller.go` | Replace `MarkFalse` calls, remove `HasFailed` guard, drop `ConditionSeverityInfo` |
| `internal/service/cloud/server.go` | Replace `MarkTrue` |
| `test/e2e/config/ionoscloud.yaml` | Update 3 CAPI component URLs from v1.8.1 → v1.9.11 |

---

## Task 1: Bump Go module dependencies

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Update the six direct dependencies in `go.mod`**

Replace the existing entries in the `require` block:

```
sigs.k8s.io/cluster-api v1.8.12        →  sigs.k8s.io/cluster-api v1.9.11
sigs.k8s.io/cluster-api/test v1.8.12   →  sigs.k8s.io/cluster-api/test v1.9.11
sigs.k8s.io/controller-runtime v0.18.7 →  sigs.k8s.io/controller-runtime v0.19.6
k8s.io/api v0.30.14                    →  k8s.io/api v0.31.3
k8s.io/apimachinery v0.30.14           →  k8s.io/apimachinery v0.31.3
k8s.io/client-go v0.30.14             →  k8s.io/client-go v0.31.3
```

- [ ] **Step 2: Run `go mod tidy` to resolve transitive deps**

```bash
go mod tidy
```

Expected: exits 0; `go.sum` updated; indirect k8s.io packages (`apiextensions-apiserver`, `apiserver`, `cluster-bootstrap`, `component-base`) bump to v0.31.x automatically.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: bump cluster-api to v1.9.11 and controller-runtime to v0.19.6"
```

---

## Task 2: Add linter alias for `util/conditions/v1beta2`

**Files:**
- Modify: `.golangci.yml`

The `importas` linter enforces aliases for all listed packages. Since we will import `sigs.k8s.io/cluster-api/util/conditions/v1beta2` with the alias `conditions` (replacing the old v1beta1 package), we must register it.

- [ ] **Step 1: Add the alias entry to `.golangci.yml`**

Find the `# Cluster API` section in the `importas.alias` list (currently ends with `capie2e`). Add one line after the `capie2e` entry:

```yaml
      - pkg: "sigs.k8s.io/cluster-api/test/e2e"
        alias: capie2e
      - pkg: "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
        alias: conditions
```

- [ ] **Step 2: Commit**

```bash
git add .golangci.yml
git commit -m "chore: add importas alias for cluster-api util/conditions/v1beta2"
```

---

## Task 3: Add v1beta2 conditions to `IonosCloudCluster`

**Files:**
- Modify: `api/v1alpha1/ionoscloudcluster_types.go`

- [ ] **Step 1: Add the `IonosCloudClusterV1Beta2Status` struct**

After the closing brace of `IonosCloudClusterStatus` (around line 86), add:

```go
// IonosCloudClusterV1Beta2Status groups all status fields that will be used when the CAPI contract moves to v1beta2.
type IonosCloudClusterV1Beta2Status struct {
	// Conditions represents the observations of the current state of the IonosCloudCluster.
	//+optional
	//+listType=map
	//+listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 2: Add the `V1Beta2` pointer field to `IonosCloudClusterStatus`**

Inside `IonosCloudClusterStatus`, after the `CurrentClusterRequest` field (around line 81), add:

```go
	// V1Beta2 groups all status fields that will be used when the CAPI contract moves to v1beta2.
	//+optional
	V1Beta2 *IonosCloudClusterV1Beta2Status `json:"v1beta2,omitempty"`
```

- [ ] **Step 3: Add `GetV1Beta2Conditions` and `SetV1Beta2Conditions` methods**

After the existing `SetConditions` method (around line 123), add:

```go
// GetV1Beta2Conditions returns the v1beta2 conditions from the status.
func (i *IonosCloudCluster) GetV1Beta2Conditions() []metav1.Condition {
	if i.Status.V1Beta2 == nil {
		return nil
	}
	return i.Status.V1Beta2.Conditions
}

// SetV1Beta2Conditions sets the v1beta2 conditions in the status.
func (i *IonosCloudCluster) SetV1Beta2Conditions(conditions []metav1.Condition) {
	if i.Status.V1Beta2 == nil {
		i.Status.V1Beta2 = &IonosCloudClusterV1Beta2Status{}
	}
	i.Status.V1Beta2.Conditions = conditions
}
```

- [ ] **Step 4: Verify the file compiles**

```bash
go build ./api/v1alpha1/...
```

Expected: exits 0, no output.

- [ ] **Step 5: Commit**

```bash
git add api/v1alpha1/ionoscloudcluster_types.go
git commit -m "feat(api): add v1beta2 conditions support to IonosCloudCluster"
```

---

## Task 4: Update `IonosCloudMachine` API type

**Files:**
- Modify: `api/v1alpha1/ionoscloudmachine_types.go`

This task removes the deprecated `errors` package usage and adds v1beta2 conditions.

- [ ] **Step 1: Remove `FailureReason` and `FailureMessage` fields from `IonosCloudMachineStatus`**

The fields are around lines 299–335. Remove both fields and their doc comments:

```go
// Remove these two blocks entirely:

	// FailureReason will be set in the event that there is a terminal problem
	// ...
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// ...
	FailureMessage *string `json:"failureMessage,omitempty"`
```

- [ ] **Step 2: Remove the `sigs.k8s.io/cluster-api/errors` import**

In the import block (around line 24), remove:

```go
	"sigs.k8s.io/cluster-api/errors"
```

- [ ] **Step 3: Add the `IonosCloudMachineV1Beta2Status` struct**

After the closing brace of `IonosCloudMachineStatus`, add:

```go
// IonosCloudMachineV1Beta2Status groups all status fields that will be used when the CAPI contract moves to v1beta2.
type IonosCloudMachineV1Beta2Status struct {
	// Conditions represents the observations of the current state of the IonosCloudMachine.
	//+optional
	//+listType=map
	//+listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 4: Add the `V1Beta2` pointer field to `IonosCloudMachineStatus`**

Inside `IonosCloudMachineStatus`, after the `Conditions` field, add:

```go
	// V1Beta2 groups all status fields that will be used when the CAPI contract moves to v1beta2.
	//+optional
	V1Beta2 *IonosCloudMachineV1Beta2Status `json:"v1beta2,omitempty"`
```

- [ ] **Step 5: Add `GetV1Beta2Conditions` and `SetV1Beta2Conditions` methods**

After the existing `SetConditions` method, add:

```go
// GetV1Beta2Conditions returns the v1beta2 conditions from the status.
func (m *IonosCloudMachine) GetV1Beta2Conditions() []metav1.Condition {
	if m.Status.V1Beta2 == nil {
		return nil
	}
	return m.Status.V1Beta2.Conditions
}

// SetV1Beta2Conditions sets the v1beta2 conditions in the status.
func (m *IonosCloudMachine) SetV1Beta2Conditions(conditions []metav1.Condition) {
	if m.Status.V1Beta2 == nil {
		m.Status.V1Beta2 = &IonosCloudMachineV1Beta2Status{}
	}
	m.Status.V1Beta2.Conditions = conditions
}
```

- [ ] **Step 6: Verify the file compiles**

```bash
go build ./api/v1alpha1/...
```

Expected: exits 0.

- [ ] **Step 7: Commit**

```bash
git add api/v1alpha1/ionoscloudmachine_types.go
git commit -m "feat(api): add v1beta2 conditions to IonosCloudMachine, remove deprecated FailureReason/FailureMessage"
```

---

## Task 5: Regenerate CRDs and DeepCopy

**Files:**
- Modify: `api/v1alpha1/zz_generated.deepcopy.go` (auto-generated)
- Modify: `config/crd/bases/*.yaml` (auto-generated)

- [ ] **Step 1: Regenerate DeepCopy methods**

```bash
make generate
```

Expected: exits 0. `api/v1alpha1/zz_generated.deepcopy.go` updated with `DeepCopyInto` for the two new `V1Beta2Status` structs and the updated `IonosCloudMachineStatus` (without `FailureReason`/`FailureMessage`).

- [ ] **Step 2: Regenerate CRD manifests**

```bash
make manifests
```

Expected: exits 0. CRD YAMLs in `config/crd/bases/` updated: `v1beta2` nested object added to status, `failureReason` and `failureMessage` removed.

- [ ] **Step 3: Commit**

```bash
git add api/v1alpha1/zz_generated.deepcopy.go config/crd/
git commit -m "chore: regenerate deepcopy and CRD manifests for v1beta2 conditions"
```

---

## Task 6: Update `scope/cluster.go`

**Files:**
- Modify: `scope/cluster.go`

Replace the v1beta1 `conditions` import and update `PatchObject`.

- [ ] **Step 1: Replace the conditions import**

In the import block, replace:

```go
	"sigs.k8s.io/cluster-api/util/conditions"
```

with:

```go
	conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
```

- [ ] **Step 2: Update `PatchObject` to use v1beta2**

The current `PatchObject` method (around line 204) is:

```go
func (c *Cluster) PatchObject() error {
	// always set the ready condition
	conditions.SetSummary(c.IonosCluster,
		conditions.WithConditions(infrav1.IonosCloudClusterReady))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return c.patchHelper.Patch(timeoutCtx, c.IonosCluster, patch.WithOwnedConditions{
		Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		},
	})
}
```

Replace it with:

```go
func (c *Cluster) PatchObject() error {
	// always set the ready condition summary
	if err := conditions.SetSummaryCondition(
		c.IonosCluster,
		c.IonosCluster,
		string(clusterv1.ReadyCondition),
		conditions.ForConditionTypes{string(infrav1.IonosCloudClusterReady)},
	); err != nil {
		return err
	}

	// NOTE(piepmatz): We don't accept and forward a context here. This is on purpose: Even if a reconciliation is
	//  aborted, we want to make sure that the final patch is applied. Reusing the context from the reconciliation
	//  would cause the patch to be aborted as well.

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return c.patchHelper.Patch(timeoutCtx, c.IonosCluster,
		patch.WithOwnedConditions{
			Conditions: []clusterv1.ConditionType{clusterv1.ReadyCondition},
		},
		patch.WithOwnedV1Beta2Conditions{
			Conditions: []string{string(clusterv1.ReadyCondition)},
		},
	)
}
```

- [ ] **Step 3: Verify the file compiles**

```bash
go build ./scope/...
```

Expected: exits 0.

- [ ] **Step 4: Run existing scope tests**

```bash
go test ./scope/... -v -count=1 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add scope/cluster.go
git commit -m "feat(scope): migrate cluster conditions to v1beta2 API"
```

---

## Task 7: Update `scope/machine.go`

**Files:**
- Modify: `scope/machine.go`

Replace the v1beta1 `conditions` import, update `PatchObject`, and remove the `HasFailed` method (which was always returning false since `SetFailure` was never called).

- [ ] **Step 1: Replace the conditions import**

In the import block, replace:

```go
	"sigs.k8s.io/cluster-api/util/conditions"
```

with:

```go
	conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
```

- [ ] **Step 2: Remove the `HasFailed` method**

Remove the following method (around lines 171–175):

```go
// HasFailed checks if the IonosCloudMachine is in a failed state.
func (m *Machine) HasFailed() bool {
	status := m.IonosMachine.Status
	return status.FailureReason != nil || status.FailureMessage != nil
}
```

- [ ] **Step 3: Update `PatchObject` to use v1beta2**

The current `PatchObject` method (around line 177) is:

```go
func (m *Machine) PatchObject() error {
	conditions.SetSummary(m.IonosMachine,
		conditions.WithConditions(
			infrav1.MachineProvisionedCondition))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// We don't accept and forward a context here. This is on purpose: Even if a reconciliation is
	// aborted, we want to make sure that the final patch is applied. Reusing the context from the reconciliation
	// would cause the patch to be aborted as well.
	return m.patchHelper.Patch(
		timeoutCtx,
		m.IonosMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.MachineProvisionedCondition,
		}})
}
```

Replace it with:

```go
func (m *Machine) PatchObject() error {
	if err := conditions.SetSummaryCondition(
		m.IonosMachine,
		m.IonosMachine,
		string(clusterv1.ReadyCondition),
		conditions.ForConditionTypes{string(infrav1.MachineProvisionedCondition)},
	); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// We don't accept and forward a context here. This is on purpose: Even if a reconciliation is
	// aborted, we want to make sure that the final patch is applied. Reusing the context from the reconciliation
	// would cause the patch to be aborted as well.
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
}
```

- [ ] **Step 4: Verify the file compiles**

```bash
go build ./scope/...
```

Expected: exits 0.

- [ ] **Step 5: Run existing scope tests**

```bash
go test ./scope/... -v -count=1 2>&1 | tail -20
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add scope/machine.go
git commit -m "feat(scope): migrate machine conditions to v1beta2 API, remove HasFailed"
```

---

## Task 8: Update `internal/controller/ionoscloudcluster_controller.go`

**Files:**
- Modify: `internal/controller/ionoscloudcluster_controller.go`

Two changes: fix predicate signatures and replace `MarkTrue` with `conditions.Set`.

- [ ] **Step 1: Replace the conditions import**

In the import block, replace:

```go
	"sigs.k8s.io/cluster-api/util/conditions"
```

with:

```go
	conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
```

- [ ] **Step 2: Fix the predicate signatures**

Find the `SetupWithManager` function. The cluster controller has `scheme *runtime.Scheme` as a struct field.

Replace (around line 264):

```go
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
```

with:

```go
		WithEventFilter(predicates.ResourceNotPaused(r.scheme, ctrl.LoggerFrom(ctx))).
```

Replace (around line 273):

```go
			builder.WithPredicates(predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx))),
```

with:

```go
			builder.WithPredicates(predicates.ClusterUnpaused(r.scheme, ctrl.LoggerFrom(ctx))),
```

- [ ] **Step 3: Replace `conditions.MarkTrue` with `conditions.Set`**

Find the call at around line 175:

```go
	conditions.MarkTrue(clusterScope.IonosCluster, infrav1.IonosCloudClusterReady)
```

Replace with:

```go
	conditions.Set(clusterScope.IonosCluster, metav1.Condition{
		Type:   string(infrav1.IonosCloudClusterReady),
		Status: metav1.ConditionTrue,
		Reason: string(infrav1.IonosCloudClusterReady),
	})
```

Make sure `metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"` is in the import block (it is already, as this file uses metav1 elsewhere).

- [ ] **Step 4: Verify the file compiles**

```bash
go build ./internal/controller/...
```

Expected: exits 0.

- [ ] **Step 5: Run controller unit tests**

```bash
go test ./internal/controller/... -v -count=1 2>&1 | tail -30
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/controller/ionoscloudcluster_controller.go
git commit -m "feat(controller): fix predicate signatures and migrate cluster conditions to v1beta2"
```

---

## Task 9: Update `internal/controller/ionoscloudmachine_controller.go`

**Files:**
- Modify: `internal/controller/ionoscloudmachine_controller.go`

Remove the `HasFailed` guard and replace two `MarkFalse` calls.

- [ ] **Step 1: Replace the conditions import**

In the import block, replace:

```go
	"sigs.k8s.io/cluster-api/util/conditions"
```

with:

```go
	conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
```

- [ ] **Step 2: Remove the `HasFailed` guard**

Find the block at around line 150:

```go
	if machineScope.HasFailed() {
		log.Info("Error state detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}
```

Remove it entirely. `HasFailed` was always returning false since `SetFailure` was never called.

- [ ] **Step 3: Replace the first `MarkFalse` call**

Find the `isInfrastructureReady` function (around line 304). Replace:

```go
		conditions.MarkFalse(
			ms.IonosMachine,
			infrav1.MachineProvisionedCondition,
			infrav1.WaitingForClusterInfrastructureReason,
			clusterv1.ConditionSeverityInfo, "")
```

with:

```go
		conditions.Set(ms.IonosMachine, metav1.Condition{
			Type:   string(infrav1.MachineProvisionedCondition),
			Status: metav1.ConditionFalse,
			Reason: infrav1.WaitingForClusterInfrastructureReason,
		})
```

- [ ] **Step 4: Replace the second `MarkFalse` call**

In the same function, replace:

```go
		conditions.MarkFalse(
			ms.IonosMachine,
			infrav1.MachineProvisionedCondition,
			infrav1.WaitingForBootstrapDataReason,
			clusterv1.ConditionSeverityInfo, "",
		)
```

with:

```go
		conditions.Set(ms.IonosMachine, metav1.Condition{
			Type:   string(infrav1.MachineProvisionedCondition),
			Status: metav1.ConditionFalse,
			Reason: infrav1.WaitingForBootstrapDataReason,
		})
```

- [ ] **Step 5: Remove unused `clusterv1.ConditionSeverityInfo` reference**

Check if `clusterv1` is still used in this file after removing `ConditionSeverityInfo`:

```bash
grep "clusterv1\." internal/controller/ionoscloudmachine_controller.go
```

It will still appear for `&clusterv1.Machine{}` and `*clusterv1.Cluster` in `SetupWithManager`. The import stays — no change needed.

- [ ] **Step 6: Verify the file compiles**

```bash
go build ./internal/controller/...
```

Expected: exits 0.

- [ ] **Step 7: Run controller unit tests**

```bash
go test ./internal/controller/... -v -count=1 2>&1 | tail -30
```

Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/controller/ionoscloudmachine_controller.go
git commit -m "feat(controller): migrate machine conditions to v1beta2, remove dead HasFailed guard"
```

---

## Task 10: Update `internal/service/cloud/server.go`

**Files:**
- Modify: `internal/service/cloud/server.go`

- [ ] **Step 1: Replace the conditions import**

In the import block (around line 32), replace:

```go
	"sigs.k8s.io/cluster-api/util/conditions"
```

with:

```go
	conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
```

- [ ] **Step 2: Replace `conditions.MarkTrue` in `FinalizeMachineProvisioning`**

Find the call at around line 148:

```go
	conditions.MarkTrue(ms.IonosMachine, infrav1.MachineProvisionedCondition)
```

Replace with:

```go
	conditions.Set(ms.IonosMachine, metav1.Condition{
		Type:   string(infrav1.MachineProvisionedCondition),
		Status: metav1.ConditionTrue,
		Reason: string(infrav1.MachineProvisionedCondition),
	})
```

`server.go` does not currently import `metav1`. Add it to the import block:

```go
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
```

(Place it in the second import group, with other `k8s.io` packages, before the `sigs.k8s.io` entries.)

- [ ] **Step 3: Verify the file compiles**

```bash
go build ./internal/service/cloud/...
```

Expected: exits 0.

- [ ] **Step 4: Run cloud service tests**

```bash
go test ./internal/service/cloud/... -v -count=1 2>&1 | tail -30
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cloud/server.go
git commit -m "feat(cloud): migrate server conditions to v1beta2 API"
```

---

## Task 11: Update e2e config

**Files:**
- Modify: `test/e2e/config/ionoscloud.yaml`

- [ ] **Step 1: Update the three CAPI component URLs**

In `test/e2e/config/ionoscloud.yaml`, find and replace all three occurrences of the v1.8.1 GitHub release URL segment:

```
kubernetes-sigs/cluster-api/releases/download/v1.8.1/
```

Replace each with:

```
kubernetes-sigs/cluster-api/releases/download/v1.9.11/
```

This affects three lines:
- `core-components.yaml` URL
- `bootstrap-components.yaml` URL
- `control-plane-components.yaml` URL

- [ ] **Step 2: Commit**

```bash
git add test/e2e/config/ionoscloud.yaml
git commit -m "chore(e2e): update CAPI component URLs to v1.9.11"
```

---

## Task 12: Regenerate mocks, build, and verify

**Files:**
- Modify: generated mock files (if changed)

- [ ] **Step 1: Regenerate mocks**

```bash
make mocks
```

Expected: exits 0. If mock signatures have changed (due to interface updates), the files in `internal/ionoscloud/clienttest/` will be updated.

- [ ] **Step 2: Run full build (compile + lint + vet)**

```bash
make build
```

Expected: exits 0. If there are any remaining compilation errors or lint violations (e.g., an unaliased import, a missing import, or a stale reference), they will surface here.

Common issues to watch for:
- If `make lint` fails with `importas: package X must be imported as Y` — check that all files importing `util/conditions/v1beta2` use the alias `conditions`.
- If `make vet` fails — check for unused imports (e.g., `clusterv1.ConditionSeverityInfo` removed but `clusterv1` still needed).

- [ ] **Step 3: Run all tests**

```bash
make test
```

Expected: exits 0. All unit and integration tests pass.

- [ ] **Step 4: Verify generated files are up-to-date**

```bash
make verify
```

Expected: exits 0. If this fails, it means `make generate` or `make manifests` produced output that doesn't match what's committed — re-run the relevant `make` target and commit the result.

- [ ] **Step 5: Commit any updated mocks**

```bash
git add -u
git status  # confirm only mock files changed, no unexpected modifications
git commit -m "chore: regenerate mocks after cluster-api v1.9.11 upgrade"
```

If no mock files changed, skip this step.
