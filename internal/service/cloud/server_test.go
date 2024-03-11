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

	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type serverSuite struct {
	ServiceTestSuite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

func (s *serverSuite) TestServerName() {
	serverName := s.service.serverName(s.infraMachine)
	s.Equal("k8s-default-test-machine", serverName)
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

	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusQueued)}, nil)
	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateBusy() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Busy),
			},
			Properties: &sdk.ServerProperties{
				Name: ptr.To(s.service.serverName(s.infraMachine)),
			},
		},
	}}, nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateAvailable() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Available),
			},
			Properties: &sdk.ServerProperties{
				Name:    ptr.To(s.service.serverName(s.infraMachine)),
				VmState: ptr.To("RUNNING"),
			},
		},
	}}, nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
}

func (s *serverSuite) TestReconcileServerNoRequest() {
	s.prepareReconcileServerRequestTest()
	s.mockGetServerCreationRequest().Return([]sdk.Request{}, nil)
	s.mockCreateServer().Return(&sdk.Server{Id: ptr.To("12345")}, "location/to/server", nil)
	s.mockListLANs().Return(&sdk.Lans{Items: &[]sdk.Lan{{
		Id: ptr.To("1"),
		Properties: &sdk.LanProperties{
			Name:   ptr.To(s.service.lanName(s.clusterScope)),
			Public: ptr.To(true),
		},
	}}}, nil)

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.Equal("ionos://12345", ptr.Deref(s.machineScope.IonosMachine.Spec.ProviderID, ""))
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) prepareReconcileServerRequestTest() {
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
	s.mockListServers().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()
}

func (s *serverSuite) TestReconcileServerDeletion() {
	s.mockGetServer(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	reqLocation := "delete/location"

	s.mockGetServerDeletionRequest(exampleServerID).Return(nil, nil)
	s.mockDeleteServer(exampleServerID).Return(reqLocation, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)
	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodDelete)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, reqLocation)
}

func (s *serverSuite) TestReconcileServerDeletionServerNotFound() {
	s.mockGetServer(exampleServerID).Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404))
	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListServers().Return(&sdk.Servers{}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(res)
}

func (s *serverSuite) TestReconcileServerDeletionUnexpectedError() {
	s.mockGetServer(exampleServerID).Return(nil, sdk.NewGenericOpenAPIError("unexpected error returned", nil, nil, 500))
	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.Error(err)
	s.False(res)
}

func (s *serverSuite) TestReconcileServerDeletionCreateRequestPending() {
	s.mockGetServer(exampleServerID).Return(nil, nil)
	s.mockListServers().Return(&sdk.Servers{}, nil)
	exampleRequest := s.examplePostRequest(sdk.RequestStatusQueued)
	s.mockGetServerCreationRequest().Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodPost)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, *exampleRequest.Metadata.RequestStatus.Href)
}

func (s *serverSuite) TestReconcileServerDeletionRequestPending() {
	s.mockGetServer(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusQueued, exampleServerID)

	s.mockGetServerDeletionRequest(exampleServerID).Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodDelete)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, *exampleRequest.Metadata.RequestStatus.Href)
}

func (s *serverSuite) TestReconcileServerDeletionRequestDone() {
	s.mockGetServer(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusDone, exampleServerID)

	s.mockGetServerDeletionRequest(exampleServerID).Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(res)

	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileServerDeletionRequestFailed() {
	s.mockGetServer(exampleServerID).Return(&sdk.Server{
		Id: ptr.To(exampleServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusFailed, exampleServerID)

	s.mockGetServerDeletionRequest(exampleServerID).Return([]sdk.Request{exampleRequest}, nil)
	s.mockDeleteServer(exampleServerID).Return("delete/triggered", nil)

	res, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodDelete)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, "delete/triggered")
}

func (s *serverSuite) TestGetServerWithProviderID() {
	serverID := exampleServerID
	s.mockGetServer(serverID).Return(&sdk.Server{}, nil)
	server, err := s.service.getServer(s.machineScope)(s.ctx)
	s.NoError(err)
	s.NotNil(server)
}

func (s *serverSuite) TestGetServerWithProviderIDNotFound() {
	serverID := exampleServerID
	s.mockGetServer(serverID).Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404))
	s.mockListServers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: nil,
		},
	}}, nil)

	server, err := s.service.getServer(s.machineScope)(s.ctx)
	s.ErrorAs(err, &sdk.GenericOpenAPIError{})
	s.Nil(server)
}

func (s *serverSuite) TestGetServerWithoutProviderIDFoundInList() {
	serverName := s.service.serverName(s.infraMachine)
	s.machineScope.IonosMachine.Spec.ProviderID = nil
	s.mockListServers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: &sdk.ServerProperties{
				Name: ptr.To(serverName),
			},
		},
	}}, nil)

	server, err := s.service.getServer(s.machineScope)(s.ctx)
	s.NoError(err)
	s.NotNil(server)
}

//nolint:unused
func (s *serverSuite) exampleServer() sdk.Server {
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

func (s *serverSuite) mockListServers() *clienttest.MockClient_ListServers_Call {
	return s.ionosClient.EXPECT().ListServers(s.ctx, s.machineScope.DatacenterID())
}

func (s *serverSuite) mockGetServer(serverID string) *clienttest.MockClient_GetServer_Call {
	return s.ionosClient.EXPECT().GetServer(s.ctx, s.machineScope.DatacenterID(), serverID)
}

func (s *serverSuite) mockDeleteServer(serverID string) *clienttest.MockClient_DeleteServer_Call {
	return s.ionosClient.EXPECT().DeleteServer(s.ctx, s.machineScope.DatacenterID(), serverID)
}

func (s *serverSuite) mockGetServerCreationRequest() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, s.service.serversURL(s.machineScope.DatacenterID()))
}

func (s *serverSuite) mockGetServerDeletionRequest(serverID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx,
		http.MethodDelete, path.Join(s.service.serversURL(s.machineScope.DatacenterID()), serverID))
}

func (s *serverSuite) mockCreateServer() *clienttest.MockClient_CreateServer_Call {
	return s.ionosClient.EXPECT().CreateServer(
		s.ctx,
		s.machineScope.DatacenterID(),
		mock.Anything,
		mock.Anything,
	)
}

func (s *serverSuite) mockListLANs() *clienttest.MockClient_ListLANs_Call {
	return s.ionosClient.EXPECT().ListLANs(s.ctx, s.machineScope.DatacenterID())
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
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.serverName(s.infraMachine)),
		href:       exampleRequestPath,
		targetID:   exampleServerID,
		targetType: sdk.SERVER,
	}
	return s.exampleRequest(opts)
}
