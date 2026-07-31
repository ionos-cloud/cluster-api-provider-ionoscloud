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
	"sigs.k8s.io/cluster-api/test/framework"

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
		// Without this, the framework defaults to its own package-private initScheme(),
		// which registers only the default CAPI schemes and not our infrav1 types — every
		// PostUpgrade hook that lists IonosCloudCluster/IonosCloudMachine objects against
		// this proxy would fail with "no kind registered for type v1alpha1...".
		KindManagementClusterNewClusterProxyFunc: func(name, kubeconfigPath string) framework.ClusterProxy {
			return framework.NewClusterProxy(name, kubeconfigPath, initScheme(), framework.WithMachineLogCollector(framework.DockerLogCollector{}))
		},
		PostNamespaceCreated: cloudEnv.createCredentialsSecretPNC,
		// InitWithKubernetesVersion is required by ClusterctlUpgradeSpec: it is the
		// Kubernetes version of the secondary (kind) management cluster the old
		// providers are installed into. Reuse the suite's configured version.
		InitWithKubernetesVersion: e2eConfig.MustGetVariable(KubernetesVersion),
		ControlPlaneMachineCount:  new(int64(1)),
	}
}

var _ = Describe("Should stage a v1.8-created cluster through v1.10 (real v0.7.0 release), v1.11 (v1beta1-conditions migration) and v1.12 (full v1beta2 migration)", Label("upgrade", "staged"), func() {
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
				BootstrapProviders:      []string{"kubeadm:v1.10.10"},
				ControlPlaneProviders:   []string{"kubeadm:v1.10.10"},
				InfrastructureProviders: []string{"ionoscloud:v0.7.0"},
				PostUpgrade:             helpers.AssertV1Beta1ClusterAndMachinesHealthy(ctx),
			},
			{
				// This hop lands the pure-v1beta2 v0.8 provider (local build) on a
				// v0.7.0-created cluster and proves both: the legacy v1beta1 conditions
				// (empty Reason) are backfilled, not rejected by the new schema, and the
				// CAPI core v1beta2 type migration itself completes cleanly.
				CoreProvider:            "cluster-api:v1.11.11",
				BootstrapProviders:      []string{"kubeadm:v1.11.11"},
				ControlPlaneProviders:   []string{"kubeadm:v1.11.11"},
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade: func(proxy framework.ClusterProxy, namespace, clusterName string) {
					helpers.AssertConditionsMigration(ctx)(proxy, namespace, clusterName)
					helpers.AssertCAPIV1Beta2Migration(ctx)(proxy, namespace, clusterName)
				},
			},
			{
				// Extra hop bumping CAPI core to v1.12.10. The v0.8.99 provider is
				// already built against v1.12.10, so this exercises the v1.11 -> v1.12
				// core upgrade and confirms the migrated conditions stay intact.
				CoreProvider:            "cluster-api:v1.12.10",
				BootstrapProviders:      []string{"kubeadm:v1.12.10"},
				ControlPlaneProviders:   []string{"kubeadm:v1.12.10"},
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
		}
		return in
	})
})

// Overlaps with the staged block's second hop on purpose: starting fresh at v1.10.10
// isolates a real v1.10->v1.11 bug from one caused by the staged block's carried-over state.
var _ = Describe("Should handle CAPI core v1beta2 type migration when upgrading from v1.10 through v1.11 to v1.12", Label("upgrade", "isolated-hop"), func() {
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
				BootstrapProviders:      []string{"kubeadm:v1.11.11"},
				ControlPlaneProviders:   []string{"kubeadm:v1.11.11"},
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
			{
				// Extra hop bumping CAPI core to v1.12.10 (the version v0.8.99 is built
				// against) to validate the v1.11 -> v1.12 core upgrade.
				CoreProvider:            "cluster-api:v1.12.10",
				BootstrapProviders:      []string{"kubeadm:v1.12.10"},
				ControlPlaneProviders:   []string{"kubeadm:v1.12.10"},
				InfrastructureProviders: []string{"ionoscloud:v0.8.99"},
				PostUpgrade:             helpers.AssertCAPIV1Beta2Migration(ctx),
			},
		}
		return in
	})
})
