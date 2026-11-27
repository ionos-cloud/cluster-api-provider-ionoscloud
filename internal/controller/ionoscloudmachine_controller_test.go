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

package controller

import (
	"errors"
	"net/http"
	"path"
	"testing"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	conditions "sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	testDatacenterID = "ccf27092-34e8-499e-a2f5-2bdee9d34a12"
	testServerID     = "dd426c63-cd1d-4c02-aca3-13b4a27c2ebf"
	testLANID        = "1"
)

func TestSetProvisioningCondition(t *testing.T) {
	machineWith := func(status metav1.ConditionStatus, reason string, provisioned bool) *infrav1.IonosCloudMachine {
		m := &infrav1.IonosCloudMachine{}
		if provisioned {
			m.Status.Initialization.Provisioned = new(true)
		}
		conditions.Set(m, metav1.Condition{Type: infrav1.MachineProvisionedCondition, Status: status, Reason: reason})
		return m
	}

	// A stale "waiting" reason is replaced once provisioning starts.
	m := machineWith(metav1.ConditionFalse, infrav1.WaitingForBootstrapDataReason, false)
	setProvisioningCondition(m, nil)
	c := conditions.Get(m, infrav1.MachineProvisionedCondition)
	require.Equal(t, metav1.ConditionFalse, c.Status)
	require.Equal(t, infrav1.MachineProvisioningReason, c.Reason)

	// A failing step is surfaced with its error.
	setProvisioningCondition(m, errors.New("error in step ReconcileServer: image lookup failed"))
	c = conditions.Get(m, infrav1.MachineProvisionedCondition)
	require.Equal(t, metav1.ConditionFalse, c.Status)
	require.Equal(t, infrav1.MachineProvisioningFailedReason, c.Reason)
	require.Equal(t, "error in step ReconcileServer: image lookup failed", c.Message)

	// A provisioned machine keeps its True condition on a transient error.
	m = machineWith(metav1.ConditionTrue, infrav1.MachineProvisionedReason, true)
	setProvisioningCondition(m, errors.New("transient"))
	c = conditions.Get(m, infrav1.MachineProvisionedCondition)
	require.Equal(t, metav1.ConditionTrue, c.Status)
	require.Equal(t, infrav1.MachineProvisionedReason, c.Reason)
}

func TestReconcileNormalIgnitionWithoutBootstrapVolume(t *testing.T) {
	r, ms, ionosClient := newIgnitionMachineTestScope(t)
	ms.IonosMachine.Spec.ProviderID = ""
	ionosClient.EXPECT().ListLANs(mock.Anything, testDatacenterID).Return(&sdk.Lans{Items: &[]sdk.Lan{{
		Id:       new(testLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: new(sdk.Available)},
	}}}, nil)
	ionosClient.EXPECT().ListServers(mock.Anything, testDatacenterID).Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil)
	ionosClient.EXPECT().GetRequests(mock.Anything, http.MethodPost, path.Join("datacenters", testDatacenterID, "servers")).
		Return(nil, nil)

	for range 2 {
		res, err := r.reconcileNormal(t.Context(), ms, newCloudService(t, ionosClient))
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res, "must not requeue, only a spec change can resolve this")

		c := conditions.Get(ms.IonosMachine, infrav1.MachineProvisionedCondition)
		require.NotNil(t, c)
		require.Equal(t, metav1.ConditionFalse, c.Status)
		require.Equal(t, infrav1.BootstrapVolumeRequiredReason, c.Reason)
		require.Contains(t, c.Message, "spec.bootstrapVolume")
		require.Nil(t, ms.IonosMachine.Status.Initialization.Provisioned)
		require.Empty(t, ms.IonosMachine.Spec.ProviderID)
	}
}

func TestReconcileNormalBootstrapVolumeRemovalDoesNotBlockProvisioning(t *testing.T) {
	r, ms, ionosClient := newIgnitionMachineTestScope(t)
	ms.IonosMachine.Spec.BootstrapVolume = &infrav1.BootstrapVolume{Image: &infrav1.ImageSpec{ID: "cfg-image-id"}}
	ms.Machine.Status.NodeRef = clusterv1.MachineNodeReference{Name: ms.Machine.Name}
	ionosClient.EXPECT().ListLANs(mock.Anything, testDatacenterID).Return(&sdk.Lans{Items: &[]sdk.Lan{{
		Id:       new(testLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: new(sdk.Available)},
	}}}, nil)
	ionosClient.EXPECT().GetServer(mock.Anything, testDatacenterID, testServerID).Return(&sdk.Server{
		Id:         new(testServerID),
		Metadata:   &sdk.DatacenterElementMetadata{State: new(sdk.Available)},
		Properties: &sdk.ServerProperties{Name: new(ms.IonosMachine.Name), VmState: new("RUNNING")},
		Entities: &sdk.ServerEntities{Volumes: &sdk.AttachedVolumes{Items: &[]sdk.Volume{{
			Id:         new("volume-id"),
			Metadata:   &sdk.DatacenterElementMetadata{State: new(sdk.Available)},
			Properties: &sdk.VolumeProperties{Name: new("cfg-test-machine-37a8eec1")},
		}}}},
	}, nil).Times(2)
	ionosClient.EXPECT().GetRequests(mock.Anything, http.MethodDelete,
		path.Join("datacenters", testDatacenterID, "volumes", "volume-id")).Return(nil, nil)
	ionosClient.EXPECT().DeleteVolume(mock.Anything, testDatacenterID, "volume-id").
		Return("", sdk.NewGenericOpenAPIError("error", nil, nil, http.StatusInternalServerError))

	res, err := r.reconcileNormal(t.Context(), ms, newCloudService(t, ionosClient))
	require.ErrorContains(t, err, "error in step ReconcileBootstrapVolume")
	require.Equal(t, ctrl.Result{RequeueAfter: defaultReconcileDuration}, res)
	require.True(t, ptr.Deref(ms.IonosMachine.Status.Initialization.Provisioned, false))

	c := conditions.Get(ms.IonosMachine, infrav1.MachineProvisionedCondition)
	require.NotNil(t, c)
	require.Equal(t, metav1.ConditionTrue, c.Status)

	require.NoError(t, ms.Finalize())
	persisted := &infrav1.IonosCloudMachine{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(ms.IonosMachine), persisted))
	c = conditions.Get(persisted, infrav1.BootstrapVolumeRemovedCondition)
	require.NotNil(t, c)
	require.Equal(t, metav1.ConditionFalse, c.Status)
	require.Equal(t, infrav1.BootstrapVolumeRemovalFailedReason, c.Reason)
	require.Contains(t, c.Message, "failed to request bootstrap volume deletion")
}

func TestReconcileDeleteBootstrapVolumeWaitsForServerCreation(t *testing.T) {
	r, ms, ionosClient := newIgnitionMachineTestScope(t)
	ms.IonosMachine.Spec.BootstrapVolume = &infrav1.BootstrapVolume{Image: &infrav1.ImageSpec{ID: "cfg-image-id"}}
	ionosClient.EXPECT().GetServer(mock.Anything, testDatacenterID, testServerID).
		Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, http.StatusNotFound))
	ionosClient.EXPECT().ListServers(mock.Anything, testDatacenterID).Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil)
	ionosClient.EXPECT().GetRequests(mock.Anything, http.MethodPost, path.Join("datacenters", testDatacenterID, "servers")).
		Return([]sdk.Request{serverCreationRequest(ms.IonosMachine.Name, sdk.RequestStatusQueued)}, nil)

	res, err := r.reconcileDelete(t.Context(), ms, newCloudService(t, ionosClient))
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{RequeueAfter: defaultReconcileDuration}, res)
	require.Equal(t, http.MethodPost, ms.IonosMachine.Status.CurrentRequest.Method)
}

func TestReconcileDeleteBootstrapVolumeAfterServerDeletion(t *testing.T) {
	r, ms, ionosClient := newIgnitionMachineTestScope(t)
	ms.IonosMachine.Spec.BootstrapVolume = &infrav1.BootstrapVolume{Image: &infrav1.ImageSpec{ID: "cfg-image-id"}}
	ionosClient.EXPECT().GetServer(mock.Anything, testDatacenterID, testServerID).
		Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, http.StatusNotFound))
	ionosClient.EXPECT().ListServers(mock.Anything, testDatacenterID).Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil)
	ionosClient.EXPECT().GetRequests(mock.Anything, http.MethodPost, path.Join("datacenters", testDatacenterID, "servers")).
		Return(nil, nil)
	ionosClient.EXPECT().ListVolumes(mock.Anything, testDatacenterID).Return(&sdk.Volumes{Items: &[]sdk.Volume{{
		Id:         new("volume-id"),
		Properties: &sdk.VolumeProperties{Name: new("cfg-test-machine-37a8eec1")},
	}}}, nil)
	ionosClient.EXPECT().GetRequests(mock.Anything, http.MethodDelete,
		path.Join("datacenters", testDatacenterID, "volumes", "volume-id")).Return(nil, nil)
	ionosClient.EXPECT().DeleteVolume(mock.Anything, testDatacenterID, "volume-id").Return("delete/volume", nil)

	res, err := r.reconcileDelete(t.Context(), ms, newCloudService(t, ionosClient))
	require.NoError(t, err)
	require.Equal(t, ctrl.Result{RequeueAfter: defaultReconcileDuration}, res)
	require.Equal(t, http.MethodDelete, ms.IonosMachine.Status.CurrentRequest.Method)
	require.Equal(t, "delete/volume", ms.IonosMachine.Status.CurrentRequest.RequestPath)
}

// newIgnitionMachineTestScope returns a scope for a machine with bootstrap data in Ignition format,
// but without a bootstrap volume.
func newIgnitionMachineTestScope(t *testing.T) (*IonosCloudMachineReconciler, *scope.Machine, *clienttest.MockClient) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, clusterv1.AddToScheme(scheme))
	require.NoError(t, infrav1.AddToScheme(scheme))

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: metav1.NamespaceDefault},
		Status: clusterv1.ClusterStatus{
			Initialization: clusterv1.ClusterInitializationStatus{InfrastructureProvisioned: new(true)},
		},
	}
	ionosCluster := &infrav1.IonosCloudCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: metav1.NamespaceDefault},
	}
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: "test-machine", Namespace: metav1.NamespaceDefault},
		Spec: clusterv1.MachineSpec{
			ClusterName: cluster.Name,
			Bootstrap:   clusterv1.Bootstrap{DataSecretName: new("bootstrap")},
		},
	}
	ionosMachine := &infrav1.IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-machine",
			Namespace:  metav1.NamespaceDefault,
			Finalizers: []string{infrav1.MachineFinalizer},
		},
		Spec: infrav1.IonosCloudMachineSpec{
			ProviderID:   "ionos://" + testServerID,
			DatacenterID: testDatacenterID,
			NetworkID:    new(testLANID),
			Disk:         &infrav1.Volume{Image: &infrav1.ImageSpec{ID: "image-id"}},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bootstrap", Namespace: metav1.NamespaceDefault},
		Data: map[string][]byte{
			"format": []byte("ignition"),
			"value":  []byte(`{"ignition":{"version":"3.4.0"}}`),
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cluster, ionosCluster, machine, ionosMachine, secret).
		WithStatusSubresource(ionosMachine).
		Build()

	clusterScope, err := scope.NewCluster(scope.ClusterParams{
		Client:       cl,
		Cluster:      cluster,
		IonosCluster: ionosCluster,
		Locker:       locker.New(),
	})
	require.NoError(t, err)

	ms, err := scope.NewMachine(scope.MachineParams{
		Client:       cl,
		Machine:      machine,
		ClusterScope: clusterScope,
		IonosMachine: ionosMachine,
		Locker:       locker.New(),
	})
	require.NoError(t, err)

	return &IonosCloudMachineReconciler{Client: cl}, ms, clienttest.NewMockClient(t)
}

func newCloudService(t *testing.T, ionosClient *clienttest.MockClient) *cloud.Service {
	t.Helper()
	svc, err := cloud.NewService(ionosClient, logr.Discard())
	require.NoError(t, err)
	return svc
}

func serverCreationRequest(name, status string) sdk.Request {
	return sdk.Request{
		Metadata: &sdk.RequestMetadata{
			RequestStatus: &sdk.RequestStatus{
				Href: new("request/server"),
				Metadata: &sdk.RequestStatusMetadata{
					Status:  &status,
					Targets: &[]sdk.RequestTarget{{Target: &sdk.ResourceReference{Type: new(sdk.SERVER)}}},
				},
			},
		},
		Properties: &sdk.RequestProperties{
			Url:  new(path.Join("datacenters", testDatacenterID, "servers")),
			Body: new(`{"properties":{"name":"` + name + `"}}`),
		},
	}
}
