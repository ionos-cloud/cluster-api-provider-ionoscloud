//go:build e2e
// +build e2e

package helpers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// Kinds and Owners for types in the core API package.
var (
	coreGroupVersion = clusterv1.GroupVersion.String()

	extensionConfigKind    = "ExtensionConfig"
	clusterClassKind       = "ClusterClass"
	clusterKind            = "Cluster"
	machineKind            = "Machine"
	machineSetKind         = "MachineSet"
	machineDeploymentKind  = "MachineDeployment"
	machineHealthCheckKind = "MachineHealthCheck"

	clusterOwner                = metav1.OwnerReference{Kind: clusterKind, APIVersion: coreGroupVersion}
	clusterController           = metav1.OwnerReference{Kind: clusterKind, APIVersion: coreGroupVersion, Controller: ptr.To(true)}
	clusterClassOwner           = metav1.OwnerReference{Kind: clusterClassKind, APIVersion: coreGroupVersion}
	machineDeploymentController = metav1.OwnerReference{Kind: machineDeploymentKind, APIVersion: coreGroupVersion, Controller: ptr.To(true)}
	machineSetController        = metav1.OwnerReference{Kind: machineSetKind, APIVersion: coreGroupVersion, Controller: ptr.To(true)}
	machineController           = metav1.OwnerReference{Kind: machineKind, APIVersion: coreGroupVersion, Controller: ptr.To(true)}
)

// Kinds for types in the IONOS Cloud infrastructure package.
var (
	ionosCloudMachineKind         = "IonosCloudMachine"
	ionosCloudMachineTemplateKind = "IonosCloudMachineTemplate"
	ionosCloudClusterKind         = "IonosCloudCluster"
	ionosCloudClusterTemplateKind = "IonosCloudClusterTemplate"
)

// IonosCloudInfraOwnerReferenceAssertions maps IONOS Cloud Infrastructure types to functions which return an error if the passed
// OwnerReferences aren't as expected.
// Note: These relationships are documented in https://github.com/kubernetes-sigs/cluster-api/tree/main/docs/book/src/reference/owner_references.md.
// That document should be updated if these references change.
var IonosCloudInfraOwnerReferenceAssertions = map[string]func([]metav1.OwnerReference) error{
	ionosCloudMachineKind: func(owners []metav1.OwnerReference) error {
		// The IonosCloudMachine must be owned and controlled by a Machine.
		// TODO(gfariasalves-ionos): Add IonosCloudMachinePool here when it is implemented,
		//  and change from HasExactOwners to HasOneOfExactOwners.
		return framework.HasExactOwners(owners, machineController)

	},
	ionosCloudMachineTemplateKind: func(owners []metav1.OwnerReference) error {
		// IonosCloudMachineTemplates created for specific Clusters in the Topology controller must be owned by a Cluster.
		// TODO(gfariasalves-ionos): Add IonosCloudClusterClass here when it is implemented,
		//  and change from HasExactOwners to HasOneOfExactOwners.
		return framework.HasExactOwners(owners, clusterOwner)
	},
	ionosCloudClusterKind: func(owners []metav1.OwnerReference) error {
		// IonosCloudCluster must be owned and controlled by a Cluster.
		return framework.HasExactOwners(owners, clusterController)
	},
}
