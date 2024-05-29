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
	"fmt"
	"net/http"
	"path"
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type serverSuite struct {
	ServiceTestSuite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

func (s *serverSuite) TestVolumeName() {
	volumeName := s.service.volumeName(s.infraMachine)
	expected := "vol-" + s.infraMachine.Name
	s.Equal(expected, volumeName)
}

func (s *serverSuite) TestReconcileServerNoBootstrapSecret() {
	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.True(requeue)
	s.Error(err)

	s.machineScope.Machine.Spec.Bootstrap.DataSecretName = ptr.To("test")
	requeue, err = s.service.ReconcileServer(s.ctx, s.machineScope)
	s.False(requeue)
	s.NoError(err)
}

func (s *serverSuite) TestReconcileServerRequestPending() {
	s.prepareReconcileServerRequestTest()

	s.mockGetServerCreationRequestCall().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusQueued)}, nil)
	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateBusy() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Busy),
			},
			Properties: &sdk.ServerProperties{
				Name: ptr.To(s.infraMachine.Name),
			},
		},
	}}, nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateAvailable() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Available),
			},
			Properties: &sdk.ServerProperties{
				Name:    ptr.To(s.infraMachine.Name),
				VmState: ptr.To("RUNNING"),
			},
			Entities: &sdk.ServerEntities{
				Nics: &sdk.Nics{
					Items: &[]sdk.Nic{{
						Properties: &sdk.NicProperties{
							Name:          ptr.To(s.service.nicName(s.infraMachine)),
							Dhcp:          ptr.To(true),
							Lan:           ptr.To(int32(1)),
							Ips:           ptr.To([]string{"198.51.100.10"}),
							Ipv6CidrBlock: ptr.To("2001:db8:2c0:301::/64"),
							Ipv6Ips:       ptr.To([]string{"2001:db8:2c0:301::1"}),
						},
					}},
				},
			},
		},
	}}, nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)

	s.NotNil(s.machineScope.IonosMachine.Status.MachineNetworkInfo)
	s.Equal([]string{"198.51.100.10"}, s.machineScope.IonosMachine.Status.MachineNetworkInfo.NICInfo[0].IPv4Addresses)
	s.Equal([]string{"2001:db8:2c0:301::1"},
		s.machineScope.IonosMachine.Status.MachineNetworkInfo.NICInfo[0].IPv6Addresses)
	s.Equal(int32(1), s.machineScope.IonosMachine.Status.MachineNetworkInfo.NICInfo[0].NetworkID)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateAvailableTurnedOff() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Id: ptr.To(exampleServerID),
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Available),
			},
			Properties: &sdk.ServerProperties{
				Name:    ptr.To(s.infraMachine.Name),
				VmState: ptr.To("SHUTOFF"),
			},
		},
	}}, nil).Once()

	s.mockStartServerCall().Return("", nil)

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileEnterpriseServerNoRequest() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{}, nil)
	s.mockCreateServerCall(infrav1.ServerTypeEnterprise).Return(&sdk.Server{Id: ptr.To("12345")}, "location/to/server", nil)
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{{
		Id: ptr.To("1"),
		Properties: &sdk.LanProperties{
			Name:   ptr.To(s.service.lanName(s.clusterScope.Cluster)),
			Public: ptr.To(true),
		},
	}}}, nil)

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.Equal("ionos://12345", ptr.Deref(s.machineScope.IonosMachine.Spec.ProviderID, ""))
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileVCPUServerNoRequest() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{}, nil)
	s.mockCreateServerCall(infrav1.ServerTypeVCPU).Return(&sdk.Server{Id: ptr.To("12345")}, "location/to/server", nil)
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{{
		Id: ptr.To("1"),
		Properties: &sdk.LanProperties{
			Name:   ptr.To(s.service.lanName(s.clusterScope.Cluster)),
			Public: ptr.To(true),
		},
	}}}, nil)

	s.infraMachine.Spec.Type = infrav1.ServerTypeVCPU
	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.Equal("ionos://12345", ptr.Deref(s.machineScope.IonosMachine.Spec.ProviderID, ""))
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) prepareReconcileServerRequestTest() {
	s.T().Helper()
	bootstrapSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: metav1.NamespaceDefault,
		},
		Data: map[string][]byte{
			"value": []byte("test"),
		},
	}

	s.NoError(s.k8sClient.Create(s.ctx, bootstrapSecret))

	s.machineScope.Machine.Spec.Bootstrap.DataSecretName = ptr.To("test")
	s.machineScope.IonosMachine.Spec.ProviderID = nil
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()
}

func (s *serverSuite) TestReconcileServerDeletion() {
	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	reqLocation := "delete/location"

	s.mockGetServerDeletionRequestCall(exampleServerID).Return(nil, nil)
	s.mockDeleteServerCall(exampleServerID, false).Return(reqLocation, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(res, err, reqLocation)
}

func (s *serverSuite) TestReconcileServerDeletionDeleteBootVolume() {
	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
		Properties: &sdk.ServerProperties{
			BootVolume: &sdk.ResourceReference{
				Id: ptr.To(exampleBootVolumeID),
			},
		},
	}, nil).Once()

	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil).Once()

	reqLocationVolume := "delete/location/volume"
	reqLocationServer := "delete/location/server"

	s.mockDeleteVolumeCall(exampleBootVolumeID).Return(reqLocationVolume, nil).Once()

	s.mockGetServerDeletionRequestCall(exampleServerID).Return(nil, nil)
	s.mockDeleteServerCall(exampleServerID, false).Return(reqLocationServer, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(res, err, reqLocationVolume)

	res, err = s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(res, err, reqLocationServer)
}

func (s *serverSuite) TestReconcileServerDeletionDeleteAllVolumes() {
	s.clusterScope.Cluster.DeletionTimestamp = ptr.To(metav1.Now())
	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
		Properties: &sdk.ServerProperties{
			BootVolume: &sdk.ResourceReference{
				Id: ptr.To(exampleBootVolumeID),
			},
		},
	}, nil).Once()

	reqLocation := "delete/location"
	s.mockGetServerDeletionRequestCall(exampleServerID).Return(nil, nil)
	s.mockDeleteServerCall(exampleServerID, true).Return(reqLocation, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(res, err, reqLocation)
}

func (s *serverSuite) validateSuccessfulDeletionResponse(success bool, err error, requestLocation string) {
	s.T().Helper()

	s.NoError(err)
	s.True(success)
	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(http.MethodDelete, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, requestLocation)
}

func (s *serverSuite) TestReconcileServerDeletionServerNotFound() {
	s.mockGetServerCall(exampleServerID).Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404))
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServersCall().Return(&sdk.Servers{}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(res)
}

func (s *serverSuite) TestReconcileServerDeletionUnexpectedError() {
	internalError := sdk.NewGenericOpenAPIError("unexpected error returned", nil, nil, 500)
	s.mockGetServerCall(exampleServerID).Return(
		nil,
		internalError,
	)
	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.Error(err)
	s.False(res)
}

func (s *serverSuite) TestReconcileServerDeletionCreateRequestPending() {
	s.mockGetServerCall(exampleServerID).Return(nil, nil)
	s.mockListServersCall().Return(&sdk.Servers{}, nil)
	exampleRequest := s.examplePostRequest(sdk.RequestStatusQueued)
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(http.MethodPost, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, *exampleRequest.Metadata.RequestStatus.Href)
}

func (s *serverSuite) TestReconcileServerDeletionRequestPending() {
	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusQueued, exampleServerID)

	s.mockGetServerDeletionRequestCall(exampleServerID).Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(http.MethodDelete, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, *exampleRequest.Metadata.RequestStatus.Href)
}

func (s *serverSuite) TestReconcileServerDeletionRequestDone() {
	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusDone, exampleServerID)

	s.mockGetServerDeletionRequestCall(exampleServerID).Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(res)

	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileServerDeletionRequestFailed() {
	s.mockGetServerCall(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusFailed, exampleServerID)

	s.mockGetServerDeletionRequestCall(exampleServerID).Return([]sdk.Request{exampleRequest}, nil)
	s.mockDeleteServerCall(exampleServerID, false).Return("delete/triggered", nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(http.MethodDelete, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal("delete/triggered", s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath)
}

func (s *serverSuite) TestGetServerWithProviderID() {
	serverID := exampleServerID
	s.mockGetServerCall(serverID).Return(&sdk.Server{}, nil)
	server, err := s.service.getServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.NotNil(server)
}

func (s *serverSuite) TestGetServerWithProviderIDNotFound() {
	serverID := exampleServerID
	s.mockGetServerCall(serverID).Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404))
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: nil,
		},
	}}, nil)

	server, err := s.service.getServer(s.ctx, s.machineScope)
	s.ErrorAs(err, &sdk.GenericOpenAPIError{})
	s.Nil(server)
}

func (s *serverSuite) TestGetServerWithoutProviderIDFoundInList() {
	s.machineScope.IonosMachine.Spec.ProviderID = nil
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: &sdk.ServerProperties{
				Name: ptr.To(s.infraMachine.Name),
			},
		},
	}}, nil)

	server, err := s.service.getServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.NotNil(server)
}

//nolint:unused
func (*serverSuite) exampleServer() sdk.Server {
	return sdk.Server{
		Id: ptr.To("1"),
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Properties: &sdk.ServerProperties{
			AvailabilityZone: ptr.To("AUTO"),
			BootVolume: &sdk.ResourceReference{
				Id:   ptr.To("1"),
				Type: ptr.To(sdk.VOLUME),
			},
			Name:    nil,
			VmState: nil,
		},
	}
}

func (s *serverSuite) mockListServersCall() *clienttest.MockClient_ListServers_Call {
	return s.ionosClient.EXPECT().ListServers(s.ctx, s.machineScope.DatacenterID())
}

func (s *serverSuite) mockDeleteVolumeCall(volumeID string) *clienttest.MockClient_DeleteVolume_Call {
	return s.ionosClient.EXPECT().DeleteVolume(s.ctx, s.machineScope.DatacenterID(), volumeID)
}

func (s *serverSuite) mockDeleteServerCall(serverID string, deleteVolumes bool) *clienttest.MockClient_DeleteServer_Call {
	return s.ionosClient.EXPECT().DeleteServer(s.ctx, s.machineScope.DatacenterID(), serverID, deleteVolumes)
}

func (s *serverSuite) mockGetServerCreationRequestCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().
		GetRequests(s.ctx, http.MethodPost, s.service.serversURL(s.machineScope.DatacenterID()))
}

func (s *serverSuite) mockGetServerDeletionRequestCall(serverID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx,
		http.MethodDelete, path.Join(s.service.serversURL(s.machineScope.DatacenterID()), serverID))
}

func (s *serverSuite) mockCreateServerCall(serverType infrav1.ServerType) *clienttest.MockClient_CreateServer_Call {
	return s.ionosClient.EXPECT().CreateServer(
		s.ctx,
		s.machineScope.DatacenterID(),
		mock.MatchedBy(hasServerType(serverType)),
		mock.Anything,
	)
}

func (s *serverSuite) mockStartServerCall() *clienttest.MockClient_StartServer_Call {
	return s.ionosClient.EXPECT().StartServer(s.ctx, s.machineScope.DatacenterID(), exampleServerID)
}

func (s *serverSuite) exampleDeleteRequest(status, serverID string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodDelete,
		url:        path.Join(s.service.serversURL(s.machineScope.DatacenterID()), serverID),
		href:       path.Join(exampleRequestPath, exampleServerID),
		targetID:   exampleServerID,
		targetType: sdk.SERVER,
	}
	return s.exampleRequest(opts)
}

func (s *serverSuite) examplePostRequest(status string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPost,
		url:        s.service.serversURL(s.machineScope.DatacenterID()),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.infraMachine.Name),
		href:       exampleRequestPath,
		targetID:   exampleServerID,
		targetType: sdk.SERVER,
	}
	return s.exampleRequest(opts)
}

func hasServerType(serverType infrav1.ServerType) func(properties sdk.ServerProperties) bool {
	return func(properties sdk.ServerProperties) bool {
		return ptr.Deref(properties.Type, "") == serverType.String()
	}
}
