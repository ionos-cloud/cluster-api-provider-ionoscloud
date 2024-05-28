//go:build e2e
// +build e2e

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
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
				framework.ValidateOwnerReferencesResilience(ctx, proxy, namespace, clusterName,
					framework.CoreOwnerReferenceAssertion,
					helpers.IonosCloudInfraOwnerReferenceAssertions,
					framework.KubeadmBootstrapOwnerReferenceAssertions,
					framework.KubeadmControlPlaneOwnerReferenceAssertions,
					framework.KubernetesReferenceAssertions,
				)
				// This check ensures that owner references are correctly updated to the correct apiVersion.
				framework.ValidateOwnerReferencesOnUpdate(ctx, proxy, namespace, clusterName,
					framework.CoreOwnerReferenceAssertion,
					helpers.IonosCloudInfraOwnerReferenceAssertions,
					framework.KubeadmBootstrapOwnerReferenceAssertions,
					framework.KubeadmControlPlaneOwnerReferenceAssertions,
					framework.KubernetesReferenceAssertions,
				)
				// This check ensures that finalizers are resilient - i.e. correctly re-reconciled - when removed.
				framework.ValidateFinalizersResilience(ctx, proxy, namespace, clusterName,
					framework.CoreFinalizersAssertion,
					framework.KubeadmControlPlaneFinalizersAssertion,
					helpers.IonosCloudInfraFinalizersAssertion,
				)
			},
		}
	})
})
