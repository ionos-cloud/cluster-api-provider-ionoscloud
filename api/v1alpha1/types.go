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

import corev1 "k8s.io/api/core/v1"

// ProvisioningRequest is a definition of a provisioning request
// in the IONOS Cloud.
type ProvisioningRequest struct {
	// Method is the request method
	Method string `json:"method"`

	// RequestPath is the sub path for the request URL
	RequestPath string `json:"requestPath"`

	// RequestStatus is the status of the request in the queue.
	//+kubebuilder:validation:Enum=QUEUED;RUNNING;DONE;FAILED
	//+optional
	State string `json:"state,omitempty"`
}

// IPAMConfig optionally defines which IP Pools to use.
// +kubebuilder:validation:XValidation:rule="!has(self.ipv4PoolRef) || (has(self.ipv4PoolRef.apiGroup) && self.ipv4PoolRef.apiGroup == 'ipam.cluster.x-k8s.io')",message="ipv4PoolRef.apiGroup must be 'ipam.cluster.x-k8s.io' when ipv4PoolRef is set"
// +kubebuilder:validation:XValidation:rule="!has(self.ipv4PoolRef) || (self.ipv4PoolRef.kind == 'InClusterIPPool' || self.ipv4PoolRef.kind == 'GlobalInClusterIPPool')",message="ipv4PoolRef.kind must be 'InClusterIPPool' or 'GlobalInClusterIPPool' when ipv4PoolRef is set"
// +kubebuilder:validation:XValidation:rule="!has(self.ipv4PoolRef) || (has(self.ipv4PoolRef.name) && size(self.ipv4PoolRef.name) > 0)",message="ipv4PoolRef.name must not be empty when ipv4PoolRef is set"
// +kubebuilder:validation:XValidation:rule="!has(self.ipv6PoolRef) || (has(self.ipv6PoolRef.apiGroup) && self.ipv6PoolRef.apiGroup == 'ipam.cluster.x-k8s.io')",message="ipv6PoolRef.apiGroup must be 'ipam.cluster.x-k8s.io' when ipv6PoolRef is set"
// +kubebuilder:validation:XValidation:rule="!has(self.ipv6PoolRef) || (self.ipv6PoolRef.kind == 'InClusterIPPool' || self.ipv6PoolRef.kind == 'GlobalInClusterIPPool')",message="ipv6PoolRef.kind must be 'InClusterIPPool' or 'GlobalInClusterIPPool' when ipv6PoolRef is set"
// +kubebuilder:validation:XValidation:rule="!has(self.ipv6PoolRef) || (has(self.ipv6PoolRef.name) && size(self.ipv6PoolRef.name) > 0)",message="ipv6PoolRef.name must not be empty when ipv6PoolRef is set"
type IPAMConfig struct {
	// IPv4PoolRef is a reference to an IPAMConfig Pool resource, which exposes IPv4 addresses.
	// The NIC will use an available IP address from the referenced pool.
	// +optional
	IPv4PoolRef *corev1.TypedLocalObjectReference `json:"ipv4PoolRef,omitempty"`

	// IPv6PoolRef is a reference to an IPAMConfig pool resource, which exposes IPv6 addresses.
	// The NIC will use an available IP address from the referenced pool.
	// +optional
	IPv6PoolRef *corev1.TypedLocalObjectReference `json:"ipv6PoolRef,omitempty"`
}
