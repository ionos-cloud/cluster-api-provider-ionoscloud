# Design: drop v1beta1, make CAPIC a pure v1beta2 provider (v0.8)

**Date:** 2026-07-17
**Status:** approved (design), pending implementation plan
**Branch / PR:** `feat/upgrade-cluster-api-v1.11.11` (#378), targeting release **v0.8.0**

---

## 1. Summary

Reshape the in-flight CAPI v1.11 upgrade so that, instead of shipping a v1beta1↔v1beta2
*bridge*, the release (**v0.8**) becomes a **pure v1beta2 provider**: it declares the
v1beta2 contract and removes every v1beta1 residue. The change spans the PR stack:

- **#378** (this branch → v0.8): the clean cut.
- **#380** (v1.12.10 bump): rebased onto #378.
- **#379** (e2e upgrade suite): rebased onto #380, with its upgrade block changed to the
  staged `v0.6.3 → v0.7.0 → v0.8` path that proves in-place upgrades survive the cut.

## 2. Background: the real version reality

The pre-existing `docs/capi-upgrade-plan.md` was written assuming *this branch* would ship
as v0.7 (a bridge), with the v1beta1 removal deferred to a later release. That assumption is
**wrong** — verified against the released tags:

| Release | CAPI | Declared contract | Implements v1beta2? | Status fields | Role |
|---|---|---|---|---|---|
| **v0.6.3** | v1.8.12 | v1beta1 | no | `status.ready` + v1beta1 `conditions` | pure v1beta1 |
| **v0.7.0** (released 2026-06-16) | v1.10.10 | v1beta1 | **no** | `status.ready` + v1beta1 `conditions` | pure v1beta1 — **not a bridge** |
| **v0.8** (this branch, before this change) | v1.11.11 | v1beta1 | yes | dual: `initialization.provisioned` + `deprecated.v1beta1.*` | expand/bridge |
| **v0.8** (after this change) | v1.11.11 | **v1beta2** | yes | only `initialization.provisioned` | **pure v1beta2** |

Decisive fact: **v0.7.0 shipped as pure v1beta1.** No dual-writing bridge exists in the
field. v0.8 (this branch) is therefore the first release to implement v1beta2 at all, and
the migration path we must protect is the staged **v0.6 → v0.7 → v0.8** in-place upgrade.

## 3. Decisions (and why)

1. **Clean cut in v0.8 (Option C), not a bridge.** CAPI v1.11 supports the v1beta2 contract
   and the guides *recommend* moving as soon as possible. We accept the cost (below) to get a
   pure v1beta2 provider in a single release rather than spreading it across v0.8/v0.9/v0.10.
2. **Existing clusters are NOT recreated.** Verified: no `IonosCloudMachineTemplate` content
   change → no template-hash change → no Machine rollout. `ProviderID *string → string` is
   safe for existing CRs (provisioned machines hold a concrete `ionos://…` value). A
   management-plane upgrade never touches running IONOS Cloud servers/nodes. A running
   Machine with a providerID + NodeRef is not torn down by a transient `provisioned=nil`.
3. **Accepted costs, documented as breaking:**
   - A brief not-ready window on the `v0.7 → v0.8` hop until the v0.8 controller reconciles
     each CR (self-heals on controller startup; strict MHCs could react during the window).
   - Removal of `.status.ready` / v1beta1 conditions is a **breaking change** for any
     external tooling reading them. No v1beta1 downconversion (minor — CAPIC CRDs are
     `v1alpha1`-only, so there is no v1beta1 API version to convert to).
   - Requires upgrading CAPI core to v1.11+ alongside CAPIC v0.8 (clusterctl contract gating).
4. **No hard e2e gate on #378.** #378 merges on `make test/verify/lint`. The empirical
   upgrade proof lives downstream in #379 (its natural home), not as a blocker on #378.
5. **Additive history on #378.** Add "declare v1beta2 / drop v1beta1" commits on top of the
   existing bridge commits (net tree = pure v1beta2). No interactive rewrite.
6. **Linear stack preserved.** `main ← #378 ← #380 ← #379`; rebase downstream onto the
   rewritten #378 in order.

## 4. SP1 — #378: the clean cut (primary deliverable)

Verified, complete change surface.

### 4.1 API types (`api/v1alpha1/`)
- `ionoscloudcluster_types.go`: remove `Status.Deprecated` field; remove
  `IonosCloudClusterDeprecatedStatus` and `IonosCloudClusterV1Beta1DeprecatedStatus`; remove
  `GetV1Beta1Conditions` / `SetV1Beta1Conditions`.
- `ionoscloudmachine_types.go`: same, plus remove `SetV1Beta1Ready`.
- **Keep** the condition-type constants (`IonosCloudClusterReady`, `MachineProvisionedCondition`)
  and the top-level `Conditions []metav1.Condition`. `clusterv1`
  (`sigs.k8s.io/cluster-api/api/core/v1beta2`) stays imported — still used by those constants
  (`clusterv1.ConditionType`). *Optional tidy:* retype the two constants to `string` (only
  ever stringified); do it only if it keeps lint/`ireturn` quiet — not required.

### 4.2 Controllers / services
- `internal/controller/ionoscloudcluster_controller.go`: delete the "set deprecated v1beta1
  ready" block (incl. its `//nolint:staticcheck`). Keep `Status.Initialization.Provisioned = new(true)`.
- `internal/service/cloud/server.go`: delete `ms.IonosMachine.SetV1Beta1Ready(true)`.

### 4.3 Scope plumbing (`scope/cluster.go`, `scope/machine.go`)
- Remove the `patch.WithOwnedV1Beta1Conditions{…}` option from both `PatchObject` calls; keep
  `patch.WithOwnedConditions{…}`.
- Simplify the `owned*Conditions` doc comments that reference the derived v1beta1 list.
- `clusterv1` remains used (`clusterv1.ReadyCondition` in `WithOwnedConditions` /
  `SetSummaryCondition`), so the import stays.

### 4.4 Generated (via `make manifests generate`)
- Both CRD YAMLs lose `status.deprecated`.
- `api/v1alpha1/zz_generated.deepcopy.go` loses the deprecated-type deepcopy funcs.

### 4.5 Metadata / docs
- `metadata.yaml`: add release series `0.8` with `contract: v1beta2` (0.7 and below stay
  `v1beta1`). This makes the *declared* contract match the CRD `cluster.x-k8s.io/v1beta2`
  annotation already present.
- `README.md`: add a v0.8 row to the compatibility matrix (v1beta2; CAPI **v1.11+** only,
  since a v1beta2-declared provider needs a v1beta2-capable core) and note v1beta1 is dropped.
- `docs/capi-upgrade-plan.md`: **rewrite** to the correct timeline (v0.6 v1.8/v1beta1 →
  v0.7 v1.10/v1beta1 → **v0.8 v1.11 pure v1beta2 clean cut**), replacing the obsolete
  bridge/Part-A/Part-B narrative.

### 4.6 Tests
- `api/v1alpha1/ionoscloudcluster_types_test.go` (≈ lines 41–45, 154, 175) and
  `ionoscloudmachine_types_test.go` (≈ line 542): remove the `Get/SetV1Beta1*` assertions.

### 4.7 What deliberately stays
- The v1beta2 implementation already on the branch (`Status.Initialization.Provisioned`,
  `[]metav1.Condition`, `conditions.Set`, CRD `cluster.x-k8s.io/v1beta2` annotation).
- No conversion webhook is added; CRDs remain `v1alpha1`.
- No CAPI version change beyond the v1.11.11 already on the branch.

### 4.8 Verification (SP1 gate)
- `make test` (unit + envtest integration), `make verify` (gen/tidy up to date), `make lint`.
- Inspection check: generated CRD YAMLs contain no `status.deprecated`; no remaining
  references to the removed symbols (`grep`).
- In-place upgrade behavior is validated by analysis here and empirically in SP3 — not gated
  on #378.

## 5. SP2 — #380: rebase the v1.12.10 bump

Rebase `feat/upgrade-cluster-api-v1.12.10` onto the rewritten #378. It is a pure dependency
bump with no v1beta1 code, so conflicts should be limited to `go.mod`/`go.sum` and any
regenerated artifacts. Re-run `make verify` after rebase. Detailed steps deferred to SP2's own
plan.

## 6. SP3 — #379: rebase + staged upgrade e2e

Rebase `feat/upgrade-e2e-suite` onto #380. Then change `test/e2e/upgrade_test.go`:
- **Replace** the existing direct `v0.6.3 → v0.6.99(local)` block (which, with the clean cut,
  silently became a direct v0.6 → v0.8 jump) with the **staged path**:
  `v0.6.3 → v0.7.0 → v0.8(local build)`. Because #379 rebases onto #380, the local v0.8
  provider is built against CAPI **v1.12.10**, so the final hop lands the core on v1.12.10
  (mirroring the existing suite design). This proves the supported in-place upgrade survives
  the pure-v1beta2 cut.
- Bump the local dev provider version tag (currently `v0.6.99`) to reflect v0.8 (e.g.
  `v0.8.99`), and update `metadata.yaml` / e2e config references accordingly.
- Reconcile e2e helpers that reference v1beta1 (`test/e2e/helpers/clusterhealth.go`,
  `conditions.go`): these legitimately assert *old* clusters during upgrade and stay, but
  their version comments/labels are updated for the staged path.
- Prerequisite satisfied: v0.6.3 and v0.7.0 are both published, so the block can `init`.

Detailed block layout deferred to SP3's own plan.

## 7. Risks & mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Transient not-ready window trips a strict MHC on `v0.7→v0.8` | low | Controller reconciles on startup; documented; SP3 e2e asserts cluster health post-upgrade |
| External tooling reads `.status.ready` and breaks | medium | Documented as a breaking change in README + release notes |
| Downstream rebase (#380/#379) conflicts | medium | Additive #378 history keeps diffs legible; rebase in order, re-run `make verify` |
| Removed symbol still referenced somewhere unseen | low | `grep` sweep in SP1 verification; `make build` fails fast otherwise |

## 8. Non-goals
- No conversion webhook; no CRD version bump.
- No v0.9 follow-up — the drop happens in v0.8.
- No CAPI bump beyond v1.11.11 on #378 (v1.12.10 is #380's job).
- No new e2e suite beyond changing the existing upgrade block in #379.

## 9. References
- CAPI v1.10→v1.11 provider migration guide
- CAPI v1.11→v1.12 provider migration guide: https://cluster-api.sigs.k8s.io/developer/providers/migrations/v1.11-to-v1.12
- Compatibility matrix: `README.md`
- Superseded planning doc (to be rewritten): `docs/capi-upgrade-plan.md`
