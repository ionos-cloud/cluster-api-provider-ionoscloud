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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IonosCloudMachineTemplateSpec defines the desired state of IonosCloudMachineTemplate
type IonosCloudMachineTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of IonosCloudMachineTemplate. Edit ionoscloudmachinetemplate_types.go to remove/update
	Template IonosCloudMachineTemplateResource `json:"template"`
}

// IonosCloudMachineTemplateStatus defines the observed state of IonosCloudMachineTemplate
type IonosCloudMachineTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IonosCloudMachineTemplate is the Schema for the ionoscloudmachinetemplates API
type IonosCloudMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IonosCloudMachineTemplateSpec   `json:"spec,omitempty"`
	Status IonosCloudMachineTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IonosCloudMachineTemplateList contains a list of IonosCloudMachineTemplate
type IonosCloudMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IonosCloudMachineTemplate `json:"items"`
}

// IonosCloudMachineTemplateResource defines the spec and metadata for IonosCloudMachineTemplate supported by capi.
type IonosCloudMachineTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the IonosCloudMachineSpec for the IonosCloudMachineTemplate.
	Spec IonosCloudMachineSpec `json:"spec"`
}

func init() {
	objectTypes = append(objectTypes, &IonosCloudMachineTemplate{}, &IonosCloudMachineTemplateList{})
}
