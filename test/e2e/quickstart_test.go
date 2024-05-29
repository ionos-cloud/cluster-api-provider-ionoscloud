//go:build e2e
// +build e2e

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/test/e2e/helpers"
)

var _ = Describe("When following the Cluster API quick-start", func() {
	e2e.QuickStartSpec(ctx, func() e2e.QuickStartSpecInput {
		return e2e.QuickStartSpecInput{
			E2EConfig:              e2eConfig,
			ClusterctlConfigPath:   clusterctlConfigPath,
			BootstrapClusterProxy:  bootstrapClusterProxy,
			ArtifactFolder:         artifactFolder,
			SkipCleanup:            skipCleanup,
			InfrastructureProvider: ptr.To(infrastructureProvider),
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
				// This check ensures that finalizers are resilient - i.e. correctly re-reconciled - when removed.
				framework.ValidateFinalizersResilience(ctx, proxy, namespace, clusterName, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
					framework.CoreFinalizersAssertion,
					framework.KubeadmControlPlaneFinalizersAssertion,
					helpers.IonosCloudInfraFinalizersAssertion,
					helpers.ExpFinalizersAssertion,
				)
				// NOTE(gfariasalves): Should we add some custom validation for the credentials secret?

				// This check ensures that the resourceVersions are stable, i.e. it verifies there are no
				// continuous reconciles when everything should be stable.
				framework.ValidateResourceVersionStable(ctx, proxy, namespace, clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName))
			},
		}
	})
})
