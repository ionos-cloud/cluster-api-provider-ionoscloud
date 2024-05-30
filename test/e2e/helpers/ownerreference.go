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

package helpers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// Kinds and Owners for types in the core API package.
var (
	coreGroupVersion = clusterv1.GroupVersion.String()

	clusterKind = "Cluster"
	machineKind = "Machine"

	clusterOwner      = metav1.OwnerReference{Kind: clusterKind, APIVersion: coreGroupVersion}
	clusterController = metav1.OwnerReference{Kind: clusterKind, APIVersion: coreGroupVersion, Controller: ptr.To(true)}
	machineController = metav1.OwnerReference{Kind: machineKind, APIVersion: coreGroupVersion, Controller: ptr.To(true)}
)

var (
	ionosCloudMachineTemplateKind = "IonosCloudMachineTemplate"

	ionosCloudClusterController = metav1.OwnerReference{Kind: "IonosCloudCluster", APIVersion: infrav1.GroupVersion.String(), Controller: ptr.To(true)}
)

var (
	clusterResourceSetBindingKind = "ClusterResourceSetBinding"
	clusterResourceSetOwner       = metav1.OwnerReference{Kind: "ClusterResourceSet", APIVersion: addonsv1.GroupVersion.String()}
)

// Kind and Owners for types in the Kubeadm ControlPlane package.
var (
	kubeadmControlPlaneKind = "KubeadmControlPlane"

	kubeadmControlPlaneGroupVersion = controlplanev1.GroupVersion.String()
	kubeadmControlPlaneController   = metav1.OwnerReference{Kind: kubeadmControlPlaneKind, APIVersion: kubeadmControlPlaneGroupVersion, Controller: ptr.To(true)}
)

// Owners and kinds for types in the Kubeadm Bootstrap package.
var (
	kubeadmConfigKind = "KubeadmConfig"

	kubeadmConfigGroupVersion = bootstrapv1.GroupVersion.String()
	kubeadmConfigController   = metav1.OwnerReference{Kind: kubeadmConfigKind, APIVersion: kubeadmConfigGroupVersion, Controller: ptr.To(true)}
)

var (
	configMapKind = "ConfigMap"
	secretKind    = "Secret"
)

// IonosCloudInfraOwnerReferenceAssertions maps IONOS Cloud Infrastructure types to functions which return an error if the passed
// OwnerReferences aren't as expected.
// Note: These relationships are documented in https://github.com/kubernetes-sigs/cluster-api/tree/main/docs/book/src/reference/owner_references.md.
// That document should be updated if these references change.
var IonosCloudInfraOwnerReferenceAssertions = map[string]func([]metav1.OwnerReference) error{
	"IonosCloudMachine": func(owners []metav1.OwnerReference) error {
		return framework.HasExactOwners(owners, machineController)

	},
	ionosCloudMachineTemplateKind: func(owners []metav1.OwnerReference) error {
		return framework.HasExactOwners(owners, clusterOwner)
	},
	"IonosCloudCluster": func(owners []metav1.OwnerReference) error {
		// IonosCloudCluster must be owned and controlled by a Cluster.
		return framework.HasExactOwners(owners, clusterController)
	},
}

// ExpOwnerReferenceAssertions maps experimental types to functions which return an error if the passed OwnerReferences
// aren't as expected.
// Note: These relationships are documented in https://github.com/kubernetes-sigs/cluster-api/tree/main/docs/book/src/reference/owner_references.md.
// That document should be updated if these references change.
var ExpOwnerReferenceAssertions = map[string]func([]metav1.OwnerReference) error{
	"ClusterResourceSet": func(owners []metav1.OwnerReference) error {
		// ClusterResourcesSet doesn't have ownerReferences (it is a clusterctl move-hierarchy root).
		return framework.HasExactOwners(owners)
	},
	// ClusterResourcesSetBinding has ClusterResourceSet set as owners on creation.
	clusterResourceSetBindingKind: func(owners []metav1.OwnerReference) error {
		return framework.HasOneOfExactOwners(owners, []metav1.OwnerReference{clusterResourceSetOwner}, []metav1.OwnerReference{clusterResourceSetOwner, clusterResourceSetOwner})
	},
}

// KubernetesReferenceAssertions maps Kubernetes types to functions which return an error if the passed OwnerReferences
// aren't as expected.
// Note: These relationships are documented in https://github.com/kubernetes-sigs/cluster-api/tree/main/docs/book/src/reference/owner_references.md.
// That document should be updated if these references change.
var KubernetesReferenceAssertions = map[string]func([]metav1.OwnerReference) error{
	secretKind: func(owners []metav1.OwnerReference) error {
		// Secrets for cluster certificates must be owned and controlled by the KubeadmControlPlane.
		// The bootstrap secret should be owned and controlled by a KubeadmControlPlane.
		// The cluster IONOS Cloud credentials secret should be owned and controlled by IonosCloudClusterController
		return framework.HasOneOfExactOwners(owners,
			[]metav1.OwnerReference{kubeadmControlPlaneController},
			[]metav1.OwnerReference{kubeadmConfigController},
			[]metav1.OwnerReference{ionosCloudClusterController},
		)
	},
	configMapKind: func(owners []metav1.OwnerReference) error {
		// The only configMaps considered here are those owned by a ClusterResourceSet.
		return framework.HasExactOwners(owners, clusterResourceSetOwner)
	},
}
