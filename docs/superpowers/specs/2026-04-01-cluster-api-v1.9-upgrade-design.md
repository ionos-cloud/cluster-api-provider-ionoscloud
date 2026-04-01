# Cluster API v1.9.11 Upgrade Design

**Date:** 2026-04-01
**Status:** Approved

## Overview

Upgrade `sigs.k8s.io/cluster-api` from v1.8.12 to v1.9.11, including all required dependency bumps, adoption of the v1beta2 conditions API, removal of the deprecated `errors` package, and updating the e2e test configuration to use v1.9.11 component manifests.

## Scope

- Go module dependency bumps
- API type changes (v1beta2 conditions, remove deprecated fields)
- Controller and scope code changes (predicates, conditions call sites)
- e2e config URL updates
- Regeneration of CRDs, DeepCopy, and mocks

## 1. Go Module Updates

Update the following direct dependencies in `go.mod`, then run `go mod tidy` to resolve transitive deps:

| Dependency | Current | Target |
|---|---|---|
| `sigs.k8s.io/cluster-api` | v1.8.12 | v1.9.11 |
| `sigs.k8s.io/cluster-api/test` | v1.8.12 | v1.9.11 |
| `sigs.k8s.io/controller-runtime` | v0.18.7 | v0.19.6 |
| `k8s.io/api` | v0.30.14 | v0.31.3 |
| `k8s.io/apimachinery` | v0.30.14 | v0.31.3 |
| `k8s.io/client-go` | v0.30.14 | v0.31.3 |

Indirect k8s.io packages (`apiextensions-apiserver`, `apiserver`, `cluster-bootstrap`, `component-base`) will align to v0.31.x automatically via `go mod tidy`.

## 2. API Type Changes

### 2a. Add v1beta2 conditions to both status types

Add a nested `V1Beta2` status struct to `IonosCloudClusterStatus` and `IonosCloudMachineStatus`, each holding a `[]metav1.Condition` field:

```go
// IonosCloudClusterV1Beta2Status groups fields used when the CAPI contract moves to v1beta2.
type IonosCloudClusterV1Beta2Status struct {
    // Conditions represents the observations of the current state.
    //+optional
    //+listType=map
    //+listMapKey=type
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// In IonosCloudClusterStatus:
// V1Beta2 groups all status fields that will be used when the CAPI contract moves to v1beta2.
//+optional
V1Beta2 *IonosCloudClusterV1Beta2Status `json:"v1beta2,omitempty"`
```

Same pattern for `IonosCloudMachineV1Beta2Status` / `IonosCloudMachineStatus`.

Both types implement the `v1beta2conditions.Getter` and `v1beta2conditions.Setter` interfaces:

```go
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

### 2b. Remove `FailureReason` and `FailureMessage` from `IonosCloudMachineStatus`

These fields use the deprecated `sigs.k8s.io/cluster-api/errors` package. They are set in one place (`scope/machine.go` via `SetFailure`) and not read by any controller logic. They are removed and their failure signaling replaced by a v1beta2 condition with `Status: metav1.ConditionFalse`.

### 2c. Remove `sigs.k8s.io/cluster-api/errors` import

`ionoscloudmachine_types.go` is the only file importing this package. Removing the `FailureReason` field eliminates the import entirely.

## 3. Controller and Scope Changes

### 3a. Fix predicate signatures

In `internal/controller/ionoscloudcluster_controller.go`, update 2 call sites where predicates now require `(scheme *runtime.Scheme, logger logr.Logger)` instead of just `(logger logr.Logger)`:

```go
// Before
predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))
predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx))

// After
predicates.ResourceNotPaused(r.Scheme, ctrl.LoggerFrom(ctx))
predicates.ClusterUnpaused(r.Scheme, ctrl.LoggerFrom(ctx))
```

Both controllers already hold `r.Scheme` from the manager.

### 3b. Migrate conditions call sites

Replace all 6 `util/conditions` (v1beta1) call sites in `internal/controller/` and `scope/` with `util/conditions/v1beta2`. Import alias: `v1beta2conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"`.

Mapping:

| Old (v1beta1) | New (v1beta2) |
|---|---|
| `conditions.MarkTrue(obj, condType)` | `v1beta2conditions.Set(obj, metav1.Condition{Type: string(condType), Status: metav1.ConditionTrue, Reason: "..."})` |
| `conditions.MarkFalse(obj, condType, reason, sev, msg)` | `v1beta2conditions.Set(obj, metav1.Condition{Type: string(condType), Status: metav1.ConditionFalse, Reason: reason, Message: msg})` |
| `conditions.SetSummary(obj, conditions.WithConditions(...))` | `v1beta2conditions.SetSummaryCondition(obj, obj, condType, v1beta2conditions.ForConditionTypes{...})` |

Call sites:
- `internal/controller/ionoscloudcluster_controller.go`: 1 `MarkTrue`
- `internal/controller/ionoscloudmachine_controller.go`: 2 `MarkFalse`
- `internal/service/cloud/server.go`: 1 `MarkTrue`
- `scope/cluster.go`: 1 `SetSummary`
- `scope/machine.go`: 1 `SetSummary`

### 3c. Remove `SetFailure` from `scope/machine.go`

The `SetFailure` method (which sets `FailureReason` and `FailureMessage`) is removed. Its call site in the machine controller is replaced with a `v1beta2conditions.Set` call using `Status: metav1.ConditionFalse`.

## 4. e2e Configuration

In `test/e2e/config/ionoscloud.yaml`, replace the three CAPI v1.8.1 component manifest URLs with v1.9.11:

- `core-components.yaml` URL: `v1.8.1` → `v1.9.11`
- `bootstrap-components.yaml` URL: `v1.8.1` → `v1.9.11`
- `control-plane-components.yaml` URL: `v1.8.1` → `v1.9.11`

## 5. Code Generation and Verification

After all code changes:

```bash
make manifests   # regenerate CRDs (picks up V1Beta2 field, drops FailureReason/FailureMessage)
make generate    # regenerate DeepCopy for new V1Beta2 status structs
make mocks       # regenerate mockery mocks in case interfaces changed
make build       # compile + lint + vet — catches remaining issues
make test        # unit + integration tests
make verify      # confirm generated files match committed state
```

## Risk Areas

- **CRD change**: removing `FailureReason`/`FailureMessage` is a backward-incompatible CRD field removal. Existing clusters with these fields set will have those values silently dropped on next status update. Since no controller logic reads these fields, runtime behavior is unaffected.
- **`SetSummaryCondition` signature**: verify exact option types for `v1beta2conditions.SetSummaryCondition` against the v1.9.11 source before implementing, as the summary API differs more substantially from v1beta1 than the simple getter/setter calls.
- **Import alias**: confirm `v1beta2conditions` is not already reserved by the `importas` linter config in `.golangci.yml` before using it.
