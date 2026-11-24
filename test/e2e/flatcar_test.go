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

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	capie2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flatcar: Should be able to create a cluster with 1 control-plane and 1 worker node using Ignition", func() {
	if os.Getenv(FlatcarImageID) == "" || os.Getenv(BootstrapVolumeImageID) == "" {
		It("Should create a workload cluster", func() {
			Skip(FlatcarImageID + " and " + BootstrapVolumeImageID + " must be set")
		})
		return
	}

	capie2e.QuickStartSpec(ctx, func() capie2e.QuickStartSpecInput {
		return capie2e.QuickStartSpecInput{
			E2EConfig:                flatcarE2EConfig(),
			ClusterctlConfigPath:     clusterctlConfigPath,
			BootstrapClusterProxy:    bootstrapClusterProxy,
			ArtifactFolder:           artifactFolder,
			SkipCleanup:              skipCleanup,
			Flavor:                   new("flatcar"),
			ControlPlaneMachineCount: new(int64(1)),
			WorkerMachineCount:       new(int64(1)),
			PostNamespaceCreated:     cloudEnv.createCredentialsSecretPNC,
			PostMachinesProvisioned: func(proxy framework.ClusterProxy, namespace, clusterName string) {
				machines := &clusterv1.MachineList{}
				Eventually(func(g Gomega) {
					g.Expect(proxy.GetClient().List(ctx, machines, runtimeclient.InNamespace(namespace),
						runtimeclient.MatchingLabels{clusterv1.ClusterNameLabel: clusterName})).To(Succeed())
					g.Expect(machines.Items).NotTo(BeEmpty())
					for _, m := range machines.Items {
						g.Expect(m.Status.NodeRef.IsDefined()).To(BeTrue(), "Machine %s has no NodeRef", m.Name)
					}
				}, e2eConfig.GetIntervals("default", "wait-nodes-ready")...).Should(Succeed())

				bootVolumes := make([]string, 0, len(machines.Items))
				configVolumes := make([]string, 0, len(machines.Items))
				for _, m := range machines.Items {
					bootVolumes = append(bootVolumes, "vol-"+m.Spec.InfrastructureRef.Name)
					configVolumes = append(configVolumes, "cfg-"+m.Spec.InfrastructureRef.Name)
				}

				By("Checking that the config volumes are deleted once the nodes joined")
				Eventually(func(g Gomega) {
					names, err := cloudEnv.volumeNames(ctx)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(names).To(ContainElements(bootVolumes))
					g.Expect(names).NotTo(ContainElement(BeElementOf(configVolumes)))
				}, e2eConfig.GetIntervals("default", "wait-volume-deletion")...).Should(Succeed())

				framework.ValidateResourceVersionStable(ctx, framework.ValidateResourceVersionStableInput{
					ClusterProxy:             proxy,
					Namespace:                namespace,
					OwnerGraphFilterFunction: clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
					WaitToBecomeStable:       e2eConfig.GetIntervals("default", "wait-resource-versions-become-stable"),
				})

				if skipCleanup {
					return
				}
				// Runs after QuickStartSpec's AfterEach has deleted the cluster.
				DeferCleanup(func() {
					By("Checking that no volume of the deleted cluster remains in the data center")
					Eventually(func(g Gomega) {
						names, err := cloudEnv.volumeNames(ctx)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(names).NotTo(ContainElement(BeElementOf(append(bootVolumes, configVolumes...))))
					}, e2eConfig.GetIntervals("default", "wait-volume-deletion")...).Should(Succeed())
				})
			},
		}
	})
})

func flatcarE2EConfig() *clusterctl.E2EConfig {
	cfg := e2eConfig.DeepCopy()
	if v := cfg.GetVariableOrEmpty(KubernetesVersionFlatcar); v != "" {
		cfg.Variables[KubernetesVersion] = v
	}
	return cfg
}
