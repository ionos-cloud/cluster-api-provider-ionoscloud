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
)

// IonosCloudClusterTemplateSpec defines the desired state of IonosCloudClusterTemplate.
type IonosCloudClusterTemplateSpec struct {
	Template IonosCloudClusterTemplateResource `json:"template"`
}

// IonosCloudClusterTemplateResource describes the data for creating a IonosCloudCluster from a template.
type IonosCloudClusterTemplateResource struct {
	Spec IonosCloudClusterSpec `json:"spec"`
}

//+kubebuilder:object:root=true

// IonosCloudClusterTemplate is the Schema for the ionoscloudclustertemplates API.
type IonosCloudClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IonosCloudClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// IonosCloudClusterTemplateList contains a list of IonosCloudClusterTemplate.
type IonosCloudClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IonosCloudCluster `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &IonosCloudClusterTemplate{}, &IonosCloudClusterTemplateList{})
}
