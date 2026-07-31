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
  `cluster.x-k8s.io/v1beta2` label set via `config/crd/kustomization.yaml`'s `commonLabels`.
  (The `+kubebuilder:metadata:annotations` marker on the Go types produces an *annotation*,
  which CAPI's contract resolution does not consult — the label is what's load-bearing.)
- Implements the v1beta2 status contract: `status.initialization.provisioned` and
  `status.conditions` (`[]metav1.Condition`).
- **Removes** `status.ready` and the v1beta1 `status.conditions` shape in place, on the same
  `v1alpha1` CRD apiVersion (no new stored version, no conversion webhook).

## Supported upgrade path

Staged, in place: **v0.6 → v0.7 → v0.8** (upgrade the CAPI core to v1.11+ alongside v0.8).

- Existing clusters keep running; **no recreation** is required (no `IonosCloudMachineTemplate`
  content change → no Machine rollout; running Machines are not torn down by a transient
  `provisioned=nil`).
- On the `v0.7 → v0.8` hop there is a brief window where existing objects report infra
  not-provisioned until the v0.8 controller reconciles them (self-heals on controller startup).
- **Breaking:** `.status.ready` and the v1beta1 `status.conditions` are removed. Any external
  tooling reading them must move to `status.initialization.provisioned` / `status.conditions`.
- **Residual risk of the in-place schema change:** objects written by v0.6/v0.7 can have
  `spec.controlPlaneEndpoint: {host: "", port: 0}` and conditions with an empty `reason` — both
  valid under the old schema, both rejected by the new one. The v0.8 controller backfills
  condition reasons and omits a zero `controlPlaneEndpoint` on its own writes
  (`api/v1alpha1/conditions_migration.go`, the `omitzero` tag on `ControlPlaneEndpoint`), and an
  in-place `Update` of an untouched legacy endpoint keeps working (Kubernetes CRD validation
  ratcheting allows an unchanged, already-invalid field to persist across an update). What still
  breaks: a fresh `Create` of the legacy zero-value shape — e.g. `clusterctl move`, which reads
  the object on the source cluster and re-creates it on the target, where there is no stored
  value to ratchet against. Verified empirically against a real apiserver (envtest, CRD swapped
  in place): a status-only patch of a legacy object round-trips cleanly, an in-place spec `Update`
  of the untouched endpoint succeeds, but a `Create` carrying the same legacy shape is rejected.

Empirical validation lives in the e2e upgrade suite (PR #379), which exercises the staged
`v0.6.3 → v0.7.0 → v0.8` path.

## References

- CAPI v1.10 → v1.11 provider migration guide:
  https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.10-to-v1.11
- CAPI v1.11 → v1.12 provider migration guide:
  https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.11-to-v1.12
- Design spec: `docs/superpowers/specs/2026-07-17-drop-v1beta1-pure-v1beta2-design.md`
- Compatibility matrix: `README.md`
