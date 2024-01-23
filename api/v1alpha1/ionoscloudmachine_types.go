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
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// IonosCloudMachineType is the named type for the API object.
	IonosCloudMachineType = "IonosCloudMachine"

	// MachineFinalizer is the finalizer for the IonosCloudMachine resources.
	// It will prevent the deletion of the resource until it was removed by the controller
	// to ensure that related cloud resources will be deleted before the IonosCloudMachine resource
	// will be removed from the API server.
	MachineFinalizer = "ionoscloudmachine.infrastructure.cluster.x-k8s.io"

	// MachineProvisionedCondition documents the status of the provisioning of a IonosCloudMachine and
	// the underlying VM.
	MachineProvisionedCondition clusterv1.ConditionType = "MachineProvisioned"

	// WaitingForClusterInfrastructureReason (Severity=Info) indicates that the IonosCloudMachine is currently
	// waiting for the cluster infrastructure to become ready.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"

	// WaitingForBootstrapDataReason (Severity=Info) indicates that the bootstrap provider has not yet finished
	// creating the bootstrap data secret and store it in the Cluster API Machine.
	WaitingForBootstrapDataReason = "WaitingForBootstrapData"
)

// VolumeDiskType specifies the type of  hard disk.
type VolumeDiskType string

const (
	// VolumeDiskTypeHDD defines the disk type HDD.
	VolumeDiskTypeHDD VolumeDiskType = "HDD"
	// VolumeDiskTypeSSDStandard defines the standard SSD disk type.
	// This is the same as VolumeDiskTypeSSD.
	VolumeDiskTypeSSDStandard VolumeDiskType = "SSD Standard"
	// VolumeDiskTypeSSDPremium defines the premium SSD disk type.
	VolumeDiskTypeSSDPremium VolumeDiskType = "SSD Premium"
)

// AvailabilityZone is the availability zone where different cloud resources are created in.
type AvailabilityZone string

const (
	// AvailabilityZoneAuto automatically selects an availability zone.
	AvailabilityZoneAuto AvailabilityZone = "AUTO"
	// AvailabilityZoneOne zone 1.
	AvailabilityZoneOne AvailabilityZone = "ZONE_1"
	// AvailabilityZoneTwo zone 2.
	AvailabilityZoneTwo AvailabilityZone = "ZONE_2"
	// AvailabilityZoneThree zone 3.
	AvailabilityZoneThree AvailabilityZone = "ZONE_3"
)

// IonosCloudMachineSpec defines the desired state of IonosCloudMachine.
type IonosCloudMachineSpec struct {
	// ProviderID is the IONOS Cloud provider ID
	// will be in the format ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a
	// +optional
	ProviderID string `json:"providerID,omitempty"`

	// DatacenterID is the ID of the data center where the machine should be created in.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="datacenterID is immutable"
	// +kubebuilder:validation:Format=uuid
	DatacenterID string `json:"datacenterID"`

	// NumCores defines the number of cores for the VM.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	NumCores int32 `json:"numCores,omitempty"`

	// AvailabilityZone is the availability zone in which the VM should be provisioned.
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2
	// +kubebuilder:default=AUTO
	// +optional
	AvailabilityZone AvailabilityZone `json:"availabilityZone,omitempty"`

	// MemoryMB is the memory size for the VM in MB.
	// Size must be specified in multiples of 256 MB with a minimum of 1024 MB
	// which is required as we are using hot-pluggable RAM by default.
	// +kubebuilder:validation:MultipleOf=1024
	// +kubebuilder:validation:Minimum=2048
	// +kubebuilder:default=3072
	// +optional
	MemoryMB int32 `json:"memoryMB,omitempty"`

	// CPUFamily defines the CPU architecture, which will be used for this VM.
	// Not all CPU architectures are available in all data centers.
	// +kubebuilder:example=AMD_OPTERON
	// +kubebuilder:validation:MinLength=1
	CPUFamily string `json:"cpuFamily"`

	// Disk defines the boot volume of the VM.
	Disk *Volume `json:"disk"`

	// AdditionalNetworks defines the additional network configurations for the VM.
	// NOTE(lubedacht): We currently only support networks with DHCP enabled.
	// +optional
	AdditionalNetworks Networks `json:"additionalNetworks,omitempty"`
}

// Networks contains a list of network configs.
type Networks []Network

// Network contains a network config.
type Network struct {
	// NetworkID represents an ID an existing LAN in the data center.
	// This LAN will be excluded from the deletion process.
	// +kubebuilder:validation:Minimum=1
	NetworkID int32 `json:"networkID"`
}

// Volume is the physical storage on the machine.
type Volume struct {
	// Name is the name of the volume
	// +optional
	Name string `json:"name,omitempty"`

	// DiskType defines the type of the hard drive.
	// +kubebuilder:validation:Enum=HDD;SSD Standard;SSD Premium
	// +kubebuilder:default=HDD
	// +optional
	DiskType VolumeDiskType `json:"diskType,omitempty"`

	// SizeGB defines the size of the volume in GB
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:default=20
	// +optional
	SizeGB int `json:"sizeGB,omitempty"`

	// AvailabilityZone is the availability zone where the volume will be created.
	// +kubebuilder:validation:Enum=AUTO;ZONE_1;ZONE_2;ZONE_3
	// +kubebuilder:default=AUTO
	// +optional
	AvailabilityZone AvailabilityZone `json:"availabilityZone,omitempty"`

	// SSHKeys contains a set of public SSH keys which will be added to the
	// list of authorized keys.
	// +listType=set
	// +optional
	SSHKeys []string `json:"sshKeys,omitempty"`
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

	// CurrentRequest shows the current provisioning request for any
	// cloud resource that is being provisioned.
	// +optional
	CurrentRequest *ProvisioningRequest `json:"currentRequest,omitempty"`
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

// GetConditions returns the observations of the operational state of the IonosCloudMachine resource.
func (m *IonosCloudMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the underlying service state of the IonosCloudMachine to the predescribed clusterv1.Conditions.
func (m *IonosCloudMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&IonosCloudMachine{}, &IonosCloudMachineList{})
}
