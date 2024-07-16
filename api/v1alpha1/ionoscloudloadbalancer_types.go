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
	// LoadBalancerFinalizer allows cleanup of resources, which are
	// associated with the IonosCloudLoadBalancer before removing it from the API server.
	LoadBalancerFinalizer = "ionoscloudloadbalancer.infrastructure.cluster.x-k8s.io"

	// LoadBalancerReadyCondition is the condition for the IonosCloudLoadBalancer, which indicates that the load balancer is ready.
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"

	// InvalidEndpointConfigurationReason indicates that the endpoints for IonosCloudCluster and IonosCloudLoadBalancer
	// have not been properly configured.
	InvalidEndpointConfigurationReason = "InvalidEndpointConfiguration"
)

// IonosCloudLoadBalancerSpec defines the desired state of IonosCloudLoadBalancer.
// +kubebuilder:validation:XValidation:rule=`!has(self.external) || has(self.external) && has(self.loadBalancerEndpoint) && size(self.loadBalancerEndpoint.host) > 0 && self.loadBalancerEndpoint.port > 0`,message="external load balancers require a load balancer endpoint"
type IonosCloudLoadBalancerSpec struct {
	// LoadBalancerEndpoint represents the endpoint of the load balanced control plane.
	// If the endpoint isn't provided, the controller will reserve a new public IP address.
	// The port is optional and defaults to 6443.
	//
	// For external load balancers, the endpoint and port must be provided.
	//+kubebuilder:validation:XValidation:rule="self.host == oldSelf.host || oldSelf.host == ''",message="control plane endpoint host cannot be updated"
	//+kubebuilder:validation:XValidation:rule="self.port == oldSelf.port || oldSelf.port == 0",message="control plane endpoint port cannot be updated"
	LoadBalancerEndpoint clusterv1.APIEndpoint `json:"loadBalancerEndpoint,omitempty"`

	// LoadBalancerSource is the actual load balancer definition.
	LoadBalancerSource `json:",inline"`
}

// LoadBalancerSource defines the source of the load balancer.
type LoadBalancerSource struct {
	// NLB is used for setting up a network load balancer.
	//+optional
	NLB *NLBSpec `json:"nlb,omitempty"`

	// KubeVIP is used for setting up a highly available control plane.
	//+optional
	KubeVIP *KubeVIPSpec `json:"kubeVIP,omitempty"`

	// External is used for making use of an external load balancer.
	//+optional
	External *ExternalLoadBalancerSpec `json:"external,omitempty"`
}

// NLBSpec defines the spec for a network load balancer.
type NLBSpec struct {
	// DatacenterID is the ID of the datacenter where the load balancer should be created.
	// This field is required for NLB load balancers and needs to match the datacenter ID
	// of the control plane machines.
	//+kubebuilder:validation:Format=uuid
	//+required
	DatacenterID string `json:"datacenterID"`
}

// KubeVIPSpec defines the spec for a high availability load balancer.
type KubeVIPSpec struct {
	// Image is the container image to use for the KubeVIP static pod.
	Image string `json:"image,omitempty"`
}

// ExternalLoadBalancerSpec defines the spec for an external load balancer.
// External load balancers need to be manually set up by the user.
type ExternalLoadBalancerSpec struct{}

// IonosCloudLoadBalancerStatus defines the observed state of IonosCloudLoadBalancer.
type IonosCloudLoadBalancerStatus struct {
	// Ready indicates that the load balancer is ready.
	//+optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IonosCloudLoadBalancer.
	//+optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// CurrentRequest shows the current provisioning request for any
	// cloud resource that is being provisioned.
	//+optional
	CurrentRequest *ProvisioningRequest `json:"currentRequest,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IonosCloudLoadBalancer is the Schema for the ionoscloudloadbalancers API
// +kubebuilder:resource:path=ionoscloudloadbalancers,scope=Namespaced,categories=cluster-api;ionoscloud,shortName=iclb
type IonosCloudLoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IonosCloudLoadBalancerSpec   `json:"spec,omitempty"`
	Status IonosCloudLoadBalancerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IonosCloudLoadBalancerList contains a list of IonosCloudLoadBalancer.
type IonosCloudLoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IonosCloudLoadBalancer `json:"items"`
}

// GetConditions returns the conditions from the status.
func (l *IonosCloudLoadBalancer) GetConditions() clusterv1.Conditions {
	return l.Status.Conditions
}

// SetConditions sets the conditions in the status.
func (l *IonosCloudLoadBalancer) SetConditions(conditions clusterv1.Conditions) {
	l.Status.Conditions = conditions
}

func init() {
	objectTypes = append(objectTypes, &IonosCloudLoadBalancer{})
}
