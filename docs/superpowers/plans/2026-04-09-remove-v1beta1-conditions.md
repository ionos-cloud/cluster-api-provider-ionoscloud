# Remove v1beta1 Conditions Backwards Compatibility

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all old v1beta1 condition support and flatten `Status.V1Beta2.Conditions` into `Status.Conditions` as `[]metav1.Condition`.

**Architecture:** The current dual-write pattern writes conditions to both `Status.Conditions` (old `clusterv1.Conditions`) and `Status.V1Beta2.Conditions` (new `[]metav1.Condition`). We remove the old system entirely: the `Status.Conditions` field changes type from `clusterv1.Conditions` to `[]metav1.Condition`, the `V1Beta2` nesting is removed, all old `v1beta1conditions.*` calls are deleted, and the accessor methods are updated.

**Tech Stack:** Go, controller-runtime, cluster-api v1.9.11, kubebuilder markers

---

### Task 1: Update IonosCloudCluster API types

**Files:**
- Modify: `api/v1alpha1/ionoscloudcluster_types.go`

- [ ] **Step 1: Change condition constant type from `clusterv1.ConditionType` to `string`**

```go
// IonosCloudClusterReady is the condition for the IonosCloudCluster, which indicates that the cluster is ready.
IonosCloudClusterReady = "ClusterReady"
```

Remove the `clusterv1.ConditionType` type annotation — it becomes an untyped string constant.

- [ ] **Step 2: Replace `Status.Conditions` type and remove `V1Beta2` field**

Replace the `IonosCloudClusterStatus` struct's `Conditions` and `V1Beta2` fields:

```go
// IonosCloudClusterStatus defines the observed state of IonosCloudCluster.
type IonosCloudClusterStatus struct {
	// Ready indicates that the cluster is ready.
	//+optional
	Ready bool `json:"ready,omitempty"`

	// Conditions represents the observations of the current state of the IonosCloudCluster.
	//+optional
	//+listType=map
	//+listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// CurrentRequestByDatacenter maps data center IDs to a pending provisioning request made during reconciliation.
	//+optional
	CurrentRequestByDatacenter map[string]ProvisioningRequest `json:"currentRequest,omitempty"`

	// CurrentClusterRequest is the current pending request made during reconciliation for the whole cluster.
	//+optional
	CurrentClusterRequest *ProvisioningRequest `json:"currentClusterRequest,omitempty"`

	// ControlPlaneEndpointIPBlockID is the IONOS Cloud UUID for the control plane endpoint IP block.
	//+optional
	ControlPlaneEndpointIPBlockID string `json:"controlPlaneEndpointIPBlockID,omitempty"`
}
```

- [ ] **Step 3: Delete `IonosCloudClusterV1Beta2Status` struct entirely**

Remove the entire struct definition (lines ~92-99 in current file).

- [ ] **Step 4: Replace accessor methods**

Replace `GetConditions`/`SetConditions` to work with `[]metav1.Condition` and delete `GetV1Beta2Conditions`/`SetV1Beta2Conditions`:

```go
// GetV1Beta2Conditions returns the v1beta2 conditions from the status.
func (i *IonosCloudCluster) GetV1Beta2Conditions() []metav1.Condition {
	return i.Status.Conditions
}

// SetV1Beta2Conditions sets the v1beta2 conditions in the status.
func (i *IonosCloudCluster) SetV1Beta2Conditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}
```

Delete the old `GetConditions` and `SetConditions` methods entirely — they served the old `clusterv1.Conditions` contract.

- [ ] **Step 5: Remove `clusterv1` import if no longer used**

Check if `clusterv1` is still needed for `clusterv1.APIEndpoint` in the spec. It is — keep the import but remove any unused references.

- [ ] **Step 6: Commit**

```bash
git add api/v1alpha1/ionoscloudcluster_types.go
git commit -m "refactor(api): flatten cluster conditions to metav1.Condition, remove v1beta1 compat"
```

---

### Task 2: Update IonosCloudMachine API types

**Files:**
- Modify: `api/v1alpha1/ionoscloudmachine_types.go`

- [ ] **Step 1: Change condition constant type from `clusterv1.ConditionType` to `string`**

```go
// MachineProvisionedCondition documents the status of the provisioning of a IonosCloudMachine and
// the underlying VM.
MachineProvisionedCondition = "MachineProvisioned"
```

Remove the `clusterv1.ConditionType` type annotation.

- [ ] **Step 2: Replace `Status.Conditions` type and remove `V1Beta2` field**

In `IonosCloudMachineStatus`, replace:

```go
// Conditions represents the observations of the current state of the IonosCloudMachine.
//+optional
//+listType=map
//+listMapKey=type
Conditions []metav1.Condition `json:"conditions,omitempty"`
```

Remove the `V1Beta2 *IonosCloudMachineV1Beta2Status` field entirely.

- [ ] **Step 3: Delete `IonosCloudMachineV1Beta2Status` struct entirely**

Remove the entire struct definition (lines ~316-323 in current file).

- [ ] **Step 4: Replace accessor methods**

Replace with v1beta2-only accessors and delete old ones:

```go
// GetV1Beta2Conditions returns the v1beta2 conditions from the status.
func (m *IonosCloudMachine) GetV1Beta2Conditions() []metav1.Condition {
	return m.Status.Conditions
}

// SetV1Beta2Conditions sets the v1beta2 conditions in the status.
func (m *IonosCloudMachine) SetV1Beta2Conditions(conditions []metav1.Condition) {
	m.Status.Conditions = conditions
}
```

Delete the old `GetConditions` and `SetConditions` methods.

- [ ] **Step 5: Remove `clusterv1` import**

After removing `clusterv1.ConditionType`, check if `clusterv1` is still needed in this file. It should not be — remove it.

- [ ] **Step 6: Commit**

```bash
git add api/v1alpha1/ionoscloudmachine_types.go
git commit -m "refactor(api): flatten machine conditions to metav1.Condition, remove v1beta1 compat"
```

---

### Task 3: Update Cluster scope

**Files:**
- Modify: `scope/cluster.go`

- [ ] **Step 1: Remove `v1beta1conditions` import**

Remove this line from imports:
```go
v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions"
```

- [ ] **Step 2: Remove old condition summary in `PatchObject`**

In `PatchObject()`, remove these lines:
```go
// always set the v1beta1 ready condition summary
v1beta1conditions.SetSummary(c.IonosCluster,
    v1beta1conditions.WithConditions(infrav1.IonosCloudClusterReady))
```

Keep only the v1beta2 summary call. Update the `ForConditionTypes` argument since `IonosCloudClusterReady` is now a plain `string` (no cast needed):

```go
// always set the ready condition summary
if err := conditions.SetSummaryCondition(
    c.IonosCluster,
    c.IonosCluster,
    clusterv1.ReadyV1Beta2Condition,
    conditions.ForConditionTypes{infrav1.IonosCloudClusterReady},
); err != nil {
    return err
}
```

- [ ] **Step 3: Remove `patch.WithOwnedConditions` from `Patch` call**

In the `patchHelper.Patch(...)` call, remove:
```go
patch.WithOwnedConditions{
    Conditions: []clusterv1.ConditionType{
        clusterv1.ReadyCondition,
        infrav1.IonosCloudClusterReady,
    },
},
```

Keep only `patch.WithOwnedV1Beta2Conditions`. Since `IonosCloudClusterReady` is now a plain `string`, the `string(...)` cast is no longer needed:

```go
return c.patchHelper.Patch(timeoutCtx, c.IonosCluster,
    patch.WithOwnedV1Beta2Conditions{
        Conditions: []string{
            clusterv1.ReadyV1Beta2Condition,
            infrav1.IonosCloudClusterReady,
        },
    },
)
```

- [ ] **Step 4: Commit**

```bash
git add scope/cluster.go
git commit -m "refactor(scope): remove v1beta1 conditions from cluster scope"
```

---

### Task 4: Update Machine scope

**Files:**
- Modify: `scope/machine.go`

- [ ] **Step 1: Remove `v1beta1conditions` import**

Remove this line from imports:
```go
v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions"
```

- [ ] **Step 2: Remove old condition summary in `PatchObject`**

Remove:
```go
// set the v1beta1 ready condition summary
v1beta1conditions.SetSummary(m.IonosMachine,
    v1beta1conditions.WithConditions(infrav1.MachineProvisionedCondition))
```

Keep only the v1beta2 summary. Update `ForConditionTypes` (no cast needed):

```go
// set the ready condition summary
if err := conditions.SetSummaryCondition(
    m.IonosMachine,
    m.IonosMachine,
    clusterv1.ReadyV1Beta2Condition,
    conditions.ForConditionTypes{infrav1.MachineProvisionedCondition},
); err != nil {
    return err
}
```

- [ ] **Step 3: Remove `patch.WithOwnedConditions` from `Patch` call**

Remove the old `patch.WithOwnedConditions{...}` block. Keep only:

```go
return m.patchHelper.Patch(
    timeoutCtx,
    m.IonosMachine,
    patch.WithOwnedV1Beta2Conditions{Conditions: []string{
        clusterv1.ReadyV1Beta2Condition,
        infrav1.MachineProvisionedCondition,
    }},
)
```

- [ ] **Step 4: Commit**

```bash
git add scope/machine.go
git commit -m "refactor(scope): remove v1beta1 conditions from machine scope"
```

---

### Task 5: Update Cluster controller

**Files:**
- Modify: `internal/controller/ionoscloudcluster_controller.go`

- [ ] **Step 1: Remove `v1beta1conditions` import**

Remove:
```go
v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions"
```

- [ ] **Step 2: Remove old condition call in `reconcileNormal`**

At line ~177, remove:
```go
v1beta1conditions.MarkTrue(clusterScope.IonosCluster, infrav1.IonosCloudClusterReady)
```

Keep only the `conditions.Set(...)` call that follows it.

- [ ] **Step 3: Commit**

```bash
git add internal/controller/ionoscloudcluster_controller.go
git commit -m "refactor(controller): remove v1beta1 conditions from cluster controller"
```

---

### Task 6: Update Machine controller

**Files:**
- Modify: `internal/controller/ionoscloudmachine_controller.go`

- [ ] **Step 1: Remove `v1beta1conditions` import**

Remove:
```go
v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions"
```

- [ ] **Step 2: Remove old condition calls in `isInfrastructureReady`**

Remove both `v1beta1conditions.MarkFalse(...)` calls (lines ~306-308 and ~322-324). Keep only the `conditions.Set(...)` calls that follow each one.

- [ ] **Step 3: Remove `clusterv1` import if no longer used**

After removing the `v1beta1conditions.MarkFalse` calls, `clusterv1.ConditionSeverityInfo` is no longer referenced. Check if `clusterv1` is still used elsewhere in this file (e.g., for `clusterv1.Cluster` type references). If still used, keep the import; otherwise remove it.

- [ ] **Step 4: Commit**

```bash
git add internal/controller/ionoscloudmachine_controller.go
git commit -m "refactor(controller): remove v1beta1 conditions from machine controller"
```

---

### Task 7: Update server service

**Files:**
- Modify: `internal/service/cloud/server.go`

- [ ] **Step 1: Remove `v1beta1conditions` import**

Remove:
```go
v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions"
```

- [ ] **Step 2: Remove old condition call in `FinalizeMachineProvisioning`**

At line ~150, remove:
```go
v1beta1conditions.MarkTrue(ms.IonosMachine, infrav1.MachineProvisionedCondition)
```

Keep only the `conditions.Set(...)` call.

- [ ] **Step 3: Commit**

```bash
git add internal/service/cloud/server.go
git commit -m "refactor(service): remove v1beta1 conditions from server service"
```

---

### Task 8: Update tests

**Files:**
- Modify: `api/v1alpha1/ionoscloudcluster_types_test.go`
- Modify: `api/v1alpha1/ionoscloudmachine_types_test.go`

- [ ] **Step 1: Update cluster condition test**

In `api/v1alpha1/ionoscloudcluster_types_test.go`, replace `TestIonosCloudCluster_Conditions`:

```go
func TestIonosCloudCluster_Conditions(t *testing.T) {
	conds := []metav1.Condition{{Type: "type", Status: metav1.ConditionTrue, Reason: "reason", LastTransitionTime: metav1.Now()}}
	cluster := &IonosCloudCluster{}

	cluster.SetV1Beta2Conditions(conds)
	require.Equal(t, conds, cluster.GetV1Beta2Conditions())
}
```

Remove the `clusterv1` import if no longer used. Remove the old `"sigs.k8s.io/cluster-api/util/conditions"` import.

- [ ] **Step 2: Update cluster status test**

In the `Context("Status")` Ginkgo block, replace:
```go
conditions.MarkTrue(fetched, clusterv1.ReadyCondition)
```
with the v1beta2 conditions API:
```go
conditions.Set(fetched, metav1.Condition{
    Type:    clusterv1.ReadyV1Beta2Condition,
    Status:  metav1.ConditionTrue,
    Reason:  clusterv1.ReadyV1Beta2Condition,
    Message: "Cluster is ready",
})
```

And update the assertion:
```go
Expect(fetched.Status.Conditions).To(BeEmpty())
```
becomes checking for `[]metav1.Condition`.

Replace:
```go
Expect(conditions.IsTrue(fetched, clusterv1.ReadyCondition)).To(BeTrue())
```
with:
```go
Expect(fetched.Status.Conditions).To(HaveLen(1))
Expect(fetched.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
```

Update the import to use `conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"` instead of the old `"sigs.k8s.io/cluster-api/util/conditions"`.

- [ ] **Step 3: Update machine condition test**

In `api/v1alpha1/ionoscloudmachine_types_test.go`, the `Context("Conditions")` block uses:
```go
conditions.MarkTrue(m, MachineProvisionedCondition)
```

This uses the old v1beta1 `conditions` import (`sigs.k8s.io/cluster-api/util/conditions`). Replace with v1beta2 API:

```go
conditions.Set(m, metav1.Condition{
    Type:    MachineProvisionedCondition,
    Status:  metav1.ConditionTrue,
    Reason:  MachineProvisionedCondition,
    Message: "Machine is provisioned",
})
```

Update assertions to check `metav1.Condition` fields:
```go
machineConditions := m.GetV1Beta2Conditions()
Expect(machineConditions).To(HaveLen(1))
Expect(machineConditions[0].Type).To(Equal(MachineProvisionedCondition))
Expect(machineConditions[0].Status).To(Equal(metav1.ConditionTrue))
```

Do the same for the Status context block which also calls `conditions.MarkTrue`.

Update the import alias to use `conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"`.

- [ ] **Step 4: Commit**

```bash
git add api/v1alpha1/ionoscloudcluster_types_test.go api/v1alpha1/ionoscloudmachine_types_test.go
git commit -m "test(api): update condition tests to use metav1.Condition"
```

---

### Task 9: Regenerate and verify

**Files:**
- Generated: `config/crd/bases/*.yaml` (CRD manifests)
- Generated: `api/v1alpha1/zz_generated.deepcopy.go`

- [ ] **Step 1: Regenerate manifests and deepcopy**

```bash
make manifests generate
```

Expected: CRDs regenerated without the `v1beta2` nested field. DeepCopy methods updated to reflect the new `[]metav1.Condition` on status directly.

- [ ] **Step 2: Run linter**

```bash
make lint-fix
```

Fix any import ordering or formatting issues.

- [ ] **Step 3: Run tests**

```bash
make test
```

Expected: All tests pass.

- [ ] **Step 4: Verify no remaining v1beta1 condition references**

```bash
grep -rn 'v1beta1conditions\|WithOwnedConditions{' --include='*.go' .
grep -rn 'GetConditions\|SetConditions' --include='*.go' . | grep -v V1Beta2 | grep -v zz_generated
grep -rn 'clusterv1\.ConditionType\|clusterv1\.Conditions\b\|clusterv1\.ConditionSeverity' --include='*.go' .
```

Expected: No matches for any of these patterns (except possibly in vendor or generated files).

- [ ] **Step 5: Commit generated files**

```bash
git add config/crd/ api/v1alpha1/zz_generated.deepcopy.go
git commit -m "chore: regenerate CRDs and deepcopy after condition migration"
```

---

### Task 10: Clean up importas linting config

**Files:**
- Modify: `.golangci.yml`

- [ ] **Step 1: Check if old `util/conditions` alias is configured**

If `.golangci.yml` has an `importas` alias for `sigs.k8s.io/cluster-api/util/conditions` (the old non-v1beta2 package), remove it since we no longer import that package.

- [ ] **Step 2: Commit if changed**

```bash
git add .golangci.yml
git commit -m "chore: remove unused importas alias for old conditions package"
```
