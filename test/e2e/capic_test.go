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
	"os"

	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	capie2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/test/e2e/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Quickstart: Should be able to create a cluster with 3 control-plane and 2 worker nodes", func() {
	capie2e.QuickStartSpec(ctx, func() capie2e.QuickStartSpecInput {
		return capie2e.QuickStartSpecInput{
			E2EConfig:                e2eConfig,
			ClusterctlConfigPath:     clusterctlConfigPath,
			BootstrapClusterProxy:    bootstrapClusterProxy,
			ArtifactFolder:           artifactFolder,
			SkipCleanup:              skipCleanup,
			ControlPlaneMachineCount: ptr.To[int64](3),
			WorkerMachineCount:       ptr.To[int64](2),
			PostNamespaceCreated:     cloudEnv.createCredentialsSecretPNC,
			PostMachinesProvisioned: func(proxy framework.ClusterProxy, namespace, clusterName string) {
				// This check ensures that owner references are resilient - i.e. correctly re-reconciled - when removed.
				framework.ValidateOwnerReferencesResilience(ctx, proxy, namespace, clusterName, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
					framework.CoreOwnerReferenceAssertion,
					helpers.ExpOwnerReferenceAssertions,
					helpers.IonosCloudInfraOwnerReferenceAssertions,
					framework.KubeadmBootstrapOwnerReferenceAssertions,
					framework.KubeadmControlPlaneOwnerReferenceAssertions,
					helpers.KubernetesReferenceAssertions,
				)
				// This check ensures that owner references are correctly updated to the correct apiVersion.
				framework.ValidateOwnerReferencesOnUpdate(ctx, proxy, namespace, clusterName, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
					framework.CoreOwnerReferenceAssertion,
					helpers.ExpOwnerReferenceAssertions,
					helpers.IonosCloudInfraOwnerReferenceAssertions,
					framework.KubeadmBootstrapOwnerReferenceAssertions,
					framework.KubeadmControlPlaneOwnerReferenceAssertions,
					helpers.KubernetesReferenceAssertions,
				)

				clusters := &infrav1.IonosCloudClusterList{}
				Expect(proxy.GetClient().List(ctx, clusters, runtimeclient.InNamespace(namespace))).NotTo(HaveOccurred())

				// This check ensures that finalizers are resilient - i.e. correctly re-reconciled - when removed.
				framework.ValidateFinalizersResilience(ctx, proxy, namespace, clusterName, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
					framework.CoreFinalizersAssertionWithLegacyClusters,
					framework.KubeadmControlPlaneFinalizersAssertion,
					helpers.IonosCloudInfraFinalizersAssertion,
					helpers.ExpFinalizersAssertion,
					helpers.KubernetesFinalizersAssertion(clusters),
				)

				// This check ensures that the resourceVersions are stable, i.e. it verifies there are no
				// continuous reconciles when everything should be stable.
				framework.ValidateResourceVersionStable(ctx, proxy, namespace, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName))
			},
		}
	})
})

var _ = Describe("Should be able to create a cluster with 1 control-plane and 1 worker node and scale it", func() {
	capie2e.MachineDeploymentScaleSpec(ctx, func() capie2e.MachineDeploymentScaleSpecInput {
		return capie2e.MachineDeploymentScaleSpecInput{
			E2EConfig:             e2eConfig,
			ClusterctlConfigPath:  clusterctlConfigPath,
			BootstrapClusterProxy: bootstrapClusterProxy,
			Flavor:                "image-selector",
			ArtifactFolder:        artifactFolder,
			SkipCleanup:           skipCleanup,
			PostNamespaceCreated:  cloudEnv.createCredentialsSecretPNC,
		}
	})
})

var _ = Describe("Should be able to create a cluster with 1 control-plane using an IP from the IPAddressPool", func() {
	capie2e.QuickStartSpec(ctx, func() capie2e.QuickStartSpecInput {
		return capie2e.QuickStartSpecInput{
			E2EConfig:                e2eConfig,
			ControlPlaneMachineCount: ptr.To(int64(1)),
			WorkerMachineCount:       ptr.To(int64(0)),
			ClusterctlConfigPath:     clusterctlConfigPath,
			BootstrapClusterProxy:    bootstrapClusterProxy,
			Flavor:                   ptr.To("ipam"),
			ArtifactFolder:           artifactFolder,
			SkipCleanup:              skipCleanup,
			PostNamespaceCreated:     cloudEnv.createCredentialsSecretPNC,
			PostMachinesProvisioned: func(managementClusterProxy framework.ClusterProxy, namespace, _ string) {
				machines := &infrav1.IonosCloudMachineList{}
				Expect(managementClusterProxy.GetClient().List(ctx, machines, runtimeclient.InNamespace(namespace))).NotTo(HaveOccurred())
				nic := machines.Items[0].Status.MachineNetworkInfo.NICInfo[0]
				desired := os.Getenv("ADDITIONAL_IPS")
				Expect(nic.IPv4Addresses).To(ContainElement(desired))
			},
		}
	})
})
