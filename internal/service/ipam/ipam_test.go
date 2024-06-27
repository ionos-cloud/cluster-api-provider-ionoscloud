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

package ipam

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

type IpamTestSuite struct {
	*require.Assertions
	suite.Suite
	k8sClient    client.Client
	ctx          context.Context
	machineScope *scope.Machine
	clusterScope *scope.Cluster
	log          logr.Logger
	service      *cloud.Service
	ipamHelper   *Helper
	capiCluster  *clusterv1.Cluster
	capiMachine  *clusterv1.Machine
	infraCluster *infrav1.IonosCloudCluster
	infraMachine *infrav1.IonosCloudMachine
	ionosClient  *clienttest.MockClient
}

func (s *IpamTestSuite) SetupSuite() {
	s.log = logr.Discard()
	s.ctx = context.Background()
	s.Assertions = s.Require()
}

func (s *IpamTestSuite) SetupTest() {
	var err error
	s.ionosClient = clienttest.NewMockClient(s.T())

	s.capiCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-cluster",
			UID:       "uid",
		},
		Spec: clusterv1.ClusterSpec{},
	}
	s.infraCluster = &infrav1.IonosCloudCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      s.capiCluster.Name,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: s.capiCluster.Name,
			},
		},
		Spec: infrav1.IonosCloudClusterSpec{
			Location: "de/txl",
		},
		Status: infrav1.IonosCloudClusterStatus{},
	}
	s.capiMachine = &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-machine",
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: s.capiCluster.Name,
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: s.capiCluster.Name,
			Version:     ptr.To("v1.26.12"),
			ProviderID:  ptr.To("ionos://dd426c63-cd1d-4c02-aca3-13b4a27c2ebf"),
		},
	}
	s.infraMachine = &infrav1.IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-machine",
			Labels: map[string]string{
				clusterv1.ClusterNameLabel:           s.capiCluster.Name,
				clusterv1.MachineDeploymentNameLabel: "test-md",
			},
		},
		Spec: infrav1.IonosCloudMachineSpec{
			ProviderID:       ptr.To("ionos://dd426c63-cd1d-4c02-aca3-13b4a27c2ebf"),
			DatacenterID:     "ccf27092-34e8-499e-a2f5-2bdee9d34a12",
			NumCores:         2,
			AvailabilityZone: infrav1.AvailabilityZoneAuto,
			MemoryMB:         4096,
			CPUFamily:        ptr.To("AMD_OPTERON"),
			Disk: &infrav1.Volume{
				Name:             "test-machine-hdd",
				DiskType:         infrav1.VolumeDiskTypeHDD,
				SizeGB:           20,
				AvailabilityZone: infrav1.AvailabilityZoneAuto,
				Image: &infrav1.ImageSpec{
					ID: "3e3e3e3e-3e3e-3e3e-3e3e-3e3e3e3e3e3e",
				},
			},
			Type: infrav1.ServerTypeEnterprise,
		},
		Status: infrav1.IonosCloudMachineStatus{},
	}

	scheme := runtime.NewScheme()
	s.NoError(clusterv1.AddToScheme(scheme), "failed to extend scheme with Cluster API types")
	s.NoError(ipamv1.AddToScheme(scheme), "failed to extend scheme with Cluster API ipam types")
	s.NoError(infrav1.AddToScheme(scheme), "failed to extend scheme with IonosCloud types")
	s.NoError(clientgoscheme.AddToScheme(scheme))

	initObjects := []client.Object{s.infraMachine, s.infraCluster, s.capiCluster, s.capiMachine}
	s.k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjects...).
		WithStatusSubresource(initObjects...).
		Build()

	s.ipamHelper = NewHelper(s.k8sClient, s.log)
	s.clusterScope, err = scope.NewCluster(scope.ClusterParams{
		Client:       s.k8sClient,
		Cluster:      s.capiCluster,
		IonosCluster: s.infraCluster,
		Locker:       locker.New(),
	})
	s.NoError(err, "failed to create cluster scope")

	s.machineScope, err = scope.NewMachine(scope.MachineParams{
		Client:       s.k8sClient,
		Machine:      s.capiMachine,
		ClusterScope: s.clusterScope,
		IonosMachine: s.infraMachine,
		Locker:       locker.New(),
	})
	s.NoError(err, "failed to create machine scope")

	s.service, err = cloud.NewService(s.ionosClient, s.log)
	s.NoError(err, "failed to create service")
}

func TestIpamTestSuite(t *testing.T) {
	suite.Run(t, new(IpamTestSuite))
}

func (s *IpamTestSuite) TestReconcileIPAddressesDontCreateClaim() {
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	s.False(requeue)
	s.NoError(err)

	// No PoolRefs provided, so the Reconcile must not create a claim.
	list := &ipamv1.IPAddressClaimList{}
	err = s.k8sClient.List(s.ctx, list)
	s.Empty(list.Items)
	s.NoError(err)
}

func (s *IpamTestSuite) TestReconcileIPAddressesPrimaryIpv4CreateClaim() {
	poolRef := defaultInClusterIPv4PoolRef()

	s.machineScope.IonosMachine.Spec.IPv4PoolRef = poolRef
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	// IPAddressClaim was created, so we need to wait for the IPAddress to be created externally.
	s.True(requeue)
	s.NoError(err)

	claim := defaultPrimaryIPv4Claim()
	err = s.k8sClient.Get(s.ctx, client.ObjectKeyFromObject(claim), claim)
	s.NoError(err)
}

func (s *IpamTestSuite) TestReconcileIPAddressesPrimaryIpv6CreateClaim() {
	poolRef := defaultInClusterIPv6PoolRef()

	s.machineScope.IonosMachine.Spec.IPv6PoolRef = poolRef
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	// IPAddressClaim was created, so we need to wait for the IPAddress to be created externally.
	s.True(requeue)
	s.NoError(err)

	claim := defaultPrimaryIPv6Claim()
	err = s.k8sClient.Get(s.ctx, client.ObjectKeyFromObject(claim), claim)
	s.NoError(err)
}

func (s *IpamTestSuite) TestReconcileIPAddressesPrimaryIpv4GetIPFromClaim() {
	poolRef := defaultInClusterIPv4PoolRef()

	claim := defaultPrimaryIPv4Claim()
	claim.Status.AddressRef.Name = "nic-test-machine-ipv4-10-0-0-2"
	err := s.k8sClient.Create(s.ctx, claim)
	s.NoError(err)

	ip := defaultIPv4Address(claim, poolRef)
	err = s.k8sClient.Create(s.ctx, ip)
	s.NoError(err)

	s.machineScope.IonosMachine.Spec.IPv4PoolRef = poolRef
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal("10.0.0.2", s.machineScope.IonosMachine.Status.MachineNetworkInfo.NICInfo[0].IPv4Addresses[0])
}

func (s *IpamTestSuite) TestReconcileIPAddressesPrimaryIpv6GetIPFromClaim() {
	poolRef := defaultInClusterIPv6PoolRef()

	claim := defaultPrimaryIPv6Claim()
	claim.Status.AddressRef.Name = "nic-test-machine-ipv6-2001-db8--"
	err := s.k8sClient.Create(s.ctx, claim)
	s.NoError(err)

	ip := defaultIPv6Address(claim, poolRef)
	err = s.k8sClient.Create(s.ctx, ip)
	s.NoError(err)

	s.machineScope.IonosMachine.Spec.IPv6PoolRef = poolRef
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal("2001:db8::", s.machineScope.IonosMachine.Status.MachineNetworkInfo.NICInfo[0].IPv6Addresses[0])
}

func (s *IpamTestSuite) TestReconcileIPAddressesAdditionalIpv4CreateClaim() {
	poolRef := defaultInClusterIPv4PoolRef()

	s.machineScope.IonosMachine.Spec.AdditionalNetworks = defaultAdditionalNetworksIpv4(poolRef)
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	// IPAddressClaim was created, so we need to wait for the IPAddress to be created externally.
	s.True(requeue)
	s.NoError(err)

	claim := defaultAdditionalIPv4Claim()
	err = s.k8sClient.Get(s.ctx, client.ObjectKeyFromObject(claim), claim)
	s.NoError(err)
}

func (s *IpamTestSuite) TestReconcileIPAddressesAdditionalIpv6CreateClaim() {
	poolRef := defaultInClusterIPv6PoolRef()

	s.machineScope.IonosMachine.Spec.AdditionalNetworks = defaultAdditionalNetworksIpv6(poolRef)
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	// IPAddressClaim was created, so we need to wait for the IPAddress to be created externally.
	s.True(requeue)
	s.NoError(err)

	claim := defaultAdditionalIPv6Claim()
	err = s.k8sClient.Get(s.ctx, client.ObjectKeyFromObject(claim), claim)
	s.NoError(err)
}

func (s *IpamTestSuite) TestReconcileIPAddressesAdditionalIpv6GetIPFromClaim() {
	poolRef := defaultInClusterIPv6PoolRef()

	claim := defaultAdditionalIPv6Claim()
	claim.Status.AddressRef.Name = "nic-test-machine-ipv6-2001-db8--"
	err := s.k8sClient.Create(s.ctx, claim)
	s.NoError(err)

	ip := defaultIPv6Address(claim, poolRef)
	err = s.k8sClient.Create(s.ctx, ip)
	s.NoError(err)

	s.machineScope.IonosMachine.Spec.AdditionalNetworks = defaultAdditionalNetworksIpv6(poolRef)
	requeue, err := s.ipamHelper.ReconcileIPAddresses(s.ctx, s.machineScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal("2001:db8::", s.machineScope.IonosMachine.Status.MachineNetworkInfo.NICInfo[1].IPv6Addresses[0])
}

func defaultInClusterIPv4PoolRef() *corev1.TypedLocalObjectReference {
	return &corev1.TypedLocalObjectReference{
		APIGroup: ptr.To("ipam.cluster.x-k8s.io"),
		Kind:     "InClusterIPPool",
		Name:     "incluster-ipv4-pool",
	}
}

func defaultInClusterIPv6PoolRef() *corev1.TypedLocalObjectReference {
	return &corev1.TypedLocalObjectReference{
		APIGroup: ptr.To("ipam.cluster.x-k8s.io"),
		Kind:     "InClusterIPPool",
		Name:     "incluster-ipv6-pool",
	}
}

func defaultIPv4Address(claim *ipamv1.IPAddressClaim, poolRef *corev1.TypedLocalObjectReference) *ipamv1.IPAddress {
	return &ipamv1.IPAddress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nic-test-machine-ipv4-10-0-0-2",
			Namespace: "default",
		},
		Spec: ipamv1.IPAddressSpec{
			ClaimRef: *localRef(claim),
			PoolRef:  *poolRef,
			Address:  "10.0.0.2",
			Prefix:   16,
		},
	}
}

func defaultIPv6Address(claim *ipamv1.IPAddressClaim, poolRef *corev1.TypedLocalObjectReference) *ipamv1.IPAddress {
	return &ipamv1.IPAddress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nic-test-machine-ipv6-2001-db8--",
			Namespace: "default",
		},
		Spec: ipamv1.IPAddressSpec{
			ClaimRef: *localRef(claim),
			PoolRef:  *poolRef,
			Address:  "2001:db8::",
			Prefix:   42,
		},
	}
}

func defaultPrimaryIPv4Claim() *ipamv1.IPAddressClaim {
	return &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nic-test-machine-ipv4",
			Namespace: "default",
		},
	}
}

func defaultAdditionalIPv4Claim() *ipamv1.IPAddressClaim {
	return &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nic-test-machine-1-ipv4",
			Namespace: "default",
		},
	}
}

func defaultAdditionalIPv6Claim() *ipamv1.IPAddressClaim {
	return &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nic-test-machine-1-ipv6",
			Namespace: "default",
		},
	}
}

func defaultAdditionalNetworksIpv6(poolRef *corev1.TypedLocalObjectReference) []infrav1.Network {
	return []infrav1.Network{{
		NetworkID: 1,
		IPAMConfig: infrav1.IPAMConfig{
			IPv6PoolRef: poolRef,
		},
	}}
}

func defaultAdditionalNetworksIpv4(poolRef *corev1.TypedLocalObjectReference) []infrav1.Network {
	return []infrav1.Network{{
		NetworkID: 1,
		IPAMConfig: infrav1.IPAMConfig{
			IPv4PoolRef: poolRef,
		},
	}}
}

func defaultPrimaryIPv6Claim() *ipamv1.IPAddressClaim {
	return &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nic-test-machine-ipv6",
			Namespace: "default",
		},
	}
}

func localRef(obj client.Object) *corev1.LocalObjectReference {
	return &corev1.LocalObjectReference{
		Name: obj.GetName(),
	}
}
