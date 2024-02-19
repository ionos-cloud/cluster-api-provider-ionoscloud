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

const testServerID = "dd426c63-cd1d-4c02-aca3-13b4a27c2ebf"

type serverSuite struct {
	ServiceTestSuite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

func (s *serverSuite) TestServerName() {
	serverName := s.service.serverName()
	s.Equal("k8s-default-test-machine", serverName)
}

func (s *serverSuite) TestReconcileServerNoBootstrapSecret() {
	requeue, err := s.service.ReconcileServer()
	s.True(requeue)
	s.Error(err)

	s.machineScope.Machine.Spec.Bootstrap.DataSecretName = ptr.To("test")
	requeue, err = s.service.ReconcileServer()
	s.False(requeue)
	s.NoError(err)
}

func (s *serverSuite) TestReconcileServerRequestPending() {
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
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: nil,
		},
	}}, nil)

	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusQueued)}, nil)
	requeue, err := s.service.ReconcileServer()
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateBusy() {
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
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()
	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Busy),
			},
			Properties: &sdk.ServerProperties{
				Name: ptr.To(s.service.serverName()),
			},
		},
	}}, nil).Once()

	requeue, err := s.service.ReconcileServer()
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerRequestDoneStateAvailable() {
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
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()
	s.mockGetServerCreationRequest().Return([]sdk.Request{s.examplePostRequest(sdk.RequestStatusDone)}, nil)
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Available),
			},
			Properties: &sdk.ServerProperties{
				Name:    ptr.To(s.service.serverName()),
				VmState: ptr.To("RUNNING"),
			},
		},
	}}, nil).Once()

	requeue, err := s.service.ReconcileServer()
	s.NoError(err)
	s.False(requeue)
}

func (s *serverSuite) TestReconcileServerNoRequest() {
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
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()
	s.mockGetServerCreationRequest().Return([]sdk.Request{}, nil)
	s.mockCreateServer().Return(&sdk.Server{Id: ptr.To("12345")}, "location/to/sever", nil)
	s.mockListLANs().Return(&sdk.Lans{Items: &[]sdk.Lan{{
		Id: ptr.To("1"),
		Properties: &sdk.LanProperties{
			Name:   ptr.To(s.service.lanName()),
			Public: ptr.To(true),
		},
	}}}, nil)

	requeue, err := s.service.ReconcileServer()
	s.Equal("ionos://12345", ptr.Deref(s.machineScope.IonosMachine.Spec.ProviderID, ""))
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerDeletion() {
	s.mockGetServer(testServerID).Return(&sdk.Server{
		Id: ptr.To(testServerID),
	}, nil)

	reqLocation := "delete/location"

	s.mockGetServerDeletionRequest(testServerID).Return(nil, nil)
	s.mockDeleteServer(testServerID).Return(reqLocation, nil)

	res, err := s.service.ReconcileServerDeletion()
	s.NoError(err)
	s.True(res)
	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodDelete)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, reqLocation)
}

func (s *serverSuite) TestReconcileServerDeletionCreateRequestPending() {
	s.mockGetServer(testServerID).Return(nil, nil)
	s.mockListSevers().Return(&sdk.Servers{}, nil)
	exampleRequest := s.examplePostRequest(sdk.RequestStatusQueued)
	s.mockGetServerCreationRequest().Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion()
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodPost)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, *exampleRequest.Metadata.RequestStatus.Href)
}

func (s *serverSuite) TestReconcileServerDeletionRequestPending() {
	s.mockGetServer(testServerID).Return(&sdk.Server{
		Id: ptr.To(testServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusQueued, testServerID)

	s.mockGetServerDeletionRequest(testServerID).Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion()
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodDelete)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, *exampleRequest.Metadata.RequestStatus.Href)
}

func (s *serverSuite) TestReconcileServerDeletionRequestDone() {
	s.mockGetServer(testServerID).Return(&sdk.Server{
		Id: ptr.To(testServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusDone, testServerID)

	s.mockGetServerDeletionRequest(testServerID).Return([]sdk.Request{exampleRequest}, nil)

	res, err := s.service.ReconcileServerDeletion()
	s.NoError(err)
	s.False(res)

	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileServerDeletionRequestFailed() {
	s.mockGetServer(testServerID).Return(&sdk.Server{
		Id: ptr.To(testServerID),
	}, nil)

	exampleRequest := s.exampleDeleteRequest(sdk.RequestStatusFailed, testServerID)

	s.mockGetServerDeletionRequest(testServerID).Return([]sdk.Request{exampleRequest}, nil)
	s.mockDeleteServer(testServerID).Return("delete/triggered", nil)

	res, err := s.service.ReconcileServerDeletion()
	s.NoError(err)
	s.True(res)

	s.NotNil(s.machineScope.IonosMachine.Status.CurrentRequest)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.Method, http.MethodDelete)
	s.Equal(s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath, "delete/triggered")
}

func (s *serverSuite) TestGetServerWithProviderID() {
	serverID := testServerID
	s.mockGetServer(serverID).Return(&sdk.Server{}, nil)
	server, err := s.service.getServer()
	s.NoError(err)
	s.NotNil(server)
}

func (s *serverSuite) TestGetServerWithProviderIDNotFound() {
	serverID := testServerID
	s.mockGetServer(serverID).Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404))
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: nil,
		},
	}}, nil)

	server, err := s.service.getServer()
	s.NoError(err)
	s.Nil(server)
}

func (s *serverSuite) TestGetServerWithoutProviderIDFoundInList() {
	serverName := s.service.serverName()
	s.machineScope.IonosMachine.Spec.ProviderID = nil
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{
		{
			Properties: &sdk.ServerProperties{
				Name: ptr.To(serverName),
			},
		},
	}}, nil)

	server, err := s.service.getServer()
	s.NoError(err)
	s.NotNil(server)
}

func (s *serverSuite) TestGetServer() {
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil)
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

func (s *serverSuite) mockListSevers() *clienttest.MockClient_ListServers_Call {
	return s.ionosClient.EXPECT().ListServers(s.ctx, s.service.datacenterID())
}

func (s *serverSuite) mockGetServer(serverID string) *clienttest.MockClient_GetServer_Call {
	return s.ionosClient.EXPECT().GetServer(s.ctx, s.service.datacenterID(), serverID)
}

func (s *serverSuite) mockDeleteServer(serverID string) *clienttest.MockClient_DeleteServer_Call {
	return s.ionosClient.EXPECT().DeleteServer(s.ctx, s.service.datacenterID(), serverID)
}

func (s *serverSuite) mockGetServerCreationRequest() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, s.service.serversURL())
}

func (s *serverSuite) mockGetServerDeletionRequest(serverID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodDelete, path.Join(s.service.serversURL(), serverID))
}

func (s *serverSuite) mockCreateServer() *clienttest.MockClient_CreateServer_Call {
	return s.ionosClient.EXPECT().CreateServer(
		s.ctx,
		s.service.datacenterID(),
		mock.Anything,
		mock.Anything,
	)
}

func (s *serverSuite) mockListLANs() *clienttest.MockClient_ListLANs_Call {
	return s.ionosClient.EXPECT().ListLANs(s.ctx, s.service.datacenterID())
}

func (s *serverSuite) exampleDeleteRequest(status, serverID string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodDelete,
		url:        path.Join(s.service.serversURL(), serverID),
		href:       path.Join(reqPath, testServerID),
		targetID:   testServerID,
		targetType: sdk.SERVER,
	}
	return s.exampleRequest(opts)
}

func (s *serverSuite) examplePostRequest(status string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPost,
		url:        s.service.serversURL(),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.serverName()),
		href:       reqPath,
		targetID:   testServerID,
		targetType: sdk.SERVER,
	}
	return s.exampleRequest(opts)
}
