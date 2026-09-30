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
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/test/framework"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

// Kinds and Owners for types in the core API package.
var (
	coreGroupVersion = clusterv1.GroupVersion.String()

	clusterOwner      = metav1.OwnerReference{Kind: clusterv1.ClusterKind, APIVersion: coreGroupVersion}
	machineController = metav1.OwnerReference{Kind: "Machine", APIVersion: coreGroupVersion, Controller: new(true)}
)

var ionosCloudClusterController = metav1.OwnerReference{Kind: infrav1.IonosCloudClusterKind, APIVersion: infrav1.GroupVersion.String(), Controller: new(false)}

// IonosCloudInfraOwnerReferenceAssertions maps IONOS Cloud Infrastructure types to functions which return an error if the passed
// OwnerReferences aren't as expected.
// Note: These relationships are documented in https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/book/src/reference/api/owner-references.md.
// That document should be updated if these references change.
var IonosCloudInfraOwnerReferenceAssertions = map[string]func(types.NamespacedName, []metav1.OwnerReference) error{
	infrav1.IonosCloudMachineType: func(_ types.NamespacedName, owners []metav1.OwnerReference) error {
		return framework.HasExactOwners(owners, machineController)
	},
	"IonosCloudMachineTemplate": func(_ types.NamespacedName, owners []metav1.OwnerReference) error {
		return framework.HasExactOwners(owners, clusterOwner)
	},
	infrav1.IonosCloudClusterKind: func(_ types.NamespacedName, owners []metav1.OwnerReference) error {
		// Since CAPI v1.13 the Cluster only controls its InfraCluster when spec.topology is set
		// (kubernetes-sigs/cluster-api#13332); our templates don't use a ClusterClass.
		return framework.HasExactOwners(owners, clusterOwner)
	},
}

// KubernetesReferenceAssertions is framework.KubernetesReferenceAssertions, plus the cluster's
// IONOS Cloud credentials Secret, which is owned by the IonosCloudCluster.
// AssertOwnerReferences runs every assertion registered for a kind, so the upstream Secret
// rule has to be replaced rather than complemented by a second map.
var KubernetesReferenceAssertions = func() map[string]func(types.NamespacedName, []metav1.OwnerReference) error {
	assertions := maps.Clone(framework.KubernetesReferenceAssertions)
	upstreamSecret := assertions["Secret"]
	assertions["Secret"] = func(nn types.NamespacedName, owners []metav1.OwnerReference) error {
		if framework.HasExactOwners(owners, ionosCloudClusterController) == nil {
			return nil
		}
		return upstreamSecret(nn, owners)
	}
	return assertions
}()
