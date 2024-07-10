//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSpec.
func (in *ImageSpec) DeepCopy() *ImageSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudCluster) DeepCopyInto(out *IonosCloudCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudCluster.
func (in *IonosCloudCluster) DeepCopy() *IonosCloudCluster {
	if in == nil {
		return nil
	}
	out := new(IonosCloudCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterList) DeepCopyInto(out *IonosCloudClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IonosCloudCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterList.
func (in *IonosCloudClusterList) DeepCopy() *IonosCloudClusterList {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterSpec) DeepCopyInto(out *IonosCloudClusterSpec) {
	*out = *in
	out.ControlPlaneEndpoint = in.ControlPlaneEndpoint
	out.CredentialsRef = in.CredentialsRef
	if in.LoadBalancerProviderRef != nil {
		in, out := &in.LoadBalancerProviderRef, &out.LoadBalancerProviderRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterSpec.
func (in *IonosCloudClusterSpec) DeepCopy() *IonosCloudClusterSpec {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterStatus) DeepCopyInto(out *IonosCloudClusterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CurrentRequestByDatacenter != nil {
		in, out := &in.CurrentRequestByDatacenter, &out.CurrentRequestByDatacenter
		*out = make(map[string]ProvisioningRequest, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.CurrentClusterRequest != nil {
		in, out := &in.CurrentClusterRequest, &out.CurrentClusterRequest
		*out = new(ProvisioningRequest)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterStatus.
func (in *IonosCloudClusterStatus) DeepCopy() *IonosCloudClusterStatus {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterTemplate) DeepCopyInto(out *IonosCloudClusterTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterTemplate.
func (in *IonosCloudClusterTemplate) DeepCopy() *IonosCloudClusterTemplate {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudClusterTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterTemplateList) DeepCopyInto(out *IonosCloudClusterTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IonosCloudCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterTemplateList.
func (in *IonosCloudClusterTemplateList) DeepCopy() *IonosCloudClusterTemplateList {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudClusterTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterTemplateResource) DeepCopyInto(out *IonosCloudClusterTemplateResource) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterTemplateResource.
func (in *IonosCloudClusterTemplateResource) DeepCopy() *IonosCloudClusterTemplateResource {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudClusterTemplateSpec) DeepCopyInto(out *IonosCloudClusterTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudClusterTemplateSpec.
func (in *IonosCloudClusterTemplateSpec) DeepCopy() *IonosCloudClusterTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(IonosCloudClusterTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudLoadBalancer) DeepCopyInto(out *IonosCloudLoadBalancer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudLoadBalancer.
func (in *IonosCloudLoadBalancer) DeepCopy() *IonosCloudLoadBalancer {
	if in == nil {
		return nil
	}
	out := new(IonosCloudLoadBalancer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudLoadBalancer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudLoadBalancerList) DeepCopyInto(out *IonosCloudLoadBalancerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IonosCloudLoadBalancer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudLoadBalancerList.
func (in *IonosCloudLoadBalancerList) DeepCopy() *IonosCloudLoadBalancerList {
	if in == nil {
		return nil
	}
	out := new(IonosCloudLoadBalancerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudLoadBalancerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudLoadBalancerSpec) DeepCopyInto(out *IonosCloudLoadBalancerSpec) {
	*out = *in
	out.LoadBalancerEndpoint = in.LoadBalancerEndpoint
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudLoadBalancerSpec.
func (in *IonosCloudLoadBalancerSpec) DeepCopy() *IonosCloudLoadBalancerSpec {
	if in == nil {
		return nil
	}
	out := new(IonosCloudLoadBalancerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudLoadBalancerStatus) DeepCopyInto(out *IonosCloudLoadBalancerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudLoadBalancerStatus.
func (in *IonosCloudLoadBalancerStatus) DeepCopy() *IonosCloudLoadBalancerStatus {
	if in == nil {
		return nil
	}
	out := new(IonosCloudLoadBalancerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachine) DeepCopyInto(out *IonosCloudMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachine.
func (in *IonosCloudMachine) DeepCopy() *IonosCloudMachine {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineList) DeepCopyInto(out *IonosCloudMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IonosCloudMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineList.
func (in *IonosCloudMachineList) DeepCopy() *IonosCloudMachineList {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineSpec) DeepCopyInto(out *IonosCloudMachineSpec) {
	*out = *in
	if in.ProviderID != nil {
		in, out := &in.ProviderID, &out.ProviderID
		*out = new(string)
		**out = **in
	}
	if in.CPUFamily != nil {
		in, out := &in.CPUFamily, &out.CPUFamily
		*out = new(string)
		**out = **in
	}
	if in.Disk != nil {
		in, out := &in.Disk, &out.Disk
		*out = new(Volume)
		(*in).DeepCopyInto(*out)
	}
	if in.AdditionalNetworks != nil {
		in, out := &in.AdditionalNetworks, &out.AdditionalNetworks
		*out = make(Networks, len(*in))
		copy(*out, *in)
	}
	if in.FailoverIP != nil {
		in, out := &in.FailoverIP, &out.FailoverIP
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineSpec.
func (in *IonosCloudMachineSpec) DeepCopy() *IonosCloudMachineSpec {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineStatus) DeepCopyInto(out *IonosCloudMachineStatus) {
	*out = *in
	if in.MachineNetworkInfo != nil {
		in, out := &in.MachineNetworkInfo, &out.MachineNetworkInfo
		*out = new(MachineNetworkInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.FailureReason != nil {
		in, out := &in.FailureReason, &out.FailureReason
		*out = new(errors.MachineStatusError)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CurrentRequest != nil {
		in, out := &in.CurrentRequest, &out.CurrentRequest
		*out = new(ProvisioningRequest)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineStatus.
func (in *IonosCloudMachineStatus) DeepCopy() *IonosCloudMachineStatus {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineTemplate) DeepCopyInto(out *IonosCloudMachineTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineTemplate.
func (in *IonosCloudMachineTemplate) DeepCopy() *IonosCloudMachineTemplate {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudMachineTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineTemplateList) DeepCopyInto(out *IonosCloudMachineTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IonosCloudMachineTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineTemplateList.
func (in *IonosCloudMachineTemplateList) DeepCopy() *IonosCloudMachineTemplateList {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IonosCloudMachineTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineTemplateResource) DeepCopyInto(out *IonosCloudMachineTemplateResource) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineTemplateResource.
func (in *IonosCloudMachineTemplateResource) DeepCopy() *IonosCloudMachineTemplateResource {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IonosCloudMachineTemplateSpec) DeepCopyInto(out *IonosCloudMachineTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IonosCloudMachineTemplateSpec.
func (in *IonosCloudMachineTemplateSpec) DeepCopy() *IonosCloudMachineTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(IonosCloudMachineTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineNetworkInfo) DeepCopyInto(out *MachineNetworkInfo) {
	*out = *in
	if in.NICInfo != nil {
		in, out := &in.NICInfo, &out.NICInfo
		*out = make([]NICInfo, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineNetworkInfo.
func (in *MachineNetworkInfo) DeepCopy() *MachineNetworkInfo {
	if in == nil {
		return nil
	}
	out := new(MachineNetworkInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NICInfo) DeepCopyInto(out *NICInfo) {
	*out = *in
	if in.IPv4Addresses != nil {
		in, out := &in.IPv4Addresses, &out.IPv4Addresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.IPv6Addresses != nil {
		in, out := &in.IPv6Addresses, &out.IPv6Addresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NICInfo.
func (in *NICInfo) DeepCopy() *NICInfo {
	if in == nil {
		return nil
	}
	out := new(NICInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Network) DeepCopyInto(out *Network) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Network.
func (in *Network) DeepCopy() *Network {
	if in == nil {
		return nil
	}
	out := new(Network)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in Networks) DeepCopyInto(out *Networks) {
	{
		in := &in
		*out = make(Networks, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Networks.
func (in Networks) DeepCopy() Networks {
	if in == nil {
		return nil
	}
	out := new(Networks)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProvisioningRequest) DeepCopyInto(out *ProvisioningRequest) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProvisioningRequest.
func (in *ProvisioningRequest) DeepCopy() *ProvisioningRequest {
	if in == nil {
		return nil
	}
	out := new(ProvisioningRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Volume) DeepCopyInto(out *Volume) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		*out = new(ImageSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Volume.
func (in *Volume) DeepCopy() *Volume {
	if in == nil {
		return nil
	}
	out := new(Volume)
	in.DeepCopyInto(out)
	return out
}
