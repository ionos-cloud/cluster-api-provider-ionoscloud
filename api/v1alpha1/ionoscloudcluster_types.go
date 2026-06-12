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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// ClusterFinalizer allows cleanup of resources, which are
	// associated with the IonosCloudCluster before removing it from the API server.
	ClusterFinalizer = "ionoscloudcluster.infrastructure.cluster.x-k8s.io"

	// IonosCloudClusterReady is the condition for the IonosCloudCluster, which indicates that the cluster is ready.
	IonosCloudClusterReady clusterv1.ConditionType = "ClusterReady"

	// IonosCloudClusterKind is the string resource kind of the IonosCloudCluster resource.
	IonosCloudClusterKind = "IonosCloudCluster"
)

//+kubebuilder:validation:XValidation:rule="!has(self.loadBalancerProviderRef) || has(self.location)",message="location is required when loadBalancerProviderRef is set"

// IonosCloudClusterSpec defines the desired state of IonosCloudCluster.
type IonosCloudClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +kubebuilder:validation:XValidation:rule="size(self.host) == 0 && self.port == 0 || self.port > 0 && self.port < 65536",message="port must be within 1-65535"
	//
	// TODO(gfariasalves): as of now, IP must be provided by the user as we still don't insert the
	// provider-provided block IP into the kube-vip manifest.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	// Location is the location where the data centers should be located.
	//
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="location is immutable"
	//+kubebuilder:example=de/txl
	//+kubebuilder:validation:MinLength=1
	//+optional
	Location string `json:"location,omitempty"`

	// CredentialsRef is a reference to the secret containing the credentials to access the IONOS Cloud API.
	//+kubebuilder:validation:XValidation:rule="has(self.name) && self.name != ''",message="credentialsRef.name must be provided"
	CredentialsRef corev1.LocalObjectReference `json:"credentialsRef"`

	// LoadBalancerProviderRef is a reference to the load balancer provider configuration.
	// An empty loadBalancerProviderRef field is allowed and means to disable any load balancer logic.
	LoadBalancerProviderRef *corev1.LocalObjectReference `json:"loadBalancerProviderRef,omitempty"`
}

// IonosCloudClusterStatus defines the observed state of IonosCloudCluster.
type IonosCloudClusterStatus struct {
	// Initialization provides observations of the IonosCloudCluster initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial
	// cluster provisioning. The value of these fields is never updated after initial provisioning is completed.
	// Use conditions to monitor the operational state of the cluster's infrastructure.
	//+optional
	Initialization IonosCloudClusterInitializationStatus `json:"initialization,omitempty,omitzero"`

	// Conditions represents the observations of the current state of the IonosCloudCluster.
	//+optional
	//+listType=map
	//+listMapKey=type
	//+kubebuilder:validation:MaxItems=32
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// CurrentRequestByDatacenter maps data center IDs to a pending provisioning request made during reconciliation.
	//+optional
	CurrentRequestByDatacenter map[string]ProvisioningRequest `json:"currentRequest,omitempty"`

	// CurrentClusterRequest is the current pending request made during reconciliation for the whole cluster.
	//+optional
	CurrentClusterRequest *ProvisioningRequest `json:"currentClusterRequest,omitempty"`

	// ControlPlaneEndpointIPBlockID is the IONOS Cloud UUID for the control plane endpoint IP block.
	//+optional
	ControlPlaneEndpointIPBlockID string `json:"controlPlaneEndpointIPBlockID,omitempty"`

	// Deprecated groups all status fields deprecated and scheduled for removal when v1beta1 contract support is dropped.
	//+optional
	Deprecated *IonosCloudClusterDeprecatedStatus `json:"deprecated,omitempty"`
}

// IonosCloudClusterInitializationStatus provides observations of the IonosCloudCluster initialization process.
// +kubebuilder:validation:MinProperties=1
type IonosCloudClusterInitializationStatus struct {
	// Provisioned is true when the infrastructure cluster is fully provisioned.
	// NOTE: this field is part of the Cluster API contract and is used to orchestrate provisioning.
	// The value of this field is never updated after initial provisioning is completed.
	//+optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// IonosCloudClusterDeprecatedStatus groups all status fields deprecated and scheduled for removal when
// v1beta1 contract support is dropped.
type IonosCloudClusterDeprecatedStatus struct {
	// V1Beta1 groups all v1beta1 status fields that are deprecated and scheduled for removal.
	//+optional
	V1Beta1 *IonosCloudClusterV1Beta1DeprecatedStatus `json:"v1beta1,omitempty"`
}

// IonosCloudClusterV1Beta1DeprecatedStatus contains deprecated v1beta1 fields.
type IonosCloudClusterV1Beta1DeprecatedStatus struct {
	// Ready indicates that the cluster is ready.
	//
	// Deprecated: Use Initialization.Provisioned instead.
	//+optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IonosCloudCluster using the deprecated v1beta1 condition type.
	//
	// Deprecated: Use the top-level conditions field instead.
	//+optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=ionoscloudclusters,scope=Namespaced,categories=cluster-api;ionoscloud,shortName=icc
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels['cluster\\.x-k8s\\.io/cluster-name']",description="Cluster"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.initialization.provisioned",description="Cluster infrastructure is ready"
//+kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.controlPlaneEndpoint",description="API Endpoint"

// IonosCloudCluster is the Schema for the ionoscloudclusters API.
type IonosCloudCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IonosCloudClusterSpec   `json:"spec,omitempty"`
	Status IonosCloudClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IonosCloudClusterList contains a list of IonosCloudCluster.
type IonosCloudClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IonosCloudCluster `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &IonosCloudCluster{}, &IonosCloudClusterList{})
}

// GetV1Beta1Conditions returns the deprecated v1beta1 conditions from status.deprecated.v1beta1.conditions.
func (i *IonosCloudCluster) GetV1Beta1Conditions() clusterv1.Conditions {
	if i.Status.Deprecated == nil || i.Status.Deprecated.V1Beta1 == nil {
		return nil
	}
	return i.Status.Deprecated.V1Beta1.Conditions
}

// SetV1Beta1Conditions sets the deprecated v1beta1 conditions in status.deprecated.v1beta1.conditions.
func (i *IonosCloudCluster) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	if i.Status.Deprecated == nil {
		i.Status.Deprecated = &IonosCloudClusterDeprecatedStatus{}
	}
	if i.Status.Deprecated.V1Beta1 == nil {
		i.Status.Deprecated.V1Beta1 = &IonosCloudClusterV1Beta1DeprecatedStatus{}
	}
	i.Status.Deprecated.V1Beta1.Conditions = conditions
}

// GetConditions returns the v1beta2 conditions from status.conditions.
func (i *IonosCloudCluster) GetConditions() []metav1.Condition {
	return i.Status.Conditions
}

// SetConditions sets the v1beta2 conditions in status.conditions.
func (i *IonosCloudCluster) SetConditions(conditions []metav1.Condition) {
	i.Status.Conditions = conditions
}

// SetCurrentClusterRequest sets the current provisioning request for the cluster.
func (i *IonosCloudCluster) SetCurrentClusterRequest(method, status, requestPath string) {
	i.Status.CurrentClusterRequest = &ProvisioningRequest{
		Method:      method,
		RequestPath: requestPath,
		State:       status,
	}
}

// DeleteCurrentClusterRequest deletes the current provisioning request for the cluster.
func (i *IonosCloudCluster) DeleteCurrentClusterRequest() {
	i.Status.CurrentClusterRequest = nil
}
