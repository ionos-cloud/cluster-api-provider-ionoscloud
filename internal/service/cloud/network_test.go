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
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/suite"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type lanSuite struct {
	ServiceTestSuite
}

func TestLANSuite(t *testing.T) {
	suite.Run(t, new(lanSuite))
}

func (s *lanSuite) TestNetworkLANName() {
	s.Equal("k8s-default-test-cluster", s.service.lanName())
}

func (s *lanSuite) TestLANURL() {
	s.Equal("datacenters/"+s.service.datacenterID()+"/lans/1", s.service.lanURL("1"))
}

func (s *lanSuite) TestLANURLs() {
	s.Equal("datacenters/"+s.service.datacenterID()+"/lans", s.service.lansURL())
}

func (s *lanSuite) Test_Network_CreateLAN_Successful() {
	s.mockCreateLANCall().Return(reqPath, nil).Once()
	s.NoError(s.service.createLAN())
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.datacenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodPost, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *lanSuite) Test_Network_DeleteLAN_Successful() {
	s.mockDeleteLANCall(lanID).Return(reqPath, nil).Once()
	s.NoError(s.service.deleteLAN(lanID))
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.datacenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodDelete, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *lanSuite) Test_Network_GetLAN_Successful() {
	lan := s.exampleLAN()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	foundLAN, err := s.service.getLAN()
	s.NoError(err)
	s.NotNil(foundLAN)
	s.Equal(lan, *foundLAN)
}

func (s *lanSuite) Test_Network_GetLAN_NotFound() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	lan, err := s.service.getLAN()
	s.NoError(err)
	s.Nil(lan)
}

func (s *lanSuite) Test_Network_GetLAN_Error_NotUnique() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN(), s.exampleLAN()}}, nil).Once()
	lan, err := s.service.getLAN()
	s.Error(err)
	s.Nil(lan)
}

func (s *lanSuite) Test_Network_RemoveLANPendingRequestFromCluster_Successful() {
	s.infraCluster.Status.CurrentRequestByDatacenter = map[string]infrav1.ProvisioningRequest{
		s.service.datacenterID(): {
			RequestPath: reqPath,
			Method:      http.MethodDelete,
			State:       sdk.RequestStatusQueued,
		},
	}
	s.NoError(s.service.removeLANPendingRequestFromCluster())
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID(), "request should be removed from status")
}

func (s *lanSuite) Test_Network_RemoveLANPendingRequestFromCluster_NoRequest() {
	s.NoError(s.service.removeLANPendingRequestFromCluster())
}

func (s *lanSuite) Test_Network_ReconcileLAN_NoExistingLAN_NoRequest_Create() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.mockCreateLANCall().Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLAN_NoExistingLAN_ExistingRequest_Pending() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusQueued), nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLAN_ExistingLAN() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLAN_ExistingLAN_Unavailable() {
	lan := s.exampleLAN()
	lan.Metadata.State = ptr.To("BUSY")
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLANDelete_LANExists_NoPendingRequests_NoOtherUsers_Delete() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.mockGetLANDeletionRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.mockDeleteLANCall(lanID).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLANDelete_LANExists_NoPendingRequests_HasOtherUsers_NoDelete() {
	lan := s.exampleLAN()
	lan.Entities.Nics.Items = &[]sdk.Nic{{Id: ptr.To("1")}}
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	s.mockGetLANDeletionRequestsCall().Return([]sdk.Request{}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID())
}

func (s *lanSuite) Test_Network_ReconcileLANDelete_NoExistingLAN_ExistingRequest_Pending() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusQueued), nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLANDelete_LANExists_ExistingRequest_Pending() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	requests := s.exampleDeleteRequest(sdk.RequestStatusQueued)
	s.mockGetLANDeletionRequestsCall().Return(requests, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) Test_Network_ReconcileLANDelete_LANDoesNotExist() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return(nil, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID())
}

const (
	reqPath = "this/is/a/path"
	lanID   = "1"
)

func (s *lanSuite) exampleLAN() sdk.Lan {
	return sdk.Lan{
		Id: ptr.To(lanID),
		Properties: &sdk.LanProperties{
			Name: ptr.To(s.service.lanName()),
		},
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(stateAvailable),
		},
		Entities: &sdk.LanEntities{
			Nics: &sdk.LanNics{
				Items: &[]sdk.Nic{},
			},
		},
	}
}

func (s *lanSuite) examplePostRequest(status string) []sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPost,
		url:        s.service.lansURL(),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.lanName()),
		href:       reqPath,
		targetID:   lanID,
		targetType: sdk.LAN,
	}
	return []sdk.Request{s.exampleRequest(opts)}
}

func (s *lanSuite) exampleDeleteRequest(status string) []sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodDelete,
		url:        s.service.lanURL(lanID),
		href:       reqPath,
		targetID:   lanID,
		targetType: sdk.LAN,
	}
	return []sdk.Request{s.exampleRequest(opts)}
}

func (s *lanSuite) mockCreateLANCall() *clienttest.MockClient_CreateLAN_Call {
	return s.ionosClient.EXPECT().CreateLAN(s.ctx, s.service.datacenterID(), sdk.LanPropertiesPost{
		Name:   ptr.To(s.service.lanName()),
		Public: ptr.To(true),
	})
}

func (s *lanSuite) mockDeleteLANCall(id string) *clienttest.MockClient_DeleteLAN_Call {
	return s.ionosClient.EXPECT().DeleteLAN(s.ctx, s.service.datacenterID(), id)
}

func (s *lanSuite) mockListLANsCall() *clienttest.MockClient_ListLANs_Call {
	return s.ionosClient.EXPECT().ListLANs(s.ctx, s.service.datacenterID())
}

func (s *lanSuite) mockGetLANCreationRequestsCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, s.service.lansURL())
}

func (s *lanSuite) mockGetLANDeletionRequestsCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodDelete, s.service.lanURL(lanID))
}
