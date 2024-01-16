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
	"errors"
	"github.com/go-logr/logr"
	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"path/filepath"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient    client.Client
	ctx          = context.Background()
	err          error
	machineScope *scope.MachineScope
	log          logr.Logger
	service      *Service
	capiCluster  *clusterv1.Cluster
	capiMachine  *clusterv1.Machine
	clusterScope *scope.ClusterScope
	infraCluster *infrav1.IonosCloudCluster
	infraMachine *infrav1.IonosCloudMachine
	ionosClient  *clienttest.MockClient
	mockErr      = errors.New("this is an error")
)

func TestAPIs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping as only short tests should run")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1alpha1 API Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
		},
		// NOTE(gfariasalves): To be removed after I finish the PR comments
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s", "1.28.0-linux-amd64"),

		ErrorIfCRDPathMissing: true,
	}

	scheme := runtime.NewScheme()
	Expect(infrav1.AddToScheme(scheme)).To(Succeed())

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	DeferCleanup(func() {
		By("tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())
	})

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	log = logf.FromContext(ctx)
})

var _ = BeforeEach(func() {
	err = nil
	ionosClient = &clienttest.MockClient{}
	capiCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-cluster",
		},
		Spec: clusterv1.ClusterSpec{
			Paused: false,
		},
	}
	infraCluster = &infrav1.IonosCloudCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceDefault,
			Name:      "test-cluster",
		},
		Spec: infrav1.IonosCloudClusterSpec{
			ContractNumber: "12345678",
		},
	}
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       k8sClient,
		Logger:       &log,
		Cluster:      capiCluster,
		IonosCluster: infraCluster,
		IonosClient:  ionosClient,
	})
	Expect(err).ToNot(HaveOccurred(), "failed to create cluster scope")
	capiMachine = &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: capiCluster.Namespace,
			Name:      "test-machine",
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: capiCluster.Name,
			Version:     ptr.To("v1.26.12"),
			ProviderID:  ptr.To("ionos://dd426c63-cd1d-4c02-aca3-13b4a27c2ebf"),
		},
	}
	infraMachine = &infrav1.IonosCloudMachine{
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
	}
	machineScope, err = scope.NewMachineScope(scope.MachineScopeParams{
		Client:       k8sClient,
		Logger:       &log,
		Cluster:      capiCluster,
		Machine:      capiMachine,
		ClusterScope: clusterScope,
		IonosMachine: infraMachine,
	})
	Expect(err).ToNot(HaveOccurred(), "failed to create machine scope")
	service, err = NewService(ctx, machineScope)
	Expect(err).ToNot(HaveOccurred(), "failed to create service")
})

var _ = BeforeEach(func() {
	//err = k8sClient.Create(ctx, capiCluster)
	//Expect(err).ToNot(HaveOccurred(), "could not create CAPI cluster")
	err = k8sClient.Create(ctx, infraCluster)
	Expect(err).ToNot(HaveOccurred(), "could not create infra cluster")
	//err = k8sClient.Create(ctx, capiMachine)
	//Expect(err).ToNot(HaveOccurred(), "could not create CAPI machine")
	err = k8sClient.Create(ctx, infraMachine)
	Expect(err).ToNot(HaveOccurred(), "could not create infra machine")
})

var _ = AfterEach(func() {
	err = k8sClient.Delete(ctx, infraMachine)
	Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred(), "could not delete infra machine")
	//err = k8sClient.Delete(ctx, capiMachine)
	//Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred(), "could not delete CAPI machine")
	err = k8sClient.Delete(ctx, infraCluster)
	Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred(), "could not delete infra cluster")
	//err = k8sClient.Delete(ctx, capiCluster)
	//Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred(), "could not delete CAPI cluster")
})

var _ = Context("Helper functions", func() {
	It("can return the correct datacenter ID", func() {
		Expect(service.dataCenterID()).To(Equal(infraMachine.Spec.DataCenterID))
	})
	It("can return the API", func() {
		Expect(service.api()).To(Equal(ionosClient))
	})
})
