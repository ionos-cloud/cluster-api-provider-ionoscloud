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
)

// IPAMConfig contains the config for ip address management.
type IPAMConfig struct {
	// IPv4PoolRef is a reference to an IPAMConfig Pool resource, which exposes IPv4 addresses.
	// The nic will use an available IP address from the referenced pool.
	// +kubebuilder:validation:XValidation:rule="self.apiGroup == 'ipam.cluster.x-k8s.io'",message="ipv4PoolRef allows only IPAMConfig apiGroup ipam.cluster.x-k8s.io"
	// +kubebuilder:validation:XValidation:rule="self.kind == 'InClusterIPPool' || self.kind == 'GlobalInClusterIPPool'",message="ipv4PoolRef allows either InClusterIPPool or GlobalInClusterIPPool"
	// +kubebuilder:validation:XValidation:rule="self.name != ''",message="ipv4PoolRef.name is required"
	// +optional
	IPv4PoolRef *corev1.TypedLocalObjectReference `json:"ipv4PoolRef,omitempty"`

	// IPv6PoolRef is a reference to an IPAMConfig pool resource, which exposes IPv6 addresses.
	// The nic will use an available IP address from the referenced pool.
	// +kubebuilder:validation:XValidation:rule="self.apiGroup == 'ipam.cluster.x-k8s.io'",message="ipv6PoolRef allows only IPAMConfig apiGroup ipam.cluster.x-k8s.io"
	// +kubebuilder:validation:XValidation:rule="self.kind == 'InClusterIPPool' || self.kind == 'GlobalInClusterIPPool'",message="ipv6PoolRef allows either InClusterIPPool or GlobalInClusterIPPool"
	// +kubebuilder:validation:XValidation:rule="self.name != ''",message="ipv6PoolRef.name is required"
	// +optional
	IPv6PoolRef *corev1.TypedLocalObjectReference `json:"ipv6PoolRef,omitempty"`
}
