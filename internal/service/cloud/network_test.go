package cloud

import (
	"fmt"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/mock"
	"k8s.io/utils/ptr"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
)

func (s *ServiceTestSuite) TestNetworkLANName() {
	s.Equal("k8s-default-test-cluster", s.service.lanName())
}

func (s *ServiceTestSuite) Test_Network_CreateLAN_Successful() {
	s.createLANCall().Return(reqPath, nil).Once()
	s.NoError(s.service.createLAN())
	s.Contains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequest[s.service.dataCenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodPost, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *ServiceTestSuite) Test_Network_CreateLAN_Error_API() {
	s.createLANCall().Return("", errMock).Once()
	err := s.service.createLAN()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.NotContains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID(), "request should not be stored in status")
}

func (s *ServiceTestSuite) Test_Network_DeleteLAN_Successful() {
	s.deleteLANCall(lanID).Return(reqPath, nil).Once()
	s.NoError(s.service.deleteLAN(lanID))
	s.Contains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequest[s.service.dataCenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodDelete, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *ServiceTestSuite) Test_Network_DeleteLAN_Error_API() {
	s.deleteLANCall(lanID).Return("", errMock).Once()
	err := s.service.deleteLAN(lanID)
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.NotContains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID(), "request should not be stored in status")
}

func (s *ServiceTestSuite) Test_Network_GetLAN_Successful() {
	lan := s.exampleLAN()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	foundLAN, err := s.service.GetLAN()
	s.NoError(err)
	s.NotNil(foundLAN)
	s.Equal(lan, *foundLAN)
}

func (s *ServiceTestSuite) Test_Network_GetLAN_Error_API() {
	s.listLANsCall().Return(nil, errMock).Once()
	lan, err := s.service.GetLAN()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.Nil(lan)
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

func (s *ServiceTestSuite) Test_Network_CheckForPendingLANRequest_Error_API() {
	s.getRequestsCall().Return(nil, errMock).Once()
	request, err := s.service.checkForPendingLANRequest(http.MethodDelete, lanID)
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.Empty(request)
}

func (s *ServiceTestSuite) Test_Network_RemoveLANPendingRequestFromCluster_Successful() {
	s.infraCluster.Status.CurrentRequest = map[string]infrav1.ProvisioningRequest{
		s.service.dataCenterID(): {
			RequestPath: reqPath,
			Method:      http.MethodDelete,
			State:       sdk.RequestStatusQueued,
		},
	}
	s.NoError(s.service.removeLANPendingRequestFromCluster())
	s.NotContains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID(), "request should be removed from status")
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
	s.Contains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID())
	req := s.infraCluster.Status.CurrentRequest[s.service.dataCenterID()]
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

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_Error_API_ListingLANs() {
	s.listLANsCall().Return(nil, errMock).Once()
	requeue, err := s.service.ReconcileLAN()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_Error_API_GetRequests() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.getRequestsCall().Return(nil, errMock).Once()
	requeue, err := s.service.ReconcileLAN()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_Error_API_CreateLAN() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.createLANCall().Return("", errMock).Once()
	requeue, err := s.service.ReconcileLAN()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLAN_Error_API_ListingLAN_RequestSucceededMeanwhile() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.getRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusDone), nil).Once()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, errMock).Once()
	requeue, err := s.service.ReconcileLAN()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_NoPendingRequests_NoOtherUsers_Delete() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.deleteLANCall(lanID).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
	s.Contains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID())
	req := s.infraCluster.Status.CurrentRequest[s.service.dataCenterID()]
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
	s.NotContains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID())
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
			requests := []sdk.Request{
				{
					Id: ptr.To("1"),
					Metadata: &sdk.RequestMetadata{
						RequestStatus: &sdk.RequestStatus{
							Metadata: &sdk.RequestStatusMetadata{
								Status:  ptr.To(tc.status),
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
			s.getRequestsCall().Return(requests, nil).Once()
			requeue, err := s.service.ReconcileLANDeletion()
			s.NoError(err)
			s.True(requeue)
		})
	}
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_ExistingRequest_Failed() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(sdk.RequestStatusFailed),
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
	}, nil).Once()
	s.deleteLANCall(lanID).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANExists_ExistingRequest_Done() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(sdk.RequestStatusDone),
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
	}, nil).Once()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID())
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_LANDoesNotExist() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_Error_API_ListingLANs() {
	s.listLANsCall().Return(nil, errMock).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_Error_API_GetRequests() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return(nil, errMock).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_Error_API_DeleteLAN() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.deleteLANCall(lanID).Return("", errMock).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_Error_API_ListingLAN_RequestSucceededMeanwhile() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.getRequestsCall().Return([]sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(sdk.RequestStatusDone),
						Message: ptr.To("test"),
						Targets: &[]sdk.RequestTarget{
							{Target: &sdk.ResourceReference{Id: ptr.To(lanID)}},
						},
					},
				},
			},
		},
	}, nil).Once()
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, errMock).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.Error(err)
	s.ErrorIs(err, errMock, "different error returned")
	s.False(requeue)
}

func (s *ServiceTestSuite) Test_Network_ReconcileLANDelete_NoLANExists() {
	s.listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequest, s.service.dataCenterID())
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
