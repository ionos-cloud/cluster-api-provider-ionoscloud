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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

// IonosCloudClusterSpec defines the desired state of IonosCloudCluster.
type IonosCloudClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	//+kubebuilder:validation:XValidation:rule="self == oldSelf || oldSelf.host == ''",message="control plane endpoint cannot be updated"
	//+optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	// Contract number is the contract number of the IONOS Cloud account.
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="contractNumber is immutable"
	ContractNumber string `json:"contractNumber"`

	// Location is the location where the data centers should be located.
	//+kubebuilder:validation:XValidation:rule="self == oldSelf",message="location is immutable"
	//+kubebuilder:example=de/txl
	//+kubebuilder:validation:MinLength=1
	Location string `json:"location"`
}

// IonosCloudClusterStatus defines the observed state of IonosCloudCluster.
type IonosCloudClusterStatus struct {
	// Ready indicates that the cluster is ready.
	//+optional
	//+kubebuilder:default=false
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IonosCloudCluster.
	//+optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// CurrentRequestByDatacenter maps data center IDs to a pending provisioning request made during reconciliation.
	//+optional
	CurrentRequestByDatacenter map[string]ProvisioningRequest `json:"currentRequest,omitempty"`

	// CurrentClusterRequest is the current pending request made during reconciliation for the whole cluster.
	//+optional
	CurrentClusterRequest *ProvisioningRequest `json:"currentClusterRequest,omitempty"`

	// ControlPlaneEndpointProviderID is the IONOS Cloud provider ID for the control plane endpoint IP block.
	// It will be in the format "ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a"
	//+optional
	ControlPlaneEndpointProviderID string `json:"controlPlaneEndpointProviderID,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels['cluster\\.x-k8s\\.io/cluster-name']",description="Cluster"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready"
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

// GetConditions returns the conditions from the status.
func (i *IonosCloudCluster) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

// SetConditions sets the conditions in the status.
func (i *IonosCloudCluster) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

// SetCurrentRequestByDatacenter sets the current provisioning request for the given data center.
// This function makes sure that the map is initialized before setting the request.
func (i *IonosCloudCluster) SetCurrentRequestByDatacenter(datacenterID string, request ProvisioningRequest) {
	if i.Status.CurrentRequestByDatacenter == nil {
		i.Status.CurrentRequestByDatacenter = map[string]ProvisioningRequest{}
	}
	i.Status.CurrentRequestByDatacenter[datacenterID] = request
}

// DeleteCurrentRequestByDatacenter deletes the current provisioning request for the given data center.
func (i *IonosCloudCluster) DeleteCurrentRequestByDatacenter(datacenterID string) {
	delete(i.Status.CurrentRequestByDatacenter, datacenterID)
}

// SetCurrentClusterRequest sets the current provisioning request for the cluster.
func (i *IonosCloudCluster) SetCurrentClusterRequest(method, status, requestPath string) {
	i.Status.CurrentClusterRequest = &ProvisioningRequest{
		Method:      method,
		RequestPath: requestPath,
		State:       status,
		Message:     nil,
	}
}

// DeleteCurrentClusterRequest deletes the current provisioning request for the cluster.
func (i *IonosCloudCluster) DeleteCurrentClusterRequest() {
	i.Status.CurrentClusterRequest = nil
}
