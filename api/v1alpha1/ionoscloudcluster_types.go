/*
Copyright 2023 IONOS Cloud.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// ClusterFinalizer allows cleanup of resources, which are
	// associated with the IonosCloudCluster before removing it from the apiserver
	ClusterFinalizer = "ionoscloudcluster.infrastructure.cluster.x-k8s.io"
)

// IonosCloudClusterSpec defines the desired state of IonosCloudCluster
type IonosCloudClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
}

// IonosCloudClusterStatus defines the observed state of IonosCloudCluster
type IonosCloudClusterStatus struct {
	// Ready indicates that the cluster is ready.
	// +optional
	// +kubebuilder:default=false
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IonosCloudCluster is the Schema for the ionoscloudclusters API
type IonosCloudCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IonosCloudClusterSpec   `json:"spec,omitempty"`
	Status IonosCloudClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IonosCloudClusterList contains a list of IonosCloudCluster
type IonosCloudClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IonosCloudCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IonosCloudCluster{}, &IonosCloudClusterList{})
}
