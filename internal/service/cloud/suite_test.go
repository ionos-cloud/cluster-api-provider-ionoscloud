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

package cloud

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// NOTE(lubedacht): Choice of IP addresses for unit tests
// https://datatracker.ietf.org/doc/rfc5737/
// 3.  Documentation Address Blocks
//
//	The blocks 192.0.2.0/24 (TEST-NET-1), 198.51.100.0/24 (TEST-NET-2),
//	and 203.0.113.0/24 (TEST-NET-3) are provided for use in
//	documentation.
const (
	// The expected endpoint IP.
	exampleEndpointIP = "203.0.113.1"
	// Used when we actually expect the endpoint IP but receive this instead.
	exampleUnexpectedIP = "203.0.113.10"
	// Used to test cases where a LAN already contains configurations with other IP addresses
	// to ensure that the service does not overwrite them.
	exampleArbitraryIP     = "203.0.113.11"
	exampleDHCPIP          = "192.0.2.2"
	exampleSecondaryDHCPIP = "192.0.2.3"
)

const (
	exampleLANID             = "42"
	exampleNICID             = "f3b3f8e4-3b6d-4b6d-8f1d-3e3e6e3e3e3e"
	exampleSecondaryNICID    = "f3b3f8e4-3b6d-4b6d-8f1d-3e3e6e3e3e3d"
	exampleIPBlockID         = "f882d597-4ee2-4b89-b01a-cbecd0f513d8"
	exampleServerID          = "dd426c63-cd1d-4c02-aca3-13b4a27c2ebf"
	exampleSecondaryServerID = "dd426c63-cd1d-4c02-aca3-13b4a27c2ebd"
	exampleRequestPath       = "/test"
	exampleLocation          = "de/txl"
)

type ServiceTestSuite struct {
	*require.Assertions
	suite.Suite
	k8sClient    client.Client
	ctx          context.Context
	machineScope *scope.Machine
	clusterScope *scope.Cluster
	log          logr.Logger
	service      *Service
	capiCluster  *clusterv1.Cluster
	capiMachine  *clusterv1.Machine
	infraCluster *infrav1.IonosCloudCluster
	infraMachine *infrav1.IonosCloudMachine
	ionosClient  *clienttest.MockClient
}

func (s *ServiceTestSuite) SetupSuite() {
	s.log = logr.Discard()
	s.ctx = context.Background()
	s.Assertions = s.Require()
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupTest() {
	var err error
	s.ionosClient = clienttest.NewMockClient(s.T())

	s.capiCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-cluster",
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
			ContractNumber: "12345678",
			Location:       "de/txl",
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
				clusterv1.ClusterNameLabel: s.capiCluster.Name,
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
					ID: ptr.To("3e3e3e3e-3e3e-3e3e-3e3e-3e3e3e3e3e3e"),
				},
			},
		},
		Status: infrav1.IonosCloudMachineStatus{},
	}

	scheme := runtime.NewScheme()
	s.NoError(clusterv1.AddToScheme(scheme), "failed to extend scheme with Cluster API types")
	s.NoError(infrav1.AddToScheme(scheme), "failed to extend scheme with IonosCloud types")
	s.NoError(clientgoscheme.AddToScheme(scheme))

	initObjects := []client.Object{s.infraMachine, s.infraCluster, s.capiCluster, s.capiMachine}
	s.k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjects...).
		WithStatusSubresource(initObjects...).
		Build()

	s.clusterScope, err = scope.NewCluster(scope.ClusterParams{
		Client:       s.k8sClient,
		Cluster:      s.capiCluster,
		IonosCluster: s.infraCluster,
	})
	s.NoError(err, "failed to create cluster scope")

	s.machineScope, err = scope.NewMachine(scope.MachineParams{
		Client:       s.k8sClient,
		Machine:      s.capiMachine,
		ClusterScope: s.clusterScope,
		IonosMachine: s.infraMachine,
	})
	s.NoError(err, "failed to create machine scope")

	s.service, err = NewService(s.ionosClient, s.log)
	s.NoError(err, "failed to create service")
}

type requestBuildOptions struct {
	status,
	method,
	url,
	body,
	href,
	requestID,
	targetID string
	targetType sdk.Type
}

func (s *ServiceTestSuite) exampleRequest(opts requestBuildOptions) sdk.Request {
	req := sdk.Request{
		Id: &opts.requestID,
		Metadata: &sdk.RequestMetadata{
			RequestStatus: &sdk.RequestStatus{
				Href: &opts.href,
				Metadata: &sdk.RequestStatusMetadata{
					Status:  &opts.status,
					Message: ptr.To("test"),
				},
			},
		},
		Properties: &sdk.RequestProperties{
			Url:    &opts.url,
			Method: &opts.method,
			Body:   &opts.body,
		},
	}

	if opts.targetType != "" || opts.targetID != "" {
		req.Metadata.RequestStatus.Metadata.Targets = &[]sdk.RequestTarget{
			{
				Target: &sdk.ResourceReference{
					Id:   &opts.targetID,
					Type: &opts.targetType,
				},
			},
		}
	}

	return req
}

func (s *ServiceTestSuite) defaultServer(m *infrav1.IonosCloudMachine, ips ...string) *sdk.Server {
	return &sdk.Server{
		Id: ptr.To(exampleServerID),
		Entities: &sdk.ServerEntities{
			Nics: &sdk.Nics{
				Items: &[]sdk.Nic{{
					Id: ptr.To(exampleNICID),
					Properties: &sdk.NicProperties{
						Dhcp: ptr.To(true),
						Name: ptr.To(s.service.nicName(m)),
						Ips:  &ips,
					},
				}},
			},
		},
	}
}
