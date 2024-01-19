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

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/mock"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

func (s *ServiceTestSuite) TestNetworkLANName() {
	s.Equal("k8s-default-test-cluster", s.service.lanName())
}

func (s *ServiceTestSuite) Test_Network_CreateLAN_Successful() {
	s.createLANCall().Return(reqPath, nil).Once()
	s.NoError(s.service.createLAN())
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.dataCenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodPost, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *ServiceTestSuite) Test_Network_DeleteLAN_Successful() {
	s.deleteLANCall(lanID).Return(reqPath, nil).Once()
	s.NoError(s.service.deleteLAN(lanID))
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.dataCenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodDelete, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *ServiceTestSuite) Test_Network_GetLAN_Successful() {
	lan := s.exampleLAN()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	foundLAN, err := s.service.GetLAN()
	s.NoError(err)
	s.NotNil(foundLAN)
	s.Equal(lan, *foundLAN)
}

func (s *ServiceTestSuite) Test_Network_GetLAN_NotFound() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	lan, err := s.service.GetLAN()
	s.NoError(err)
	s.Nil(lan)
}

func (s *ServiceTestSuite) Test_Network_GetLAN_Error_NotUnique() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN(), s.exampleLAN()}}, nil).Once()
	lan, err := s.service.GetLAN()
	s.Error(err)
	s.Nil(lan)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_NoPendingRequest() {
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	request, err := s.service.checkForPendingLANRequest(http.MethodDelete, lanID)
	s.NoError(err)
	s.Empty(request)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_DeleteRequest() {
	s.getRequestsCall().Return([]sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Targets: &[]sdk.RequestTarget{
							{
								Target: &sdk.ResourceReference{
									Id: ptr.To(lanID),
								},
							},
						},
						Status:  ptr.To(sdk.RequestStatusQueued),
						Message: ptr.To("test"),
					},
				},
			},
		},
	}, nil).Once()
	status, err := s.service.checkForPendingLANRequest(http.MethodDelete, lanID)
	s.NoError(err)
	s.Equal(sdk.RequestStatusQueued, status)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_PostRequest() {
	s.getRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusQueued), nil).Once()
	status, err := s.service.checkForPendingLANRequest(http.MethodPost, "")
	s.NoError(err)
	s.Equal(sdk.RequestStatusQueued, status)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_Error_DifferentName_DeleteRequest() {
	requests := []sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Targets: &[]sdk.RequestTarget{
							{
								Target: &sdk.ResourceReference{
									Id: ptr.To("different"),
								},
							},
						},
						Status:  ptr.To(sdk.RequestStatusQueued),
						Message: ptr.To("test"),
					},
				},
			},
		},
	}
	s.getRequestsCall().Return(requests, nil).Once()
	status, err := s.service.checkForPendingLANRequest(http.MethodDelete, lanID)
	s.NoError(err)
	s.Empty(status)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_Error_DifferentName_PostRequest() {
	requests := []sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(sdk.RequestStatusQueued),
						Message: ptr.To("test"),
					},
				},
			},
			Properties: &sdk.RequestProperties{
				Method: ptr.To(http.MethodPost),
				Body:   ptr.To(`{"properties": {"name": "different"}}`),
			},
		},
	}
	s.getRequestsCall().Return(requests, nil).Once()
	status, err := s.service.checkForPendingLANRequest(http.MethodPost, "")
	s.NoError(err)
	s.Empty(status)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_Error_UnsupportedMethod() {
	request, err := s.service.checkForPendingLANRequest(http.MethodTrace, lanID)
	s.Error(err)
	s.Empty(request)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_Error_Delete_NoLanID() {
	request, err := s.service.checkForPendingLANRequest(http.MethodDelete, "")
	s.Error(err)
	s.Empty(request)
}

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_Error_Post_WithLanID() {
	request, err := s.service.checkForPendingLANRequest(http.MethodPost, lanID)
	s.Error(err)
	s.Empty(request)
}

func (s *ServiceTestSuite) Test_Network_RemoveLANPendingRequestFromCluster_Successful() {
	s.infraCluster.Status.CurrentRequestByDatacenter = map[string]infrav1.ProvisioningRequest{
		s.service.dataCenterID(): {
			RequestPath: reqPath,
			Method:      http.MethodDelete,
			State:       sdk.RequestStatusQueued,
		},
	}
	s.NoError(s.service.removeLANPendingRequestFromCluster())
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID(), "request should be removed from status")
}

func (s *ServiceTestSuite) Test_Network_RemoveLANPendingRequestFromCluster_NoRequest() {
	s.NoError(s.service.removeLANPendingRequestFromCluster())
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_NoExistingLAN_NoRequest_Create() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.createLANCall().Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID())
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.dataCenterID()]
	s.Equal(reqPath, req.RequestPath, "Request path is different than expected")
	s.Equal(http.MethodPost, req.Method, "Request method is different than expected")
	s.Equal(infrav1.RequestStatusQueued, req.State, "Request state is different than expected")
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_NoExistingLAN_ExistingRequest_NotFailed() {
	testCases := []struct {
		name   string
		status string
	}{
		{
			name:   "request is queued",
			status: sdk.RequestStatusQueued,
		},
		{
			name:   "request is running",
			status: sdk.RequestStatusRunning,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
			s.getRequestsCall().Return(s.examplePostRequest(tc.status), nil).Once()
			requeue, err := s.service.ReconcileLAN()
			s.NoError(err)
			s.True(requeue)
		})
	}
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_NoExistingLAN_ExistingRequest_Failed_Create() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.getRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusFailed), nil).Once()
	s.createLANCall().Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_NoExistingLAN_ExistingRequest_Done_Retry() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.getRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusDone), nil).Once()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_ExistingLAN() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_NoPendingRequests_NoOtherUsers_Delete() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.deleteLANCall(lanID).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID())
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.dataCenterID()]
	s.Equal(reqPath, req.RequestPath, "Request path is different than expected")
	s.Equal(http.MethodDelete, req.Method, "Request method is different than expected")
	s.Equal(infrav1.RequestStatusQueued, req.State, "Request state is different than expected")
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_NoPendingRequests_HasOtherUsers_NoDelete() {
	lan := s.exampleLAN()
	lan.Entities.Nics.Items = &[]sdk.Nic{{Id: ptr.To("1")}}
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID())
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_ExistingRequest_InProgress() {
	testCases := []struct {
		name   string
		status string
	}{
		{
			name:   "request is queued",
			status: sdk.RequestStatusQueued,
		},
		{
			name:   "request is running",
			status: sdk.RequestStatusRunning,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
			requests := s.exampleDeleteRequest(tc.status)
			s.getRequestsCall().Return(requests, nil).Once()
			requeue, err := s.service.ReconcileLANDeletion()
			s.NoError(err)
			s.True(requeue)
		})
	}
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_ExistingRequest_Failed() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return(s.exampleDeleteRequest(sdk.RequestStatusFailed), nil).Once()
	s.deleteLANCall(lanID).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_ExistingRequest_Done() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return(s.exampleDeleteRequest(sdk.RequestStatusDone), nil).Once()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID())
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANDoesNotExist() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_NoLANExists() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.dataCenterID())
}

const (
	reqPath = "this/is/a/path"
	lanID   = "1"
)

func (s *ServiceTestSuite) exampleLAN() sdk.Lan {
	return sdk.Lan{
		Id: ptr.To(lanID),
		Properties: &sdk.LanProperties{
			Name: ptr.To(s.service.lanName()),
		},
		Entities: &sdk.LanEntities{
			Nics: &sdk.LanNics{
				Items: &[]sdk.Nic{},
			},
		},
	}
}

func (s *ServiceTestSuite) examplePostRequest(status string) []sdk.Request {
	body := fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.lanName())
	return []sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(status),
						Message: ptr.To("test"),
					},
				},
			},
			Properties: &sdk.RequestProperties{
				Method: ptr.To(http.MethodPost),
				Body:   ptr.To(body),
			},
		},
	}
}

func (s *ServiceTestSuite) exampleDeleteRequest(status string) []sdk.Request {
	return []sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(status),
						Message: ptr.To("test"),
						Targets: &[]sdk.RequestTarget{
							{
								Target: &sdk.ResourceReference{Id: ptr.To(lanID)},
							},
						},
					},
				},
			},
		},
	}
}

func (s *ServiceTestSuite) createLANCall() *clienttest.MockClient_CreateLAN_Call {
	return s.ionosClient.EXPECT().CreateLAN(s.ctx, s.service.dataCenterID(), sdk.LanPropertiesPost{
		Name:   ptr.To(s.service.lanName()),
		Public: ptr.To(true),
	})
}

func (s *ServiceTestSuite) deleteLANCall(id string) *clienttest.MockClient_DeleteLAN_Call {
	return s.ionosClient.EXPECT().DeleteLAN(s.ctx, s.service.dataCenterID(), id)
}

func (s *ServiceTestSuite) listLANsCall() *clienttest.MockClient_ListLANs_Call {
	return s.ionosClient.EXPECT().ListLANs(s.ctx, s.service.dataCenterID())
}

func (s *ServiceTestSuite) getRequestsCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, mock.Anything,
		path.Join("datacenters", s.service.dataCenterID(), "lans"))
}
