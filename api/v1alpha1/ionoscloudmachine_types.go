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
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// IonosCloudMachineType is the named type for the API object.
	IonosCloudMachineType = "IonosCloudMachine"
)

// VolumeDiskType specifies the type of  hard disk.
type VolumeDiskType string

const (
	// VolumeDiskTypeHDD defines the disk type HDD.
	VolumeDiskTypeHDD VolumeDiskType = "HDD"
	// VolumeDiskTypeSSD defines the disk type SSD.
	VolumeDiskTypeSSD VolumeDiskType = "SSD"
)

// AvailabilityZone is the availability zone, where volumes are created.
type AvailabilityZone string

const (
	// AvailabilityZoneAuto selected an automatic availability zone.
	AvailabilityZoneAuto = "AUTO"
	// AvailabilityZoneOne zone 1.
	AvailabilityZoneOne = "ZONE_1"
	// AvailabilityZoneTwo zone 2.
	AvailabilityZoneTwo = "ZONE_2"
	// AvailabilityZoneThree zone 3.
	AvailabilityZoneThree = "ZONE_3"
)

// IonosCloudMachineSpec defines the desired state of IonosCloudMachine.
type IonosCloudMachineSpec struct {
	// ProviderID is the IONOS Cloud provider ID
	// will be in the format ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a
	// +optional
	ProviderID string `json:"providerId,omitempty"`

	// DatacenterID is the ID of the datacenter, where the machine should be created.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="DatacenterID is immutable"
	DatacenterID string `json:"datacenterID"`

	// ServerID is the unique identifier for a server in the IONOS Cloud context.
	// The value will be set, once the server was created.
	// +optional
	ServerID string `json:"serverID,omitempty"`

	// Cores defines the total number of cores for the enterprise server.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	NumCores int32 `json:"numCores,omitempty"`

	// MemoryMiB is the memory size for the enterprise server in MB.
	// Size must be specified in multiples of 256 MB with a minimum of 1024 MB
	// which is required as we are using hot-pluggable RAM by default.
	// +kubebuilder:validation:MultipleOf=256
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:default=1024
	// +optional
	MemoryMiB int32 `json:"memoryMiB,omitempty"`

	// CPUFamily defines the CPU architecture, which will be used for this enterprise server.
	// The not all CPU architectures are available in all datacenters.
	// +kubebuilder:example=AMD_OPTERON
	CPUFamily string `json:"cpuFamily"`

	// Disk defines the boot volume of the machine.
	// +optional
	Disk *Volume `json:"disk,omitempty"`

	// Network defines the network configuration for the enterprise server.
	Network *Network `json:"network,omitempty"`
}

// Network contains a network config.
type Network struct {
	// IPs is an optional set of IP addresses, which have been
	// reserved in the corresponding datacenter.
	// +listType=set
	// +optional
	IPs []string `json:"ips,omitempty"`

	// UseDHCP sets whether dhcp should be used or not.
	// NOTE(lubedacht) currently we do not support private clusters
	// therefore dhcp must be set to true.
	// +kubebuilder:default=true
	// +optional
	UseDHCP *bool `json:"useDhcp"`
}

// Volume is the physical storage on the machine.
type Volume struct {
	// Name is the name of the volume
	// +optional
	Name string `json:"name,omitempty"`

	// DiskType defines the type of the hard drive.
	// +kubebuilder:validation:Enum=HDD;SSD
	// +kubebuilder:default=HDD
	// +optional
	DiskType VolumeDiskType `json:"diskType,omitempty"`

	// SizeGB defines the size of the volume in GB
	// +kubebuilder:validation:Minimum=5
	SizeGB int `json:"sizeGB"`

	// AvailabilityZone is the availabilityZone where the volume will be created.
	// +kubebuilder:default=AUTO
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// SSHKeys contains a set of public ssh keys which will be added to the
	// list of authorized keys.
	// +listType=set
	SSHKeys []string `json:"SSHKeys,omitempty"`
}

// IonosCloudMachineStatus defines the observed state of IonosCloudMachine.
type IonosCloudMachineStatus struct {
	// Ready indicates the machine has been provisioned and is ready.
	// +optional
	Ready bool `json:"ready"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of IonosCloudMachines
	// can be added as events to the IonosCloudMachine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of IonosCloudMachines
	// can be added as events to the IonosCloudMachine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the IonosCloudMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IonosCloudMachine is the Schema for the ionoscloudmachines API.
type IonosCloudMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IonosCloudMachineSpec   `json:"spec,omitempty"`
	Status IonosCloudMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IonosCloudMachineList contains a list of IonosCloudMachine.
type IonosCloudMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IonosCloudMachine `json:"items"`
}

// GetConditions returns the observations of the operational state of the ProxmoxMachine resource.
func (m *IonosCloudMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the underlying service state of the ProxmoxMachine to the predescribed clusterv1.Conditions.
func (m *IonosCloudMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IonosCloudMachine{}, &IonosCloudMachineList{})
}
