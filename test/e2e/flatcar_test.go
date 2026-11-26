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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	capie2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	conditions "sigs.k8s.io/cluster-api/util/conditions"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"

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

				servers := make([]string, 0, len(machines.Items))
				bootVolumes := make([]string, 0, len(machines.Items))
				configVolumes := make([]string, 0, len(machines.Items))
				for _, m := range machines.Items {
					name := m.Spec.InfrastructureRef.Name
					servers = append(servers, name)
					bootVolumes = append(bootVolumes, "vol-"+name)
					configVolumes = append(configVolumes, configVolumeName(namespace, name))
				}

				By("Checking that the config volumes are deleted once the nodes joined")
				Eventually(func(g Gomega) {
					names, err := cloudEnv.volumeNames(ctx)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(names).To(ContainElements(bootVolumes))
					g.Expect(names).NotTo(ContainElement(BeElementOf(configVolumes)))

					for _, name := range servers {
						infraMachine := &infrav1.IonosCloudMachine{}
						g.Expect(proxy.GetClient().Get(ctx, runtimeclient.ObjectKey{
							Namespace: namespace, Name: name,
						}, infraMachine)).To(Succeed())
						g.Expect(conditions.IsTrue(infraMachine, infrav1.BootstrapVolumeRemovedCondition)).
							To(BeTrue(), "IonosCloudMachine %s has not reported its config volume as removed", name)
					}
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
					By("Checking that no server or volume of the deleted cluster remains in the data center")
					Eventually(func(g Gomega) {
						names, err := cloudEnv.serverNames(ctx)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(names).NotTo(ContainElement(BeElementOf(servers)))
						names, err = cloudEnv.volumeNames(ctx)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(names).NotTo(ContainElement(BeElementOf(append(bootVolumes, configVolumes...))))
					}, e2eConfig.GetIntervals("default", "wait-volume-deletion")...).Should(Succeed())
				})
			},
		}
	})
})

// configVolumeName returns the name of the config volume of an IonosCloudMachine.
func configVolumeName(namespace, name string) string {
	namespaceHash := sha256.Sum256([]byte(namespace))
	return fmt.Sprintf("cfg-%s-%s", name, hex.EncodeToString(namespaceHash[:])[:8])
}

// flatcarE2EConfig returns the e2e config with KUBERNETES_VERSION set to the version of the Flatcar node image.
// An empty KUBERNETES_VERSION_FLATCAR env var (an unset CI variable) keeps the config default.
func flatcarE2EConfig() *clusterctl.E2EConfig {
	cfg := e2eConfig.DeepCopy()
	version := os.Getenv(KubernetesVersionFlatcar)
	if version == "" {
		version = cfg.Variables[KubernetesVersionFlatcar]
	}
	Expect(version).NotTo(BeEmpty(), "%s must be set", KubernetesVersionFlatcar)

	// The framework reads variables from the environment first, so the config value would be ignored.
	if v, ok := os.LookupEnv(KubernetesVersion); ok && v != version {
		Fail(fmt.Sprintf("%s=%q overrides the Flatcar version %q; unset it or set %s to the same version",
			KubernetesVersion, v, version, KubernetesVersionFlatcar))
	}
	cfg.Variables[KubernetesVersion] = version
	return cfg
}
