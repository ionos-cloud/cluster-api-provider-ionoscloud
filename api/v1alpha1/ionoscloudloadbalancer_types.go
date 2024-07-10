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

	// IonosCloudLoadBalancerReady is the condition for the IonosCloudLoadBalancer, which indicates that the load balancer is ready.
	IonosCloudLoadBalancerReady clusterv1.ConditionType = "LoadBalancerReady"
)

// IonosCloudLoadBalancerSpec defines the desired state of IonosCloudLoadBalancer.
type IonosCloudLoadBalancerSpec struct {
	// LoadBalancerEndpoint represents the endpoint of the load balanced control plane.
	//+kubebuilder:validation:XValidation:rule="self.host == oldSelf.host || oldSelf.host == ''",message="control plane endpoint host cannot be updated"
	//+kubebuilder:validation:XValidation:rule="self.port == oldSelf.port || oldSelf.port == 0",message="control plane endpoint port cannot be updated"
	LoadBalancerEndpoint clusterv1.APIEndpoint `json:"loadBalancerEndpoint,omitempty"`
}

// IonosCloudLoadBalancerStatus defines the observed state of IonosCloudLoadBalancer.
type IonosCloudLoadBalancerStatus struct {
	// Ready indicates that the load balancer is ready.
	//+optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the IonosCloudLoadBalancer.
	//+optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
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

func init() {
	objectTypes = append(objectTypes, &IonosCloudLoadBalancer{})
}
