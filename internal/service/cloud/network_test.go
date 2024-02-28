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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

const (
	reqPath        = "this/is/a/path"
	lanID          = "1"
	arbitraryNICID = "f3b3f8e3-1f3d-11ec-82a8-0242ac130003"
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

func (s *lanSuite) TestNetworkCreateLANSuccessful() {
	s.mockCreateLANCall().Return(reqPath, nil).Once()
	s.NoError(s.service.createLAN())
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.datacenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodPost, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *lanSuite) TestNetworkDeleteLANSuccessful() {
	s.mockDeleteLANCall(lanID).Return(reqPath, nil).Once()
	s.NoError(s.service.deleteLAN(lanID))
	s.Contains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID(), "request should be stored in status")
	req := s.infraCluster.Status.CurrentRequestByDatacenter[s.service.datacenterID()]
	s.Equal(reqPath, req.RequestPath, "request path should be stored in status")
	s.Equal(http.MethodDelete, req.Method, "request method should be stored in status")
	s.Equal(infrav1.RequestStatusQueued, req.State, "request status should be stored in status")
}

func (s *lanSuite) TestNetworkGetLANSuccessful() {
	lan := s.exampleLAN()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	foundLAN, err := s.service.getLAN()
	s.NoError(err)
	s.NotNil(foundLAN)
	s.Equal(lan, *foundLAN)
}

func (s *lanSuite) TestNetworkGetLANNotFound() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	lan, err := s.service.getLAN()
	s.NoError(err)
	s.Nil(lan)
}

func (s *lanSuite) TestNetworkGetLANErrorNotUnique() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN(), s.exampleLAN()}}, nil).Once()
	lan, err := s.service.getLAN()
	s.Error(err)
	s.Nil(lan)
}

func (s *lanSuite) TestNetworkRemoveLANPendingRequestFromClusterSuccessful() {
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

func (s *lanSuite) TestNetworkRemoveLANPendingRequestFromClusterNoRequest() {
	s.NoError(s.service.removeLANPendingRequestFromCluster())
}

func (s *lanSuite) TestNetworkReconcileLANNoExistingLANNoRequestCreate() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.mockCreateLANCall().Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANNoExistingLANExistingRequestPending() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusQueued), nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANExistingLAN() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANExistingLANUnavailable() {
	lan := s.exampleLAN()
	lan.Metadata.State = ptr.To("BUSY")
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	requeue, err := s.service.ReconcileLAN()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANDeleteLANExistsNoPendingRequestsNoOtherUsersDelete() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.mockGetLANDeletionRequestsCall().Return([]sdk.Request{}, nil).Once()
	s.mockDeleteLANCall(lanID).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANDeleteLANExistsNoPendingRequestsHasOtherUsersNoDelete() {
	lan := s.exampleLAN()
	lan.Entities.Nics.Items = &[]sdk.Nic{{Id: ptr.To("1")}}
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
	s.mockGetLANDeletionRequestsCall().Return([]sdk.Request{}, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID())
}

func (s *lanSuite) TestNetworkReconcileLANDeleteNoExistingLANExistingRequestPending() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return(s.examplePostRequest(sdk.RequestStatusQueued), nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANDeleteLANExistsExistingRequestPending() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	requests := s.exampleDeleteRequest(sdk.RequestStatusQueued)
	s.mockGetLANDeletionRequestsCall().Return(requests, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.True(requeue)
}

func (s *lanSuite) TestNetworkReconcileLANDeleteLANDoesNotExist() {
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
	s.mockGetLANCreationRequestsCall().Return(nil, nil).Once()
	requeue, err := s.service.ReconcileLANDeletion()
	s.NoError(err)
	s.False(requeue)
	s.NotContains(s.infraCluster.Status.CurrentRequestByDatacenter, s.service.datacenterID())
}

func (s *lanSuite) TestReconcileIPFailoverNICNotInFailoverGroup() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	testServer := defaultServer(s.service.serverName(), testDHCPIP, testEndpointIP)

	s.mockGetServerCall(testServerID).Return(testServer, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
	s.mockGetLANPatchRequestCall().Return([]sdk.Request{s.examplePatchRequest(sdk.RequestStatusDone)}, nil).Once()

	props := sdk.LanProperties{
		IpFailover: &[]sdk.IPFailover{{
			Ip:      ptr.To(testEndpointIP),
			NicUuid: ptr.To(testNICID),
		}},
	}

	s.mockPatchLANCall(props).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileIPFailover()
	s.checkSuccessfulFailoverGroupPatch(expectedPatchResult{
		err:           err,
		requeue:       requeue,
		requestPath:   reqPath,
		requestMethod: http.MethodPatch,
		requestState:  infrav1.RequestStatusQueued,
	})
}

func (s *lanSuite) TestReconcileIPFailoverNICAlreadyInFailoverGroup() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	testServer := defaultServer(s.service.serverName(), testDHCPIP, testEndpointIP)
	testLAN := s.exampleLAN()
	testLAN.Properties.IpFailover = &[]sdk.IPFailover{{
		Ip:      ptr.To(testEndpointIP),
		NicUuid: ptr.To(testNICID),
	}}

	s.mockGetServerCall(testServerID).Return(testServer, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{testLAN}}, nil).Once()
	s.mockGetLANPatchRequestCall().Return([]sdk.Request{s.examplePatchRequest(sdk.RequestStatusDone)}, nil).Once()

	requeue, err := s.service.ReconcileIPFailover()

	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) TestReconcileIPFailoverNICHasWrongIPInFailoverGroup() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	testServer := defaultServer(s.service.serverName(), testDHCPIP, testEndpointIP)
	testLAN := s.exampleLAN()

	testLAN.Properties.IpFailover = &[]sdk.IPFailover{{
		// we expect that the first entry will also be included in the Patch request.
		Ip:      ptr.To(testArbitraryIP),
		NicUuid: ptr.To(arbitraryNICID),
	}, {
		// LAN contains the NIC but has an incorrect IP
		Ip:      ptr.To(testUnexpectedIP),
		NicUuid: ptr.To(testNICID),
	}}

	s.mockGetServerCall(testServerID).Return(testServer, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{testLAN}}, nil).Once()
	s.mockGetLANPatchRequestCall().Return([]sdk.Request{s.examplePatchRequest(sdk.RequestStatusDone)}, nil).Once()

	// expect a patch request to update the IP
	props := sdk.LanProperties{
		IpFailover: &[]sdk.IPFailover{{
			Ip:      ptr.To(testArbitraryIP),
			NicUuid: ptr.To(arbitraryNICID),
		}, {
			Ip:      ptr.To(testEndpointIP),
			NicUuid: ptr.To(testNICID),
		}},
	}

	s.mockPatchLANCall(props).Return(reqPath, nil).Once()
	requeue, err := s.service.ReconcileIPFailover()

	s.checkSuccessfulFailoverGroupPatch(expectedPatchResult{
		err:           err,
		requeue:       requeue,
		requestPath:   reqPath,
		requestMethod: http.MethodPatch,
		requestState:  infrav1.RequestStatusQueued,
	})
}

type expectedPatchResult struct {
	err           error
	requeue       bool
	requestPath   string
	requestMethod string
	requestState  infrav1.RequestStatus
}

func (s *lanSuite) checkSuccessfulFailoverGroupPatch(result expectedPatchResult) {
	s.NoError(result.err)
	s.True(result.requeue)
	s.Equal(result.requestPath, s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath)
	s.Equal(result.requestMethod, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal(result.requestState, s.machineScope.IonosMachine.Status.CurrentRequest.State)
}

func (s *lanSuite) TestReconcileIPFailoverAnotherNICInFailoverGroup() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	testServer := defaultServer(s.service.serverName(), testDHCPIP, testEndpointIP)
	testLAN := s.exampleLAN()

	testLAN.Properties.IpFailover = &[]sdk.IPFailover{{
		Ip:      ptr.To(testArbitraryIP),
		NicUuid: ptr.To(arbitraryNICID),
	}, {
		Ip: ptr.To(testEndpointIP),
		// arbitrary NIC ID with the correct endpoint IP
		NicUuid: ptr.To("f3b3f8e4-1f3d-11ec-82a8-0242ac130003"),
	}}

	s.mockGetServerCall(testServerID).Return(testServer, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{testLAN}}, nil).Once()
	s.mockGetLANPatchRequestCall().Return([]sdk.Request{s.examplePatchRequest(sdk.RequestStatusDone)}, nil).Once()

	requeue, err := s.service.ReconcileIPFailover()

	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) TestReconcileIPFailoverOnWorkerNode() {
	// machine does not have a control plane label
	requeue, err := s.service.ReconcileIPFailover()

	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) TestReconcileIPFailoverDeletion() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	testServer := defaultServer(s.service.serverName(), testDHCPIP, testEndpointIP)
	testLAN := s.exampleLAN()

	testLAN.Properties.IpFailover = &[]sdk.IPFailover{{
		Ip:      ptr.To(testArbitraryIP),
		NicUuid: ptr.To(arbitraryNICID),
	}, {
		Ip:      ptr.To(testEndpointIP),
		NicUuid: ptr.To(testNICID),
	}}

	s.mockGetServerCall(testServerID).Return(testServer, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{testLAN}}, nil).Once()
	s.mockGetLANPatchRequestCall().Return([]sdk.Request{s.examplePatchRequest(sdk.RequestStatusDone)}, nil).Once()

	props := sdk.LanProperties{
		IpFailover: &[]sdk.IPFailover{{
			Ip:      ptr.To(testArbitraryIP),
			NicUuid: ptr.To(arbitraryNICID),
		}},
	}

	s.mockPatchLANCall(props).Return(reqPath, nil).Once()

	requeue, err := s.service.ReconcileIPFailoverDeletion()

	s.NoError(err)
	s.True(requeue)
	s.Equal(reqPath, s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath)
	s.Equal(http.MethodPatch, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal(infrav1.RequestStatusQueued, s.machineScope.IonosMachine.Status.CurrentRequest.State)
}

func (s *lanSuite) TestReconcileIPFailoverDeletionServerNotFound() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	s.mockGetServerCall(testServerID).Return(nil, sdk.NewGenericOpenAPIError("server not found", nil, nil, 404)).Once()
	s.mockListServerCall().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()

	requeue, err := s.service.ReconcileIPFailoverDeletion()

	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) TestReconcileIPFailoverDeletionPrimaryNICNotFound() {
	s.machineScope.Machine.SetLabels(map[string]string{clusterv1.MachineControlPlaneLabel: "true"})
	s.machineScope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = testEndpointIP

	testServer := &sdk.Server{
		Id: ptr.To(testServerID),
	}

	s.mockGetServerCall(testServerID).Return(testServer, nil).Once()

	requeue, err := s.service.ReconcileIPFailoverDeletion()

	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) TestReconcileIPFailoverDeletionOnWorker() {
	requeue, err := s.service.ReconcileIPFailoverDeletion()

	s.NoError(err)
	s.False(requeue)
}

func (s *lanSuite) exampleLAN() sdk.Lan {
	return sdk.Lan{
		Id: ptr.To(lanID),
		Properties: &sdk.LanProperties{
			Name: ptr.To(s.service.lanName()),
		},
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
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

func (s *lanSuite) examplePatchRequest(status string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPatch,
		url:        s.service.lanURL(lanID),
		href:       reqPath,
		targetID:   lanID,
		targetType: sdk.LAN,
	}
	return s.exampleRequest(opts)
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

func (s *lanSuite) mockPatchLANCall(props sdk.LanProperties) *clienttest.MockClient_PatchLAN_Call {
	return s.ionosClient.EXPECT().PatchLAN(s.ctx, s.service.datacenterID(), lanID, props)
}

func (s *lanSuite) mockGetLANCreationRequestsCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, s.service.lansURL())
}

func (s *lanSuite) mockGetLANPatchRequestCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPatch, s.service.lanURL(lanID))
}

func (s *lanSuite) mockGetLANDeletionRequestsCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodDelete, s.service.lanURL(lanID))
}

func (s *lanSuite) mockGetServerCall(serverID string) *clienttest.MockClient_GetServer_Call {
	return s.ionosClient.EXPECT().GetServer(s.ctx, s.service.datacenterID(), serverID)
}

func (s *lanSuite) mockListServerCall() *clienttest.MockClient_ListServers_Call {
	return s.ionosClient.EXPECT().ListServers(s.ctx, s.service.datacenterID())
}
