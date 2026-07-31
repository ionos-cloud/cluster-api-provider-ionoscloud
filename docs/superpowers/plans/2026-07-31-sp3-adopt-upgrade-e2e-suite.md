# SP3 — Adopt the staged upgrade e2e suite onto pure-v1beta2 #380 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the staged-upgrade e2e suite (currently on `origin/feat/upgrade-e2e-suite`, mirrored locally as `e2e-suite-v1.12`, commit `a74c4eb`) onto the rewritten PR stack `main ← #378 (feat/upgrade-cluster-api-v1.11.11) ← #380 (feat/upgrade-cluster-api-v1.12.10)`, dropping the v1beta1-bridge production-code changes that branch predates the pure-v1beta2 clean cut (`docs/superpowers/specs/2026-07-17-drop-v1beta1-pure-v1beta2-design.md`) made obsolete, and restaging the "single-hop" upgrade block through the published `v0.7.0` release instead of jumping directly from `v0.6.3` to a local `v0.8` build.

**Architecture:** Branch `feat/upgrade-e2e-suite-v0.8` off `feat/upgrade-cluster-api-v1.12.10` (current tip `b1e7fc4`). Cherry-pick/hand-port only the e2e-only files from `e2e-suite-v1.12` (test data, helpers, config, CI, Makefile) — never its `internal/controller/`, `scope/*.go`, or `metadata.yaml` changes, which re-add the v1beta1 deprecated-status bridge that #378 intentionally removed. Rename the local dev provider tag from `v0.6.99` to `v0.8.99` throughout (it now builds v0.8, not a v0.6.x patch). Restructure the "single-hop" `Describe` block in `upgrade_test.go` to stage through the real `v0.7.0` release, matching the already-staged "multi-hop" block.

**Tech Stack:** Ginkgo/Gomega e2e suite (`test/e2e/`, build tag `e2e`), `sigs.k8s.io/cluster-api/test/framework` `ClusterctlUpgradeSpec`, clusterctl config (`test/e2e/config/ionoscloud.yaml`), kind-based secondary management cluster.

## Global Constraints

- Do not reintroduce `Status.Deprecated` / `SetV1Beta1Ready` / `WithOwnedV1Beta1Conditions` anywhere in production code (`internal/`, `scope/`, `api/v1alpha1/`) — that is the SP1 decision this branch must not undo.
- `test/e2e/config/ionoscloud.yaml`'s `v1.11.11` and `v1.12.10` core/bootstrap/control-plane entries must declare `contract: v1beta2` (not `v1beta1` as `e2e-suite-v1.12` has them — that branch predates the clean cut). `v1.10.10` and `v1.8.12` stay `contract: v1beta1` (real upstream fact, unrelated to our provider).
- The local dev infra-provider version name is `v0.8.99` (was `v0.6.99`), everywhere it's referenced: `test/e2e/config/ionoscloud.yaml` and `test/e2e/upgrade_test.go`.
- Every new/ported file must build under `go build -tags e2e ./...` and `go vet -tags e2e ./...`.
- Base branch for all work: `feat/upgrade-cluster-api-v1.12.10` at commit `b1e7fc4` (SP2, already rebased onto pure-v1beta2 #378).

---

### Task 1: Branch setup and e2e data/config file ports

**Files:**
- Create: `test/e2e/data/shared/v1.8/metadata.yaml`
- Create: `test/e2e/data/shared/v1.10/metadata.yaml`
- Modify: `test/e2e/config/ionoscloud.yaml`

**Interfaces:** none (data/config only).

- [ ] **Step 1: Create the branch**

```bash
git checkout -b feat/upgrade-e2e-suite-v0.8 feat/upgrade-cluster-api-v1.12.10
```

- [ ] **Step 2: Add the v1.8 CAPI core metadata override**

Create `test/e2e/data/shared/v1.8/metadata.yaml`:

```yaml
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
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

- [ ] **Step 3: Add the v1.10 CAPI core metadata override**

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

Note: `test/e2e/data/shared/v1.12/metadata.yaml` already exists on this branch (added in SP2) with `minor: 12` and `minor: 11` both `contract: v1beta2` — do not recreate it.

- [ ] **Step 4: Replace `test/e2e/config/ionoscloud.yaml` with the merged version**

Register the v1.8/v1.10/v1.12 core+bootstrap+control-plane provider versions (v1.12 as `v1beta2`, matching this branch's contract, not `e2e-suite-v1.12`'s stale `v1beta1`), add the `v0.6.3`/`v0.7.0` published infra-provider versions, and rename the local dev entry to `v0.8.99`:

```yaml
images:
  - name: ghcr.io/ionos-cloud/cluster-api-provider-ionoscloud:e2e

providers:
  - name: cluster-api
    type: CoreProvider
    versions:
    - name: "v1.12.10"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.12.10/core-components.yaml"
      type: url
      contract: v1beta2
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.12/metadata.yaml"
    - name: "v1.11.11"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.11/core-components.yaml"
      type: url
      contract: v1beta2
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.11/metadata.yaml"
    - name: "v1.10.10"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/core-components.yaml"
      type: url
      contract: v1beta1
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.10/metadata.yaml"
    - name: "v1.8.12"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.8.12/core-components.yaml"
      type: url
      contract: v1beta1
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.8/metadata.yaml"
  - name: kubeadm
    type: BootstrapProvider
    versions:
    - name: "v1.12.10"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.12.10/bootstrap-components.yaml"
      type: url
      contract: v1beta2
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.12/metadata.yaml"
    - name: "v1.11.11"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.11/bootstrap-components.yaml"
      type: url
      contract: v1beta2
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.11/metadata.yaml"
    - name: "v1.10.10"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/bootstrap-components.yaml"
      type: url
      contract: v1beta1
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.10/metadata.yaml"
    - name: "v1.8.12"
      value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.8.12/bootstrap-components.yaml"
      type: url
      contract: v1beta1
      replacements:
        - old: --metrics-addr=127.0.0.1:8080
          new: --metrics-addr=:8443
      files:
        - sourcePath: "../data/shared/v1.8/metadata.yaml"
  - name: kubeadm
    type: ControlPlaneProvider
    versions:
      - name: "v1.12.10"
        value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.12.10/control-plane-components.yaml"
        type: url
        contract: v1beta2
        replacements:
          - old: --metrics-addr=127.0.0.1:8080
            new: --metrics-addr=:8443
        files:
          - sourcePath: "../data/shared/v1.12/metadata.yaml"
      - name: "v1.11.11"
        value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.11.11/control-plane-components.yaml"
        type: url
        contract: v1beta2
        replacements:
          - old: --metrics-addr=127.0.0.1:8080
            new: --metrics-addr=:8443
        files:
          - sourcePath: "../data/shared/v1.11/metadata.yaml"
      - name: "v1.10.10"
        value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/control-plane-components.yaml"
        type: url
        contract: v1beta1
        replacements:
          - old: --metrics-addr=127.0.0.1:8080
            new: --metrics-addr=:8443
        files:
          - sourcePath: "../data/shared/v1.10/metadata.yaml"
      - name: "v1.8.12"
        value: "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.8.12/control-plane-components.yaml"
        type: url
        contract: v1beta1
        replacements:
          - old: --metrics-addr=127.0.0.1:8080
            new: --metrics-addr=:8443
        files:
          - sourcePath: "../data/shared/v1.8/metadata.yaml"
  - name: in-cluster
    type: IPAMProvider
    versions:
      - name: v1.0.0
        # Use manifest from source files
        value: "https://github.com/kubernetes-sigs/cluster-api-ipam-provider-in-cluster/releases/download/v1.0.0/ipam-components.yaml"
        type: url
        contract: v1beta1
        files:
          - sourcePath: "../data/shared/capi-ipam/v1.0/metadata.yaml"
        replacements:
          - old: "imagePullPolicy: Always"
            new: "imagePullPolicy: IfNotPresent"
  - name: ionoscloud
    type: InfrastructureProvider
    versions:
      - name: v0.6.3
        value: "https://github.com/ionos-cloud/cluster-api-provider-ionoscloud/releases/download/v0.6.3/infrastructure-components.yaml"
        type: url
        contract: v1beta1
        files:
          - sourcePath: "../data/infrastructure-ionoscloud/cluster-template.yaml"
      - name: v0.7.0
        value: "https://github.com/ionos-cloud/cluster-api-provider-ionoscloud/releases/download/v0.7.0/infrastructure-components.yaml"
        type: url
        contract: v1beta1
        files:
          # The published v0.7.0 metadata.yaml is missing its own "0.7" releaseSeries
          # entry, so clusterctl can't resolve the version's contract and init fails.
          # Override it with the repo metadata.yaml, which declares 0.7 as v1beta1.
          - sourcePath: "../../../metadata.yaml"
          - sourcePath: "../data/infrastructure-ionoscloud/cluster-template.yaml"
      - name: v0.8.99
        value: "../../../config/default"
        replacements:
          - old: ghcr.io/ionos-cloud/cluster-api-provider-ionoscloud:dev
            new: ghcr.io/ionos-cloud/cluster-api-provider-ionoscloud:e2e
          - old: "--leader-elect"
            new: "--leader-elect\n        - --insecure-diagnostics"
        files:
          - sourcePath: "../../../metadata.yaml"
          - sourcePath: "../data/infrastructure-ionoscloud/cluster-template.yaml"
          - sourcePath: "../data/infrastructure-ionoscloud/cluster-template-ipam.yaml"
          - sourcePath: "../data/infrastructure-ionoscloud/cluster-template-image-selector.yaml"
variables:
  # Default variables for the e2e test; those values could be overridden via env variables, thus
  # allowing the same e2e config file to be reused in different Prow jobs e.g. each one with a K8s version permutation.
  # The following Kubernetes versions should be the latest versions with already published kindest/node images.
  # This avoids building node images in the default case which improves the test duration significantly.
  KUBERNETES_VERSION: "v1.30.6"
  IP_FAMILY: "ipv4"
  CNI: "./data/cni/calico.yaml"
  KUBETEST_CONFIGURATION: "./data/kubetest/conformance.yaml"
  CLUSTER_NAME: "e2e-cluster-${RANDOM}"
  # Enabling the feature flags by setting the env variables.
  # Note: EXP_CLUSTER_RESOURCE_SET is enabled per default with CAPI v1.7.0.
  # We still have to enable them here for clusterctl upgrade tests that use older versions.
  EXP_CLUSTER_RESOURCE_SET: "true"
  IONOSCLOUD_MACHINE_MEMORY_MB: ""
  IONOSCLOUD_MACHINE_NUM_CORES: ""
  IONOSCLOUD_MACHINE_SSH_KEYS: ""
  IONOSCLOUD_DATACENTER_ID: ""
  CONTROL_PLANE_ENDPOINT_IP: ""
  CONTROL_PLANE_ENDPOINT_LOCATION: ""
  IONOSCLOUD_MACHINE_IMAGE_ID: ""

intervals:
  default/wait-controllers: [ "30m", "10s" ]
  default/wait-cluster: [ "30m", "10s" ]
  default/wait-control-plane: [ "30m", "10s" ]
  default/wait-worker-nodes: [ "30m", "10s" ]
  default/wait-delete-cluster: [ "30m", "10s" ]
  default/wait-nodes-ready: ["30m", "10s"]
  scale/wait-cluster: ["30m", "10s"]
  scale/wait-control-plane: ["30m", "10s"]
  scale/wait-worker-nodes: ["30m", "10s"]
```

- [ ] **Step 5: Sanity-check the YAML parses and commit**

Run: `python3 -c "import yaml; yaml.safe_load(open('test/e2e/config/ionoscloud.yaml'))"`
Expected: no output (valid YAML). If `python3`/`yaml` isn't available, skip — the e2e build step in Task 5 will catch a malformed file anyway.

```bash
git add test/e2e/config/ionoscloud.yaml test/e2e/data/shared/v1.8/metadata.yaml test/e2e/data/shared/v1.10/metadata.yaml
git commit -m "test(e2e): register CAPI v1.8.12/v1.10.10/v1.12.10 and CAPIC v0.6.3/v0.7.0 for upgrade tests

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Port the upgrade-assertion helpers, stripped of v1beta1 residue

**Files:**
- Create: `test/e2e/helpers/conditions.go`
- Create: `test/e2e/helpers/clusterhealth.go`

**Interfaces:**
- Produces: `helpers.AssertConditionsMigration(ctx context.Context) func(framework.ClusterProxy, string, string)`, `helpers.AssertCAPIV1Beta2Migration(ctx context.Context) func(framework.ClusterProxy, string, string)`, `helpers.AssertV1Beta1ClusterAndMachinesHealthy(ctx context.Context) func(framework.ClusterProxy, string, string)` — all consumed by Task 3's `upgrade_test.go`.

- [ ] **Step 1: Create `test/e2e/helpers/conditions.go`**

This is `e2e-suite-v1.12`'s version of the file with the `deprecatedReady`/`Status.Deprecated.V1Beta1` assertion removed — that field no longer exists after SP1's clean cut, and there is nothing left to migrate *to* it, so the check is dropped rather than adapted:

```go
//go:build e2e

/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helpers

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/cluster-api/test/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"

	. "github.com/onsi/gomega"
)

// AssertConditionsMigration returns a PostUpgrade hook that verifies CAPIC resources
// carry the v1beta2 status fields after upgrading from v0.6.x to v0.8+.
func AssertConditionsMigration(ctx context.Context) func(framework.ClusterProxy, string, string) {
	return func(proxy framework.ClusterProxy, namespace, _ string) {
		c := proxy.GetClient()

		clusters := &infrav1.IonosCloudClusterList{}
		Expect(c.List(ctx, clusters, runtimeclient.InNamespace(namespace))).To(Succeed())
		Expect(clusters.Items).NotTo(BeEmpty(),
			"expected at least one IonosCloudCluster in namespace %s", namespace)

		for i := range clusters.Items {
			assertClusterConditionsMigrated(&clusters.Items[i])
		}

		machines := &infrav1.IonosCloudMachineList{}
		Expect(c.List(ctx, machines, runtimeclient.InNamespace(namespace))).To(Succeed())
		Expect(machines.Items).NotTo(BeEmpty(),
			"expected at least one IonosCloudMachine in namespace %s", namespace)

		for i := range machines.Items {
			assertMachineConditionsMigrated(&machines.Items[i])
		}
	}
}

// AssertCAPIV1Beta2Migration returns a PostUpgrade hook that verifies CAPI core
// v1beta2 types are active and the Cluster is available after upgrading to v1.11+.
func AssertCAPIV1Beta2Migration(ctx context.Context) func(framework.ClusterProxy, string, string) {
	return func(proxy framework.ClusterProxy, namespace, clusterName string) {
		c := proxy.GetClient()

		// Assert the clusters.cluster.x-k8s.io CRD storage migration to v1beta2 has completed
		// (status.storedVersions == [v1beta2], not just spec.versions[].storage).
		framework.ValidateCRDMigration(ctx, proxy, namespace, clusterName,
			func(crd apiextensionsv1.CustomResourceDefinition) bool {
				return crd.Name == "clusters.cluster.x-k8s.io"
			},
			clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName))

		// Assert Cluster has Available=True.
		framework.VerifyClusterAvailable(ctx, framework.VerifyClusterAvailableInput{
			Getter:    proxy.GetClient(),
			Name:      clusterName,
			Namespace: namespace,
		})

		// Assert no continuous reconcile loop (resource versions stable).
		framework.ValidateResourceVersionStable(ctx, framework.ValidateResourceVersionStableInput{
			ClusterProxy:             proxy,
			Namespace:                namespace,
			OwnerGraphFilterFunction: clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
		})

		// Assert CAPIC resources also carry the v1beta2 fields (upgrade included CAPIC too).
		clusters := &infrav1.IonosCloudClusterList{}
		Expect(c.List(ctx, clusters, runtimeclient.InNamespace(namespace))).To(Succeed())
		for i := range clusters.Items {
			assertClusterConditionsMigrated(&clusters.Items[i])
		}

		machines := &infrav1.IonosCloudMachineList{}
		Expect(c.List(ctx, machines, runtimeclient.InNamespace(namespace))).To(Succeed())
		for i := range machines.Items {
			assertMachineConditionsMigrated(&machines.Items[i])
		}
	}
}

func assertClusterConditionsMigrated(cluster *infrav1.IonosCloudCluster) {
	assertConditionsMigrated("IonosCloudCluster", cluster.Name,
		cluster.Status.Initialization.Provisioned, cluster.Status.Conditions)
}

func assertMachineConditionsMigrated(machine *infrav1.IonosCloudMachine) {
	assertConditionsMigrated("IonosCloudMachine", machine.Name,
		machine.Status.Initialization.Provisioned, machine.Status.Conditions)
}

// assertConditionsMigrated verifies the v1beta2 status shape shared by IonosCloudCluster
// and IonosCloudMachine after upgrading from v0.6.x/v0.7.x to v0.8+, including that any
// condition carried over from the old v1beta1 conditions API (empty Reason) was backfilled
// by api/v1alpha1.BackfillLegacyConditionReasons rather than rejected by the new schema.
func assertConditionsMigrated(kind, name string, provisioned *bool, conditions []metav1.Condition) {
	// status.initialization.provisioned must be set and true.
	Expect(provisioned).NotTo(BeNil(),
		"%s %s: status.initialization.provisioned must be set after upgrade", kind, name)
	Expect(*provisioned).To(BeTrue(),
		"%s %s: status.initialization.provisioned must be true after upgrade", kind, name)

	// status.conditions (metav1.Condition) must be non-empty.
	Expect(conditions).NotTo(BeEmpty(),
		"%s %s: status.conditions must be non-empty after upgrade", kind, name)
	hasTrue := false
	for _, cond := range conditions {
		if cond.Status == metav1.ConditionTrue {
			hasTrue = true
		}
		Expect(cond.Reason).NotTo(BeEmpty(),
			"%s %s: condition %q must carry a non-empty Reason after upgrade "+
				"(legacy v1beta1 conditions must be backfilled, see api/v1alpha1.BackfillLegacyConditionReasons)",
			kind, name, cond.Type)
	}
	Expect(hasTrue).To(BeTrue(),
		"%s %s: at least one metav1.Condition must have Status=True", kind, name)
}
```

- [ ] **Step 2: Create `test/e2e/helpers/clusterhealth.go` verbatim**

This file only reads the CAPI-core `clusterv1beta1.Cluster`/`MachineList` `.Status.Phase` — nothing here touches our provider's removed fields, so it ports unchanged:

```go
//go:build e2e

/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helpers

import (
	"context"

	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // only contract CAPI v1.10.x serves
	"sigs.k8s.io/cluster-api/test/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/gomega"
)

// AssertV1Beta1ClusterAndMachinesHealthy is a PostUpgrade hook checking Cluster/Machine
// Status.Phase only — safe to use when the upgrade target doesn't serve v1beta2 yet.
func AssertV1Beta1ClusterAndMachinesHealthy(ctx context.Context) func(framework.ClusterProxy, string, string) {
	return func(proxy framework.ClusterProxy, namespace, clusterName string) {
		c := proxy.GetClient()

		cluster := &clusterv1beta1.Cluster{}
		Expect(c.Get(ctx, runtimeclient.ObjectKey{Namespace: namespace, Name: clusterName}, cluster)).To(Succeed())
		Expect(cluster.Status.Phase).To(Equal(string(clusterv1beta1.ClusterPhaseProvisioned)),
			"Cluster %s should be in the Provisioned phase", clusterName)

		machines := &clusterv1beta1.MachineList{}
		Expect(c.List(ctx, machines, runtimeclient.InNamespace(namespace),
			runtimeclient.MatchingLabels{clusterv1beta1.ClusterNameLabel: clusterName})).To(Succeed())
		Expect(machines.Items).NotTo(BeEmpty(), "expected at least one Machine for cluster %s", clusterName)

		for _, m := range machines.Items {
			Expect(m.Status.Phase).To(Equal(string(clusterv1beta1.MachinePhaseRunning)),
				"Machine %s should be in the Running phase", m.Name)
		}
	}
}
```

- [ ] **Step 3: Compile-check and commit**

Run: `go build -tags e2e ./test/e2e/...`
Expected: builds clean (both helpers only reference types that already exist on this branch: `infrav1.IonosCloudClusterList/IonosCloudMachineList`, `Status.Initialization.Provisioned`, `Status.Conditions` — none of the removed `Status.Deprecated` fields).

```bash
git add test/e2e/helpers/conditions.go test/e2e/helpers/clusterhealth.go
git commit -m "test(e2e): add upgrade PostUpgrade assertion helpers for conditions migration

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Port and restage `upgrade_test.go`

**Files:**
- Create: `test/e2e/upgrade_test.go`

**Interfaces:**
- Consumes: `helpers.AssertConditionsMigration`, `helpers.AssertCAPIV1Beta2Migration`, `helpers.AssertV1Beta1ClusterAndMachinesHealthy` (Task 2); `e2eConfig`, `clusterctlConfigPath`, `bootstrapClusterProxy`, `artifactFolder`, `skipCleanup`, `cloudEnv.createCredentialsSecretPNC`, `KubernetesVersion` (all pre-existing in `test/e2e/suite_test.go`/`common.go`, unchanged by this plan).

- [ ] **Step 1: Create `test/e2e/upgrade_test.go`**

Three blocks, ported from `e2e-suite-v1.12`'s `upgrade_test.go` with two changes: every `v0.6.99` local-dev infra-provider reference becomes `v0.8.99`, and the first ("single-hop") block is restaged to go through the published `v0.7.0` release instead of jumping straight from `v0.6.3` to the local build — the direct jump would otherwise silently skip proving the `v0.6.3 → v0.7.0` hop that `BackfillLegacyConditionReasons` (`api/v1alpha1/conditions_migration.go`) exists to protect:

```go
//go:build e2e

/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	capie2e "sigs.k8s.io/cluster-api/test/e2e"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/test/e2e/helpers"

	. "github.com/onsi/ginkgo/v2"
)

// upgradeSpecBaseInput returns the ClusterctlUpgradeSpecInput fields shared by all
// upgrade test blocks. Each block overrides init/upgrade versions and PostUpgrade.
func upgradeSpecBaseInput() capie2e.ClusterctlUpgradeSpecInput {
	return capie2e.ClusterctlUpgradeSpecInput{
		E2EConfig:                   e2eConfig,
		ClusterctlConfigPath:        clusterctlConfigPath,
		BootstrapClusterProxy:       bootstrapClusterProxy,
		ArtifactFolder:              artifactFolder,
		SkipCleanup:                 skipCleanup,
		UseKindForManagementCluster: true,
		PostNamespaceCreated:        cloudEnv.createCredentialsSecretPNC,
		// InitWithKubernetesVersion is required by ClusterctlUpgradeSpec: it is the
		// Kubernetes version of the secondary (kind) management cluster the old
		// providers are installed into. Reuse the suite's configured version.
		InitWithKubernetesVersion: e2eConfig.MustGetVariable(KubernetesVersion),
		ControlPlaneMachineCount:  new(int64(1)),
	}
}

var _ = Describe("Should migrate CAPIC conditions from v1beta1 to metav1.Condition on the staged v0.6.3 -> v0.7.0 -> v0.8 provider upgrade", Label("upgrade", "single-hop"), func() {
	capie2e.ClusterctlUpgradeSpec(ctx, func() capie2e.ClusterctlUpgradeSpecInput {
		in := upgradeSpecBaseInput()
		in.InitWithBinary = "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.8.12/clusterctl-{OS}-{ARCH}"
		in.InitWithCoreProvider = "cluster-api:v1.8.12"
		// Pin bootstrap/control-plane to the same version as core. Without this they
		// fall back to the latest matching the "*" contract (v1.12.10), which would
		// mismatch the v1.8.12 core provider and fail clusterctl init.
		in.InitWithBootstrapProviders = []string{"kubeadm:v1.8.12"}
		in.InitWithControlPlaneProviders = []string{"kubeadm:v1.8.12"}
		in.InitWithInfrastructureProviders = []string{"ionoscloud:v0.6.3"}
		in.Upgrades = []capie2e.ClusterctlUpgradeSpecInputUpgrade{
			{
				// Real released hop: proves v0.6.3 (v1.8.12, v1beta1) -> v0.7.0 (v1.10.10,
				// still pure v1beta1 — see docs/capi-upgrade-plan.md) survives before the
				// pure-v1beta2 v0.8 cut is applied.
				WithBinary:              "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/clusterctl-{OS}-{ARCH}",
				CoreProvider:            "cluster-api:v1.10.10",
				InfrastructureProviders: []string{"ionoscloud:v0.7.0"},
				PostUpgrade:             helpers.AssertV1Beta1ClusterAndMachinesHealthy(ctx),
			},
			{
				// This hop lands the pure-v1beta2 v0.8 provider (local build) on a
				// v0.7.0-created cluster and proves the legacy v1beta1 conditions
				// (empty Reason) are backfilled, not rejected by the new schema.
				CoreProvider:            "cluster-api:v1.11.11",
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertConditionsMigration(ctx),
			},
			{
				// Extra hop bumping CAPI core to v1.12.10. The v0.8.99 provider is
				// already built against v1.12.10, so this exercises the v1.11 -> v1.12
				// core upgrade and confirms the migrated conditions stay intact.
				CoreProvider:            "cluster-api:v1.12.10",
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
		}
		return in
	})
})

var _ = Describe("Should keep a v1.8-created cluster reconcilable after staging the upgrade through v1.10 and v1.11 before reaching v1.12", Label("upgrade", "multi-hop"), func() {
	capie2e.ClusterctlUpgradeSpec(ctx, func() capie2e.ClusterctlUpgradeSpecInput {
		in := upgradeSpecBaseInput()
		in.InitWithBinary = "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.8.12/clusterctl-{OS}-{ARCH}"
		in.InitWithCoreProvider = "cluster-api:v1.8.12"
		in.InitWithBootstrapProviders = []string{"kubeadm:v1.8.12"}
		in.InitWithControlPlaneProviders = []string{"kubeadm:v1.8.12"}
		in.InitWithInfrastructureProviders = []string{"ionoscloud:v0.6.3"}
		in.Upgrades = []capie2e.ClusterctlUpgradeSpecInputUpgrade{
			{
				// Landing point this block validates. Must not be the last entry (v1.10 has
				// no v1beta2 API). WithBinary is required too: the in-process v1.12.10
				// clusterctl client refuses to move the core provider anywhere but v1beta2.
				WithBinary:              "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/clusterctl-{OS}-{ARCH}",
				CoreProvider:            "cluster-api:v1.10.10",
				InfrastructureProviders: []string{"ionoscloud:v0.7.0"},
				PostUpgrade:             helpers.AssertV1Beta1ClusterAndMachinesHealthy(ctx),
			},
			{
				// Intermediate v1beta2 landing at v1.11 before the final v1.12 hop.
				CoreProvider:            "cluster-api:v1.11.11",
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
			{
				// Final hop bumping CAPI core to v1.12.10 (the version v0.8.99 is built
				// against). Also satisfies the framework's mandatory last-step v1beta2 checks.
				CoreProvider:            "cluster-api:v1.12.10",
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
		}
		return in
	})
})

// Overlaps with the staged block's second hop on purpose: starting fresh at v1.10.10
// isolates a real v1.10->v1.11 bug from one caused by the staged block's carried-over state.
var _ = Describe("Should handle CAPI core v1beta2 type migration when upgrading from v1.10 through v1.11 to v1.12", Label("upgrade", "single-hop"), func() {
	capie2e.ClusterctlUpgradeSpec(ctx, func() capie2e.ClusterctlUpgradeSpecInput {
		in := upgradeSpecBaseInput()
		in.InitWithBinary = "https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.10/clusterctl-{OS}-{ARCH}"
		in.InitWithCoreProvider = "cluster-api:v1.10.10"
		// Pin bootstrap/control-plane to match the v1.10.10 core provider (see block above).
		in.InitWithBootstrapProviders = []string{"kubeadm:v1.10.10"}
		in.InitWithControlPlaneProviders = []string{"kubeadm:v1.10.10"}
		in.InitWithInfrastructureProviders = []string{"ionoscloud:v0.7.0"}
		in.Upgrades = []capie2e.ClusterctlUpgradeSpecInputUpgrade{
			{
				CoreProvider:            "cluster-api:v1.11.11",
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
			{
				// Extra hop bumping CAPI core to v1.12.10 (the version v0.8.99 is built
				// against) to validate the v1.11 -> v1.12 core upgrade.
				CoreProvider:            "cluster-api:v1.12.10",
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
		}
		return in
	})
})
```

- [ ] **Step 2: Compile-check and commit**

Run: `go build -tags e2e ./test/e2e/...` and `go vet -tags e2e ./test/e2e/...`
Expected: both clean.

```bash
git add test/e2e/upgrade_test.go
git commit -m "test(e2e): add staged upgrade suite for conditions schema and CAPI v1beta2 migration

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: CI wiring — Makefile target and scheduled workflow

**Files:**
- Modify: `Makefile`
- Create: `.github/workflows/e2e-upgrade.yaml`

**Interfaces:** none (build/CI plumbing only).

- [ ] **Step 1: Add the `test-e2e-upgrade` Makefile target and exclude `upgrade` from the default e2e run**

In `Makefile`, find:

```makefile
GINKGO_LABEL ?= "!Conformance"
```

Replace with:

```makefile
# Exclude the "upgrade" label by default: the provider-upgrade suite is heavy
# (spins up its own kind management cluster and installs old providers per block)
# and is run separately via `make test-e2e-upgrade`.
GINKGO_LABEL ?= "!Conformance && !upgrade"
```

Then, immediately after the existing `test-e2e:` target's recipe (the block ending in `-e2e.skip-resource-cleanup=$(SKIP_RESOURCE_CLEANUP) -e2e.use-existing-cluster=$(USE_EXISTING_CLUSTER)`), add:

```makefile
.PHONY: test-e2e-upgrade
test-e2e-upgrade: docker-build-e2e ## Run only the provider-upgrade e2e tests (Label "upgrade")
	CGO_ENABLED=1 go run github.com/onsi/ginkgo/v2/ginkgo -v --trace \
	-poll-progress-after=$(GINKGO_POLL_PROGRESS_AFTER) \
	-poll-progress-interval=$(GINKGO_POLL_PROGRESS_INTERVAL) --tags=e2e --fail-fast \
	--nodes=1 --label-filter="upgrade" --timeout=$(GINKGO_TIMEOUT) --no-color=$(GINKGO_NOCOLOR) \
	--output-dir="$(ARTIFACTS)" --junit-report="junit.e2e_upgrade.1.xml" $(GINKGO_ARGS) $(ROOT_DIR)/$(TEST_DIR)/e2e -- \
	-e2e.artifacts-folder="$(ARTIFACTS)" -e2e.config="$(E2E_CONF_FILE)" \
	-e2e.skip-resource-cleanup=$(SKIP_RESOURCE_CLEANUP) -e2e.use-existing-cluster=$(USE_EXISTING_CLUSTER)
```

- [ ] **Step 2: Create the scheduled upgrade-e2e workflow**

Create `.github/workflows/e2e-upgrade.yaml`:

```yaml
name: End-to-end upgrade tests
on:
  schedule:
    # Weekly, matching the conformance suite's cadence. Offset to Tuesday to
    # avoid contending for IONOS Cloud resources with conformance (Mondays).
    - cron: "0 5 * * 2"
  workflow_dispatch: {}
jobs:
  e2e-upgrade:
    runs-on: ubuntu-latest
    environment: e2e
    env:
      IONOS_TOKEN: ${{ secrets.IONOS_TOKEN }}
      IONOSCLOUD_MACHINE_IMAGE_ID: ${{ vars.IONOSCLOUD_MACHINE_IMAGE_ID }}
      CONTROL_PLANE_ENDPOINT_LOCATION: ${{ vars.CONTROL_PLANE_ENDPOINT_LOCATION }}
      IONOSCLOUD_CUBE_TEMPLATE_ID: ${{ vars.IONOSCLOUD_CUBE_TEMPLATE_ID }}
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go environment
        uses: actions/setup-go@v5
        with:
          go-version-file: "go.mod"

      - name: Run e2e upgrade tests
        run: make test-e2e-upgrade

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        if: success() || failure()
        with:
          name: logs-upgrade
          path: _artifacts
          retention-days: 7
```

- [ ] **Step 3: Verify the Makefile parses and commit**

Run: `make -n test-e2e-upgrade | head -5`
Expected: prints the planned `ginkgo` command line without executing it (no error about an unknown target).

```bash
git add Makefile .github/workflows/e2e-upgrade.yaml
git commit -m "ci(e2e): add scheduled upgrade e2e workflow and test-e2e-upgrade target

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Full verification gate

**Files:** none created; verification only.

**Interfaces:** none.

- [ ] **Step 1: Confirm no v1beta1 production-code residue was reintroduced**

Run: `grep -rniE "SetV1Beta1Ready|GetV1Beta1Conditions|SetV1Beta1Conditions|WithOwnedV1Beta1Conditions|Status.Deprecated" --include="*.go" internal/ scope/ api/ | grep -v zz_generated`
Expected: no matches. If any appear, the wrong hunks from `e2e-suite-v1.12` were ported — undo them, keeping only the `test/e2e/`, `Makefile`, and `.github/` changes from this plan.

- [ ] **Step 2: Build everything, including the e2e build tag**

Run: `go build ./... && go build -tags e2e ./...`
Expected: both succeed with no output.

- [ ] **Step 3: Vet everything, including the e2e build tag**

Run: `go vet ./... && go vet -tags e2e ./...`
Expected: both succeed with no output.

- [ ] **Step 4: Run the full non-e2e test suite**

Run: `make unit-test`
Expected: all packages `ok`, race detector clean.

- [ ] **Step 5: Verify generated artifacts and go.mod/go.sum are still up to date**

Run: `make verify`
Expected: `verify-gen` and `verify-tidy` pass (no diff). This task doesn't touch generated code or dependencies, so it should be a no-op; if it reports a diff, something in Task 1-4 accidentally changed a non-e2e file.

- [ ] **Step 6: Confirm the new branch sits cleanly on #380**

Run: `git log --oneline feat/upgrade-cluster-api-v1.12.10..feat/upgrade-e2e-suite-v0.8`
Expected: exactly 4 commits (Tasks 1-4, in order), all authored on top of `b1e7fc4`.

---

## Self-Review

**Spec coverage** (against `docs/superpowers/specs/2026-07-17-drop-v1beta1-pure-v1beta2-design.md` §6):
- "Rebase `feat/upgrade-e2e-suite` onto #380" → done via fresh branch off `feat/upgrade-cluster-api-v1.12.10`, porting only e2e-scoped files (Tasks 1-4), explicitly excluding the v1beta1-bridge production-code hunks. ✅
- "Replace the direct `v0.6.3 → v0.6.99(local)` block with the staged path `v0.6.3 → v0.7.0 → v0.8(local build)`" → Task 3, first `Describe` block. ✅
- "Bump the local dev provider version tag... to `v0.8.99`" → Task 1 Step 4 (config) and Task 3 (test file), every reference renamed. ✅
- "Reconcile e2e helpers that reference v1beta1" → Task 2: `clusterhealth.go` ports unchanged (only touches CAPI-core v1beta1 types, which still exist and are legitimately asserted mid-upgrade); `conditions.go` drops the `Status.Deprecated.V1Beta1` assertion, replaced with a Reason-non-empty check that documents *why* (backfill, not removal-without-replacement). ✅
- "Prerequisite satisfied: v0.6.3 and v0.7.0 are both published" → verified as an assumption carried from the existing suite design (`docs/superpowers/specs/2026-06-16-upgrade-e2e-suite-design.md`), not re-derived here.

**Deviation from the source branch, called out explicitly:** `e2e-suite-v1.12`'s `test/e2e/config/ionoscloud.yaml` declares `v1.11.11`/`v1.12.10` core as `contract: v1beta1` — that branch predates the 2026-07-17 pure-v1beta2 decision. This plan keeps them `v1beta2`, matching the already-committed fix on `feat/upgrade-cluster-api-v1.11.11` (commit `b5db700`) and `feat/upgrade-cluster-api-v1.12.10` (commit `b1e7fc4`).

**Placeholder scan:** No TBD/TODO; every file-creation step shows complete file content; every command has an expected result. ✅

**Type consistency:** `helpers.AssertConditionsMigration` / `AssertCAPIV1Beta2Migration` / `AssertV1Beta1ClusterAndMachinesHealthy` signatures in Task 2 match their call sites in Task 3 exactly (`func(context.Context) func(framework.ClusterProxy, string, string)`). Infra-provider version string `v0.8.99` is consistent across Task 1's config and Task 3's test file. ✅
