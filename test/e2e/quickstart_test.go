//go:build e2e
// +build e2e

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
	. "github.com/onsi/ginkgo/v2"
	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	capie2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/test/e2e/helpers"
)

var _ = FDescribe("When following the Cluster API quick-start", func() {
	capie2e.QuickStartSpec(ctx, func() capie2e.QuickStartSpecInput {
		return capie2e.QuickStartSpecInput{
			E2EConfig:               e2eConfig,
			ClusterctlConfigPath:    clusterctlConfigPath,
			BootstrapClusterProxy:   bootstrapClusterProxy,
			ArtifactFolder:          artifactFolder,
			SkipCleanup:             skipCleanup,
			PostMachinesProvisioned: postMachinesProvisionedQuickstart,
		}
	})
})

var _ = Describe("When following the Cluster API quick-start (high available)", func() {
	capie2e.QuickStartSpec(ctx, func() capie2e.QuickStartSpecInput {
		return capie2e.QuickStartSpecInput{
			E2EConfig:                e2eConfig,
			ClusterctlConfigPath:     clusterctlConfigPath,
			BootstrapClusterProxy:    bootstrapClusterProxy,
			ArtifactFolder:           artifactFolder,
			SkipCleanup:              skipCleanup,
			ControlPlaneMachineCount: ptr.To[int64](3),
			WorkerMachineCount:       ptr.To[int64](2),
			PostMachinesProvisioned:  postMachinesProvisionedQuickstart,
		}
	})
})

func postMachinesProvisionedQuickstart(proxy framework.ClusterProxy, namespace, clusterName string) {
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
	// This check ensures that finalizers are resilient - i.e. correctly re-reconciled - when removed.
	framework.ValidateFinalizersResilience(ctx, proxy, namespace, clusterName, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
		framework.CoreFinalizersAssertion,
		framework.KubeadmControlPlaneFinalizersAssertion,
		helpers.IonosCloudInfraFinalizersAssertion,
		helpers.ExpFinalizersAssertion,
		helpers.KubernetesFinalizersAssertion,
	)

	// This check ensures that the resourceVersions are stable, i.e. it verifies there are no
	// continuous reconciles when everything should be stable.
	framework.ValidateResourceVersionStable(ctx, proxy, namespace, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName))
}
