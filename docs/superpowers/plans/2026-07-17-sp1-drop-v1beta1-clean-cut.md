# SP1 — Drop v1beta1 (pure v1beta2 v0.8) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make CAPIC a pure v1beta2 provider in v0.8 by declaring the v1beta2 contract and removing every v1beta1 residue from `#378` (branch `feat/upgrade-cluster-api-v1.11.11`).

**Architecture:** This is a removal-and-declaration change on top of the existing v1beta2 implementation already on the branch. We delete the dual-write plumbing (deprecated `status.deprecated.v1beta1.*` fields, `Get/SetV1Beta1*` helpers, v1beta1 condition ownership in patch calls), flip the declared contract in `metadata.yaml`, regenerate CRDs/deepcopy, and update docs. Commits are additive (no history rewrite).

**Tech Stack:** Go 1.26, kubebuilder v4, controller-gen (`make generate`/`make manifests`), Cluster API v1.11.11 (`sigs.k8s.io/cluster-api/api/core/v1beta2`), Ginkgo + envtest, testify.

## Global Constraints

- Go **1.26+**; do not change `go.mod` CAPI version (stays `sigs.k8s.io/cluster-api v1.11.11` — the v1.12 bump is PR #380, out of scope here).
- Never hand-edit generated files (`api/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases/*.yaml`); regenerate via `make generate` / `make manifests`.
- Linting is strict (`golangci.yml`, `default: none`). Run `make lint-fix` before considering a task done; unused imports are compile errors — remove them.
- Every commit message ends with the trailer line: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.
- This is removal-heavy work: "tests" are compilation (`go build`/`go vet`), the existing suite still passing, and `grep` assertions that removed symbols are gone. Behavior that must be preserved (`Status.Initialization.Provisioned = true`) is already covered by existing controller/server tests — do not add new tests for it.
- Keep the CRD annotation `cluster.x-k8s.io/v1beta2=v1alpha1` and all v1beta2 status fields. Do NOT add a conversion webhook. Do NOT change the CRD version (`v1alpha1`).

---

### Task 1: Stop writing the deprecated v1beta1 `ready` field

**Files:**
- Modify: `internal/controller/ionoscloudcluster_controller.go` (~lines 181–189)
- Modify: `internal/service/cloud/server.go` (~lines 148–150)

**Interfaces:**
- Consumes: nothing new.
- Produces: nothing new. After this task the deprecated helpers/fields still exist but are no longer written from the reconcile paths.

- [ ] **Step 1: Remove the deprecated dual-write in the cluster controller**

In `internal/controller/ionoscloudcluster_controller.go`, replace this block:

```go
	clusterScope.IonosCluster.Status.Initialization.Provisioned = new(true)
	// Set deprecated v1beta1 ready field for backwards compatibility.
	if clusterScope.IonosCluster.Status.Deprecated == nil {
		clusterScope.IonosCluster.Status.Deprecated = &infrav1.IonosCloudClusterDeprecatedStatus{}
	}
	if clusterScope.IonosCluster.Status.Deprecated.V1Beta1 == nil {
		clusterScope.IonosCluster.Status.Deprecated.V1Beta1 = &infrav1.IonosCloudClusterV1Beta1DeprecatedStatus{}
	}
	clusterScope.IonosCluster.Status.Deprecated.V1Beta1.Ready = true //nolint:staticcheck // Intentionally setting deprecated field for v1beta1 backwards compatibility.
	return ctrl.Result{}, nil
```

with:

```go
	clusterScope.IonosCluster.Status.Initialization.Provisioned = new(true)
	return ctrl.Result{}, nil
```

- [ ] **Step 2: Remove the deprecated dual-write in the server service**

In `internal/service/cloud/server.go`, replace:

```go
	ms.IonosMachine.Status.Initialization.Provisioned = new(true)
	// Set deprecated v1beta1 ready field for backwards compatibility.
	ms.IonosMachine.SetV1Beta1Ready(true)
	conditions.Set(ms.IonosMachine, metav1.Condition{
```

with:

```go
	ms.IonosMachine.Status.Initialization.Provisioned = new(true)
	conditions.Set(ms.IonosMachine, metav1.Condition{
```

- [ ] **Step 3: Verify it compiles and vets**

Run: `go build ./... && go vet ./internal/...`
Expected: no output, exit 0.

- [ ] **Step 4: Run the affected unit tests**

Run: `go test ./internal/controller/... ./internal/service/cloud/... -short`
Expected: `ok` for both packages (no test asserts the deprecated `ready` write).

- [ ] **Step 5: Commit**

```bash
git add internal/controller/ionoscloudcluster_controller.go internal/service/cloud/server.go
git commit -m "refactor: stop writing deprecated v1beta1 ready field

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Drop v1beta1 condition ownership from the scope patch calls

**Files:**
- Modify: `scope/cluster.go` (~lines 39–47 and ~230–242)
- Modify: `scope/machine.go` (~lines 38–46 and ~197–209)

**Interfaces:**
- Consumes: `patch.WithOwnedConditions` (unchanged; kept).
- Produces: `PatchObject` on both scopes now declares only v1beta2 condition ownership.

- [ ] **Step 1: Simplify the `ownedClusterConditions` doc comment**

In `scope/cluster.go`, replace:

```go
// ownedClusterConditions is the single source of truth for the v2 condition types owned by the
// cluster controller. WithOwnedV1Beta1Conditions is derived from this list (only ReadyCondition
// is carried into v1beta1; provider-specific conditions are v2-only). Keeping both lists adjacent
// and derived from one variable makes it structurally impossible to add a condition to one side
// while forgetting the other.
var ownedClusterConditions = []string{
```

with:

```go
// ownedClusterConditions is the single source of truth for the condition types owned by the
// cluster controller.
var ownedClusterConditions = []string{
```

- [ ] **Step 2: Remove the v1beta1 ownership option from the cluster patch call**

In `scope/cluster.go`, replace:

```go
	// V1Beta1 ownership covers only the cross-cutting Ready condition; provider-specific
	// conditions (IonosCloudClusterReady) are v2-only and intentionally omitted here.
	if patchErr := c.patchHelper.Patch(timeoutCtx, c.IonosCluster,
		patch.WithOwnedV1Beta1Conditions{
			Conditions: []clusterv1.ConditionType{clusterv1.ReadyCondition},
		},
		// V2 ownership is derived from ownedClusterConditions — the single source of truth.
		patch.WithOwnedConditions{
			Conditions: ownedClusterConditions,
		},
	); patchErr != nil {
		return patchErr
	}
```

with:

```go
	// Ownership is derived from ownedClusterConditions — the single source of truth.
	if patchErr := c.patchHelper.Patch(timeoutCtx, c.IonosCluster,
		patch.WithOwnedConditions{
			Conditions: ownedClusterConditions,
		},
	); patchErr != nil {
		return patchErr
	}
```

- [ ] **Step 3: Simplify the `ownedMachineConditions` doc comment**

In `scope/machine.go`, replace:

```go
// ownedMachineConditions is the single source of truth for the v2 condition types owned by the
// machine controller. WithOwnedV1Beta1Conditions is derived from this list (only ReadyCondition
// is carried into v1beta1; provider-specific conditions are v2-only). Keeping both lists adjacent
// and derived from one variable makes it structurally impossible to add a condition to one side
// while forgetting the other.
var ownedMachineConditions = []string{
```

with:

```go
// ownedMachineConditions is the single source of truth for the condition types owned by the
// machine controller.
var ownedMachineConditions = []string{
```

- [ ] **Step 4: Remove the v1beta1 ownership option from the machine patch call**

In `scope/machine.go`, replace:

```go
	if patchErr := m.patchHelper.Patch(
		timeoutCtx,
		m.IonosMachine,
		// V1Beta1 ownership covers only the cross-cutting Ready condition; provider-specific
		// conditions (MachineProvisionedCondition) are v2-only and intentionally omitted here.
		patch.WithOwnedV1Beta1Conditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		}},
		// V2 ownership is derived from ownedMachineConditions — the single source of truth.
		patch.WithOwnedConditions{Conditions: ownedMachineConditions},
	); patchErr != nil {
		return patchErr
	}
```

with:

```go
	if patchErr := m.patchHelper.Patch(
		timeoutCtx,
		m.IonosMachine,
		// Ownership is derived from ownedMachineConditions — the single source of truth.
		patch.WithOwnedConditions{Conditions: ownedMachineConditions},
	); patchErr != nil {
		return patchErr
	}
```

- [ ] **Step 5: Verify build/vet (confirms `clusterv1` still used via `ReadyCondition`)**

Run: `go build ./scope/... && go vet ./scope/...`
Expected: no output, exit 0. (If `clusterv1` reports as unused, something else changed — it must remain, used by `string(clusterv1.ReadyCondition)` in both `owned*Conditions` and `SetSummaryCondition`.)

- [ ] **Step 6: Run scope tests**

Run: `go test ./scope/... -short`
Expected: `ok  github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope`.

- [ ] **Step 7: Commit**

```bash
git add scope/cluster.go scope/machine.go
git commit -m "refactor(scope): drop v1beta1 condition ownership from patch calls

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Remove the deprecated v1beta1 condition tests

**Files:**
- Modify: `api/v1alpha1/ionoscloudcluster_types_test.go`
- Modify: `api/v1alpha1/ionoscloudmachine_types_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: the api test package no longer references `Get/SetV1Beta1Conditions` or `deprecatedv1beta1conditions`, so Task 4 can remove those helpers without breaking tests.

- [ ] **Step 1: Delete the standalone `TestIonosCloudCluster_Conditions` unit test**

In `api/v1alpha1/ionoscloudcluster_types_test.go`, delete this function entirely:

```go
func TestIonosCloudCluster_Conditions(t *testing.T) {
	conds := clusterv1.Conditions{{Type: "type"}}
	cluster := &IonosCloudCluster{}

	cluster.SetV1Beta1Conditions(conds)
	require.Equal(t, conds, cluster.GetV1Beta1Conditions())
}
```

- [ ] **Step 2: Remove the deprecated assertions from the cluster "Status" spec**

In the same file, inside `Context("Status", ...)`, delete these three lines (leaving the surrounding `Provisioned`/`CurrentRequestByDatacenter` assertions intact):

- `Expect(fetched.GetV1Beta1Conditions()).To(BeEmpty())`
- `deprecatedv1beta1conditions.MarkTrue(fetched, clusterv1.ReadyCondition)`
- `Expect(fetched.GetV1Beta1Conditions()).To(HaveLen(1))`
- `Expect(deprecatedv1beta1conditions.IsTrue(fetched, clusterv1.ReadyCondition)).To(BeTrue())`

After the edit that spec block reads:

```go
	Context("Status", func() {
		It("should correctly get and set the status", func() {
			By("initially having an empty status")

			cluster := defaultCluster()
			Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())

			key := client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}
			fetched := &IonosCloudCluster{}
			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Status.Initialization.Provisioned).To(BeNil())
			Expect(fetched.Status.CurrentRequestByDatacenter).To(BeEmpty())

			By("retrieving the cluster and setting the status")
			fetched.Status.Initialization = IonosCloudClusterInitializationStatus{Provisioned: new(true)}
			wantProvisionRequest := ProvisioningRequest{
				Method:      "POST",
				RequestPath: "/path/to/resource",
				State:       "QUEUED",
			}
			fetched.Status.CurrentRequestByDatacenter = map[string]ProvisioningRequest{
				"123": wantProvisionRequest,
			}

			By("updating the cluster status")
			Expect(k8sClient.Status().Update(context.Background(), fetched)).To(Succeed())

			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Status.Initialization.Provisioned).To(HaveValue(BeTrue()))
			Expect(fetched.Status.CurrentRequestByDatacenter).To(HaveLen(1))
			Expect(fetched.Status.CurrentRequestByDatacenter["123"]).To(Equal(wantProvisionRequest))

			By("Removing the entry from the status again")
			delete(fetched.Status.CurrentRequestByDatacenter, "123")
			Expect(k8sClient.Status().Update(context.Background(), fetched)).To(Succeed())

			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Status.CurrentRequestByDatacenter).To(BeEmpty())
		})
	})
```

- [ ] **Step 3: Fix the cluster test imports (remove now-unused ones)**

In `api/v1alpha1/ionoscloudcluster_types_test.go`, replace the import block:

```go
import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	deprecatedv1beta1conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)
```

with (drop `testing`, `require`, `deprecatedv1beta1conditions`; keep `clusterv1` — still used by `clusterv1.APIEndpoint` in `defaultCluster`):

```go
import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)
```

- [ ] **Step 4: Delete the machine `Context("Conditions", ...)` spec**

In `api/v1alpha1/ionoscloudmachine_types_test.go`, delete this entire block:

```go
	Context("Conditions", func() {
		It("should correctly set and get the conditions", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(
				context.Background(), client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

			// Calls SetConditions with required fields
			deprecatedv1beta1conditions.MarkTrue(m, MachineProvisionedCondition)

			Expect(k8sClient.Status().Update(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(context.Background(),
				client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

			machineConditions := m.GetV1Beta1Conditions()
			Expect(machineConditions).To(HaveLen(1))
			Expect(machineConditions[0].Type).To(Equal(MachineProvisionedCondition))
			Expect(machineConditions[0].Status).To(Equal(corev1.ConditionTrue))
		})
	})
```

- [ ] **Step 5: Remove the deprecated condition set-up from the machine "Status" spec**

In the same file, inside `Context("Status", ...)`, delete the single line:

```go
			deprecatedv1beta1conditions.MarkTrue(m, MachineProvisionedCondition)
```

(The `want := *m.DeepCopy()` / `cmp.Diff(want.Status, m.Status)` round-trip below stays and still passes — it now compares a status without the deprecated subtree.)

- [ ] **Step 6: Fix the machine test imports (remove `deprecatedv1beta1conditions`)**

In `api/v1alpha1/ionoscloudmachine_types_test.go`, replace the import block:

```go
import (
	"context"

	"github.com/google/go-cmp/cmp"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	deprecatedv1beta1conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)
```

with (drop `deprecatedv1beta1conditions`; keep `corev1` — still used by `corev1.TypedLocalObjectReference` elsewhere):

```go
import (
	"context"

	"github.com/google/go-cmp/cmp"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)
```

- [ ] **Step 7: Verify the api test package still builds and vets**

Run: `go vet ./api/...`
Expected: no output, exit 0. (Compilation covers the test files; a leftover unused import or missing symbol fails here.)

- [ ] **Step 8: Commit**

```bash
git add api/v1alpha1/ionoscloudcluster_types_test.go api/v1alpha1/ionoscloudmachine_types_test.go
git commit -m "test(api): remove deprecated v1beta1 condition tests

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Remove deprecated v1beta1 status fields/helpers and regenerate

**Files:**
- Modify: `api/v1alpha1/ionoscloudcluster_types.go`
- Modify: `api/v1alpha1/ionoscloudmachine_types.go`
- Regenerate: `api/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases/infrastructure.cluster.x-k8s.io_ionoscloudclusters.yaml`, `config/crd/bases/infrastructure.cluster.x-k8s.io_ionoscloudmachines.yaml`

**Interfaces:**
- Consumes: nothing new.
- Produces: `IonosCloudClusterStatus` / `IonosCloudMachineStatus` no longer have a `Deprecated` field; the `Deprecated*`/`V1Beta1*` types and `Get/SetV1Beta1Conditions`/`SetV1Beta1Ready` methods no longer exist. `clusterv1` stays imported in both types files (used by the `clusterv1.ConditionType` constants).

- [ ] **Step 1: Remove the `Deprecated` field from `IonosCloudClusterStatus`**

In `api/v1alpha1/ionoscloudcluster_types.go`, delete these three lines (the last member of the status struct):

```go
	// Deprecated groups all status fields deprecated and scheduled for removal when v1beta1 contract support is dropped.
	//+optional
	Deprecated *IonosCloudClusterDeprecatedStatus `json:"deprecated,omitempty"`
```

- [ ] **Step 2: Remove the cluster deprecated status types**

In the same file, delete both type declarations:

```go
// IonosCloudClusterDeprecatedStatus groups all status fields deprecated and scheduled for removal when
// v1beta1 contract support is dropped.
type IonosCloudClusterDeprecatedStatus struct {
	// V1Beta1 groups all v1beta1 status fields that are deprecated and scheduled for removal.
	//+optional
	V1Beta1 *IonosCloudClusterV1Beta1DeprecatedStatus `json:"v1beta1,omitempty"`
}

// IonosCloudClusterV1Beta1DeprecatedStatus contains deprecated v1beta1 fields.
type IonosCloudClusterV1Beta1DeprecatedStatus struct {
	// Ready indicates that the cluster is ready.
	//
	// Deprecated: Use Initialization.Provisioned instead.
	//+optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IonosCloudCluster using the deprecated v1beta1 condition type.
	//
	// Deprecated: Use the top-level conditions field instead.
	//+optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}
```

- [ ] **Step 3: Remove the cluster deprecated helper methods**

In the same file, delete both methods:

```go
// GetV1Beta1Conditions returns the deprecated v1beta1 conditions from status.deprecated.v1beta1.conditions.
func (i *IonosCloudCluster) GetV1Beta1Conditions() clusterv1.Conditions {
	if i.Status.Deprecated == nil || i.Status.Deprecated.V1Beta1 == nil {
		return nil
	}
	return i.Status.Deprecated.V1Beta1.Conditions
}

// SetV1Beta1Conditions sets the deprecated v1beta1 conditions in status.deprecated.v1beta1.conditions.
func (i *IonosCloudCluster) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	if i.Status.Deprecated == nil {
		i.Status.Deprecated = &IonosCloudClusterDeprecatedStatus{}
	}
	if i.Status.Deprecated.V1Beta1 == nil {
		i.Status.Deprecated.V1Beta1 = &IonosCloudClusterV1Beta1DeprecatedStatus{}
	}
	i.Status.Deprecated.V1Beta1.Conditions = conditions
}
```

- [ ] **Step 4: Remove the `Deprecated` field from `IonosCloudMachineStatus`**

In `api/v1alpha1/ionoscloudmachine_types.go`, delete these three lines:

```go
	// Deprecated groups all status fields deprecated and scheduled for removal when v1beta1 contract support is dropped.
	//+optional
	Deprecated *IonosCloudMachineDeprecatedStatus `json:"deprecated,omitempty"`
```

- [ ] **Step 5: Remove the machine deprecated status types**

In the same file, delete both type declarations:

```go
// IonosCloudMachineDeprecatedStatus groups all status fields deprecated and scheduled for removal when
// v1beta1 contract support is dropped.
type IonosCloudMachineDeprecatedStatus struct {
	// V1Beta1 groups all v1beta1 status fields that are deprecated and scheduled for removal.
	//+optional
	V1Beta1 *IonosCloudMachineV1Beta1DeprecatedStatus `json:"v1beta1,omitempty"`
}

// IonosCloudMachineV1Beta1DeprecatedStatus contains deprecated v1beta1 fields.
type IonosCloudMachineV1Beta1DeprecatedStatus struct {
	// Ready indicates the VM has been provisioned and is ready.
	//
	// Deprecated: Use Initialization.Provisioned instead.
	//+optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IonosCloudMachine using the deprecated v1beta1 condition type.
	//
	// Deprecated: Use the top-level conditions field instead.
	//+optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}
```

- [ ] **Step 6: Remove the machine deprecated helper methods**

In the same file, delete all three methods:

```go
// GetV1Beta1Conditions returns the deprecated v1beta1 conditions from status.deprecated.v1beta1.conditions.
func (m *IonosCloudMachine) GetV1Beta1Conditions() clusterv1.Conditions {
	if m.Status.Deprecated == nil || m.Status.Deprecated.V1Beta1 == nil {
		return nil
	}
	return m.Status.Deprecated.V1Beta1.Conditions
}

// SetV1Beta1Conditions sets the deprecated v1beta1 conditions in status.deprecated.v1beta1.conditions.
func (m *IonosCloudMachine) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	if m.Status.Deprecated == nil {
		m.Status.Deprecated = &IonosCloudMachineDeprecatedStatus{}
	}
	if m.Status.Deprecated.V1Beta1 == nil {
		m.Status.Deprecated.V1Beta1 = &IonosCloudMachineV1Beta1DeprecatedStatus{}
	}
	m.Status.Deprecated.V1Beta1.Conditions = conditions
}

// SetV1Beta1Ready sets the deprecated v1beta1 ready field.
func (m *IonosCloudMachine) SetV1Beta1Ready(ready bool) {
	if m.Status.Deprecated == nil {
		m.Status.Deprecated = &IonosCloudMachineDeprecatedStatus{}
	}
	if m.Status.Deprecated.V1Beta1 == nil {
		m.Status.Deprecated.V1Beta1 = &IonosCloudMachineV1Beta1DeprecatedStatus{}
	}
	m.Status.Deprecated.V1Beta1.Ready = ready
}
```

- [ ] **Step 7: Regenerate deepcopy and CRDs**

Run: `make generate manifests`
Expected: succeeds. `git status` shows modifications to `api/v1alpha1/zz_generated.deepcopy.go` and the two CRD YAMLs under `config/crd/bases/`.

- [ ] **Step 8: Confirm the generated artifacts no longer contain the deprecated subtree**

Run: `grep -rniE "deprecated|v1beta1" config/crd/bases/infrastructure.cluster.x-k8s.io_ionoscloudclusters.yaml config/crd/bases/infrastructure.cluster.x-k8s.io_ionoscloudmachines.yaml api/v1alpha1/zz_generated.deepcopy.go`
Expected: no matches (empty output).

- [ ] **Step 9: Build and vet the whole module**

Run: `go build ./... && go vet ./...`
Expected: no output, exit 0. (If `clusterv1` reports unused in a types file, verify the `IonosCloudClusterReady` / `MachineProvisionedCondition` constants still reference `clusterv1.ConditionType` — they must remain.)

- [ ] **Step 10: Commit**

```bash
git add api/v1alpha1/ionoscloudcluster_types.go api/v1alpha1/ionoscloudmachine_types.go api/v1alpha1/zz_generated.deepcopy.go config/crd/bases/
git commit -m "feat(api)!: remove deprecated v1beta1 status fields and helpers

BREAKING CHANGE: status.ready and the v1beta1 status.conditions are removed from
IonosCloudCluster/IonosCloudMachine. Consumers must read status.initialization.provisioned
and status.conditions (metav1.Condition).

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Declare the v1beta2 contract (metadata + README)

**Files:**
- Modify: `metadata.yaml`
- Modify: `README.md` (compatibility matrix, ~lines 50–55)

**Interfaces:**
- Consumes: nothing.
- Produces: clusterctl sees the `0.8` release series as `contract: v1beta2`, matching the CRD annotation.

- [ ] **Step 1: Add the v0.8 release series as v1beta2**

In `metadata.yaml`, replace:

```yaml
releaseSeries:
- major: 0
  minor: 7
  contract: v1beta1
```

with:

```yaml
releaseSeries:
- major: 0
  minor: 8
  contract: v1beta2
- major: 0
  minor: 7
  contract: v1beta1
```

- [ ] **Step 2: Add the v0.8 row to the README compatibility matrix**

In `README.md`, replace the v0.7 matrix row:

```
| CAPIC v1alpha1 (v0.7) |             ☓              |             ☓              |              ☓              |                  ✓                  |
```

with the v0.7 row followed by a new v0.8 row:

```
| CAPIC v1alpha1 (v0.7) |             ☓              |             ☓              |              ☓              |                  ✓                  |
| CAPIC v1alpha1 (v0.8) |             ☓              |             ☓              |              ☓              |                  ✓                  |
```

- [ ] **Step 3: Add a contract note under the matrix**

In `README.md`, immediately after the compatibility table (before the `### Kubernetes Versions` heading), insert:

```markdown

> **v0.8 declares the Cluster API `v1beta2` contract and drops `v1beta1`.** It requires a
> `v1beta2`-capable core (CAPI v1.11+). The deprecated `status.ready` and v1beta1
> `status.conditions` fields are removed — read `status.initialization.provisioned` and
> `status.conditions` instead. Upgrade existing clusters via the staged path
> `v0.6 → v0.7 → v0.8`.
```

- [ ] **Step 4: Sanity-check YAML validity**

Run: `go run sigs.k8s.io/cluster-api/cmd/clusterctl version >/dev/null 2>&1; grep -nE "minor:|contract:" metadata.yaml`
Expected: the `grep` output lists `minor: 8` / `contract: v1beta2` at the top, then `minor: 7` / `contract: v1beta1`, etc. (The `clusterctl` invocation is optional; if unavailable, just eyeball the YAML indentation matches the existing entries.)

- [ ] **Step 5: Commit**

```bash
git add metadata.yaml README.md
git commit -m "feat(metadata): declare v1beta2 contract for v0.8

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: Rewrite the CAPI upgrade plan doc

**Files:**
- Overwrite: `docs/capi-upgrade-plan.md`

**Interfaces:**
- Consumes: nothing.
- Produces: an accurate migration doc reflecting the real v0.6/v0.7/v0.8 timeline and the clean cut.

- [ ] **Step 1: Replace the entire contents of `docs/capi-upgrade-plan.md`**

Write this exact content:

````markdown
# Cluster API upgrade & v1beta2 migration

How CAPIC moves to the Cluster API **v1beta2** contract, and how the release series line up.

## Release timeline (actual)

| CAPIC | CAPI | Declared contract | Implements v1beta2 | Notes |
|-------|------|-------------------|--------------------|-------|
| v0.6.x | v1.8 | v1beta1 | no | pure v1beta1 |
| v0.7.0 | v1.10 | v1beta1 | no | pure v1beta1 (CAPI dependency bump only) |
| **v0.8** | **v1.11** | **v1beta2** | yes | **pure v1beta2 — v1beta1 support dropped** |

There is no dual-writing "bridge" release: v0.7.0 shipped as pure v1beta1, so v0.8 is the
first release to implement v1beta2, and it does so as a clean cut.

## What v0.8 changes

- Declares the **v1beta2** contract (`metadata.yaml`), matching the CRD
  `cluster.x-k8s.io/v1beta2` annotation.
- Implements the v1beta2 status contract: `status.initialization.provisioned` and
  `status.conditions` (`[]metav1.Condition`).
- **Removes** all v1beta1 residue: the `status.deprecated.v1beta1.*` fields, the
  `Get/SetV1Beta1Conditions` / `SetV1Beta1Ready` helpers, and the v1beta1 condition
  ownership in the patch helpers.

## Supported upgrade path

Staged, in place: **v0.6 → v0.7 → v0.8** (upgrade the CAPI core to v1.11+ alongside v0.8).

- Existing clusters keep running; **no recreation** is required (no `IonosCloudMachineTemplate`
  content change → no Machine rollout; running Machines are not torn down by a transient
  `provisioned=nil`).
- On the `v0.7 → v0.8` hop there is a brief window where existing objects report infra
  not-provisioned until the v0.8 controller reconciles them (self-heals on controller startup).
- **Breaking:** `.status.ready` and the v1beta1 `status.conditions` are removed. Any external
  tooling reading them must move to `status.initialization.provisioned` / `status.conditions`.

Empirical validation lives in the e2e upgrade suite (PR #379), which exercises the staged
`v0.6.3 → v0.7.0 → v0.8` path.

## References

- CAPI v1.11 → v1.12 provider migration guide:
  https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.11-to-v1.12
- Design spec: `docs/superpowers/specs/2026-07-17-drop-v1beta1-pure-v1beta2-design.md`
- Compatibility matrix: `README.md`
````

- [ ] **Step 2: Commit**

```bash
git add docs/capi-upgrade-plan.md
git commit -m "docs: rewrite capi upgrade plan for pure-v1beta2 v0.8

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 7: Full verification gate

**Files:** none created; may regenerate/format.

**Interfaces:** none.

- [ ] **Step 1: Confirm no v1beta1 residue remains in production Go code**

Run: `grep -rniE "v1beta1|V1Beta1" --include="*.go" api/ scope/ internal/ cmd/ | grep -v zz_generated | grep -viE "_test.go"`
Expected: no matches (empty output). The only `clusterv1` imports point at `.../api/core/v1beta2` (string `v1beta2`), so they do not match.

- [ ] **Step 2: Confirm the deprecated symbols are fully gone**

Run: `grep -rniE "SetV1Beta1Ready|GetV1Beta1Conditions|SetV1Beta1Conditions|WithOwnedV1Beta1Conditions|DeprecatedStatus" --include="*.go" . | grep -v zz_generated`
Expected: no matches (empty output).

- [ ] **Step 3: Run linters**

Run: `make lint`
Expected: `0 issues`. If `make lint-fix` changes formatting, re-stage and amend the most relevant commit or add a small `chore: lint-fix` commit.

- [ ] **Step 4: Run the full test suite**

Run: `make test`
Expected: unit tests, vet, and the api envtest integration suite all pass (`ok`/green). This exercises the retained v1beta2 status round-trip and the controller/service provisioning behavior.

- [ ] **Step 5: Verify generated artifacts are up to date**

Run: `make verify`
Expected: `verify-gen` and `verify-tidy` pass (no diff). If it reports a diff, run `make generate manifests` / `go mod tidy`, commit the result, and re-run.

- [ ] **Step 6: Final commit (only if steps 3/5 produced changes)**

```bash
git add -A
git commit -m "chore: regenerate and lint-fix after dropping v1beta1

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

If nothing changed, skip this step.

---

## Self-Review

**Spec coverage** (against `docs/superpowers/specs/2026-07-17-drop-v1beta1-pure-v1beta2-design.md` §4):
- §4.1 API types (fields/types/helpers) → Task 4. ✅
- §4.2 controllers/services dual-write → Task 1. ✅
- §4.3 scope plumbing → Task 2. ✅
- §4.4 generated (deepcopy/CRDs) → Task 4 steps 7–8. ✅
- §4.5 metadata + README + capi-upgrade-plan rewrite → Tasks 5 & 6. ✅
- §4.6 tests → Task 3. ✅
- §4.7 what stays (v1beta2 impl, CRD annotation, condition constants) → preserved by construction; asserted in Task 4 step 9 and Task 7 step 1. ✅
- §4.8 verification (make test/verify/lint + grep + no `status.deprecated`) → Task 7 + Task 4 step 8. ✅

SP2 (#380 rebase) and SP3 (#379 rebase + staged e2e) are separate plans, per the spec's decomposition — intentionally out of scope here.

**Placeholder scan:** No TBD/TODO; every code step shows the exact before/after content and every command has an expected result. ✅

**Type consistency:** `WithOwnedConditions` / `ownedClusterConditions` / `ownedMachineConditions` used consistently; `clusterv1` = `sigs.k8s.io/cluster-api/api/core/v1beta2` throughout; helper names (`GetV1Beta1Conditions`, `SetV1Beta1Conditions`, `SetV1Beta1Ready`) match the removals in Tasks 1/3/4. ✅
````
