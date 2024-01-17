package cloud

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

var errMock = errors.New("this is an error")

type ServiceTestSuite struct {
	*require.Assertions
	suite.Suite
	k8sClient    client.Client
	ctx          context.Context
	machineScope *scope.MachineScope
	clusterScope *scope.ClusterScope
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
	s.ionosClient = &clienttest.MockClient{}

	s.capiCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-cluster",
		},
		Spec: clusterv1.ClusterSpec{
			Paused: false,
		},
	}
	s.infraCluster = &infrav1.IonosCloudCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      s.capiCluster.Name,
		},
		Spec: infrav1.IonosCloudClusterSpec{
			ContractNumber: "12345678",
		},
		Status: infrav1.IonosCloudClusterStatus{},
	}
	s.capiMachine = &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.capiCluster.Namespace,
			Name:      "test-machine",
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
		},
		Spec: infrav1.IonosCloudMachineSpec{
			ProviderID:       "ionos://8c19a898-fda9-4783-a939-d778aeee217f",
			DataCenterID:     "ccf27092-34e8-499e-a2f5-2bdee9d34a12",
			NumCores:         2,
			AvailabilityZone: infrav1.AvailabilityZoneAuto,
			MemoryMB:         4096,
			CPUFamily:        "AMD_OPTERON",
			Disk: infrav1.Volume{
				Name:             "test-machine-hdd",
				DiskType:         infrav1.VolumeDiskTypeHDD,
				SizeGB:           20,
				AvailabilityZone: infrav1.AvailabilityZoneAuto,
				SSHKeys:          []string{"ssh-rsa AAAAB3Nz"},
			},
			Network: &infrav1.Network{
				IPs:     []string{"1.2.3.4"},
				UseDHCP: ptr.To(true),
			},
		},
		Status: infrav1.IonosCloudMachineStatus{},
	}

	scheme := runtime.NewScheme()
	s.NoError(clusterv1.AddToScheme(scheme), "failed to extend scheme with Cluster API types")
	s.NoError(infrav1.AddToScheme(scheme), "failed to extend scheme with IonosCloud types")

	s.k8sClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(s.infraMachine, s.infraCluster, s.capiCluster, s.capiMachine).
		WithStatusSubresource(
			&infrav1.IonosCloudMachine{}, &infrav1.IonosCloudCluster{}, &clusterv1.Cluster{}, &clusterv1.Machine{}).
		Build()

	s.clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       s.k8sClient,
		Logger:       &s.log,
		Cluster:      s.capiCluster,
		IonosCluster: s.infraCluster,
		IonosClient:  s.ionosClient,
	})
	s.NoError(err, "failed to create cluster scope")

	s.machineScope, err = scope.NewMachineScope(scope.MachineScopeParams{
		Client:       s.k8sClient,
		Logger:       &s.log,
		Cluster:      s.capiCluster,
		Machine:      s.capiMachine,
		ClusterScope: s.clusterScope,
		IonosMachine: s.infraMachine,
	})
	s.NoError(err, "failed to create machine scope")

	s.service, err = NewService(s.ctx, s.machineScope)
	s.NoError(err, "failed to create service")
}
