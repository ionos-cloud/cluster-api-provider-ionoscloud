# Cluster API v1.10.10 Upgrade Design

## Overview

Upgrade the IONOS Cloud Cluster API provider from Cluster API v1.8.12 to v1.10.10.
This is a version-bump-only upgrade implementing migration guide recommendations,
without adopting v1beta2 conditions or metav1.Condition.

## References

- [v1.8-to-v1.9 migration guide](https://release-1-9.cluster-api.sigs.k8s.io/developer/providers/migrations/v1.8-to-v1.9)
- [v1.9-to-v1.10 migration guide](https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.9-to-v1.10)
- PR #352 (v1.8→v1.9 reference implementation)
- PR #353 (v1.9→v1.10 reference implementation)

## Deliverables

Two PRs, merged in order.

---

## PR 1: Version Bump + Migration Guide Recommendations

### Version Bumps

| Dependency | Current | Target |
|---|---|---|
| `sigs.k8s.io/cluster-api` | v1.8.12 | v1.10.10 |
| `sigs.k8s.io/cluster-api/test` | v1.8.12 | v1.10.10 |
| `sigs.k8s.io/controller-runtime` | v0.18.7 | v0.20.x (aligned with CAPI v1.10) |
| `k8s.io/*` | v0.30.x | v0.32.x (aligned with controller-runtime) |

### Migration Fixes (v1.8 to v1.9)

**Predicate signature changes:**
- `ResourceNotPaused` and `ClusterUnpaused` now require a `scheme` argument.
- Affected file: `internal/controller/ionoscloudcluster_controller.go` (lines 264, 273).
- Pass `mgr.GetScheme()` or the runtime scheme as the new first argument.

### Migration Fixes (v1.9 to v1.10)

**E2EConfig function renames:**
- `GetVariable()` renamed to `MustGetVariable()`.
- Affected file: `test/e2e/suite_test.go` (2 call sites at line 222 and 236).

**Addons API graduated from experimental:**
- Import path `sigs.k8s.io/cluster-api/exp/addons/api/v1beta1` changed to `sigs.k8s.io/cluster-api/api/addons/v1beta1`.
- Affected files: `test/e2e/helpers/finalizers.go` (line 26), `test/e2e/helpers/ownerreference.go` (line 27).

**CRDMigrator controller:**
- Add `--skip-crd-migration-phases` command-line flag to `cmd/main.go`.
- Wire the `CRDMigrator` controller, registering all 4 provider CRDs:
  - `IonosCloudCluster`
  - `IonosCloudClusterTemplate`
  - `IonosCloudMachine`
  - `IonosCloudMachineTemplate`
- Add RBAC markers for CRD and custom resource management.
- Reference: PR #353 implementation, CAPI PR #11889.

### README Compatibility Table

Update the compatibility table in `README.md` to add a `Cluster API v1beta1 (v1.10)` column.
The next CAPIC release row (v0.7 or whatever the release version will be) should show compatibility
with v1.10. Existing rows for older CAPIC versions should show `☓` for v1.10.

### E2E / Config Updates

- Update CAPI component URLs in `test/e2e/config/ionoscloud.yaml` to v1.10.10.
- Add `test/e2e/data/shared/v1.10/metadata.yaml` for clusterctl compatibility.

### Generated Files

- `make manifests` (regenerate CRDs and RBAC for CRDMigrator markers)
- `make generate` (regenerate deepcopy)
- `make mocks` (regenerate mocks if Client interface changed)
- `go mod tidy`

### Explicitly Out of Scope

- No v1beta2 conditions (`V1Beta2` status fields, `GetV1Beta2Conditions`/`SetV1Beta2Conditions`)
- No `metav1.Condition` adoption
- No removal of `FailureReason`/`FailureMessage` (deferred to PR 2)

---

## PR 2: Remove Deprecated FailureReason/FailureMessage

The `sigs.k8s.io/cluster-api/errors` package was deprecated in CAPI v1.8 and the
v1.8-to-v1.9 migration guide recommends removal.

### Changes

- **`api/v1alpha1/ionoscloudmachine_types.go`**: Remove `FailureReason *errors.MachineStatusError` and `FailureMessage *string` fields from `IonosCloudMachineStatus`. Remove `sigs.k8s.io/cluster-api/errors` import.
- **`scope/machine.go`**: Remove `HasFailed()` method (lines 172-175) which checks these fields.
- **`scope/machine_test.go`**: Remove any tests for `HasFailed()`.
- **`api/v1alpha1/ionoscloudmachine_types_test.go`**: Remove test cases referencing `FailureReason`/`FailureMessage`.
- **Generated files**: `make manifests` + `make generate` to regenerate CRDs and deepcopy.
