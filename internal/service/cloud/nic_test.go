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
	"net/http"
	"path"
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/suite"

	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
)

const (
	testNICID  = "f3b3f8e4-3b6d-4b6d-8f1d-3e3e6e3e3e3e"
	testDHCPIP = "1.2.3.1"
)

const (
	assertMessageNICIsNil         = "reconcileNICConfig() should return a NIC"
	assertMessageNICErrorOccurred = "reconcileNICConfig() should not return an error"
	assertCurrentRequestIsNil     = "reconcileNICConfig() should not store a pending request"
)

type nicSuite struct {
	ServiceTestSuite
}

func TestNICSuite(t *testing.T) {
	suite.Run(t, new(nicSuite))
}

func (s *nicSuite) TestReconcileNICConfig() {
	s.mockGetServer(testServerID).Return(defaultServer(s.service.serverName(), testDHCPIP), nil).Once()

	// no patch request
	s.mockGetLatestNICPatchRequest(testServerID, testNICID).Return([]sdk.Request{}, nil).Once()
	location := "test/nic/request/path"

	expectedIPs := []string{testDHCPIP, endpointIP}
	s.mockPatchNIC(testServerID, testNICID, sdk.NicProperties{Ips: &expectedIPs}).Return(location, nil).Once()
	// expect request to be successful
	s.mockWaitForRequest(location).Return(nil).Once()

	nic, err := s.service.reconcileNICConfig(endpointIP)

	s.NotNil(nic, assertMessageNICIsNil)
	s.NoError(err, assertMessageNICErrorOccurred)
	s.Nil(s.service.scope.IonosMachine.Status.CurrentRequest, assertCurrentRequestIsNil)
}

func (s *nicSuite) TestReconcileNICConfigIPIsSet() {
	s.mockGetServer(testServerID).Return(defaultServer(s.service.serverName(), testDHCPIP, endpointIP), nil).Once()
	s.mockGetLatestNICPatchRequest(testServerID, testNICID).Return([]sdk.Request{}, nil).Once()
	nic, err := s.service.reconcileNICConfig(endpointIP)

	s.NotNil(nic, assertMessageNICIsNil)
	s.NoError(err, assertMessageNICErrorOccurred)
	s.Nil(s.service.scope.IonosMachine.Status.CurrentRequest, assertCurrentRequestIsNil)
}

func (s *nicSuite) TestReconcileNICConfigPatchRequestPending() {
	s.mockGetServer(testServerID).Return(defaultServer(s.service.serverName(), testDHCPIP), nil).Once()

	patchRequest := s.examplePatchRequest(sdk.RequestStatusQueued, testServerID, testNICID)

	s.mockGetLatestNICPatchRequest(testServerID, testNICID).Return(
		[]sdk.Request{patchRequest},
		nil).Once()

	// expect request to be successful
	s.mockWaitForRequest(*patchRequest.Metadata.RequestStatus.Href).Return(nil).Once()

	nic, err := s.service.reconcileNICConfig(endpointIP)
	s.NotNil(nic, assertMessageNICIsNil)
	s.NoError(err, assertMessageNICErrorOccurred)
	s.Nil(s.service.scope.IonosMachine.Status.CurrentRequest, assertCurrentRequestIsNil)
}

func (s *nicSuite) mockGetServer(serverID string) *clienttest.MockClient_GetServer_Call {
	return s.ionosClient.EXPECT().GetServer(s.ctx, s.service.datacenterID(), serverID)
}

func (s *nicSuite) mockPatchNIC(serverID, nicID string, props sdk.NicProperties) *clienttest.MockClient_PatchNIC_Call {
	return s.ionosClient.EXPECT().PatchNIC(s.ctx, s.service.datacenterID(), serverID, nicID, props)
}

func (s *nicSuite) mockGetLatestNICPatchRequest(serverID, nicID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPatch, s.service.nicURL(serverID, nicID))
}

func (s *nicSuite) mockWaitForRequest(location string) *clienttest.MockClient_WaitForRequest_Call {
	return s.ionosClient.EXPECT().WaitForRequest(s.ctx, location)
}

func (s *nicSuite) examplePatchRequest(status, serverID, nicID string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPatch,
		url:        s.service.nicURL(serverID, nicID),
		href:       path.Join(reqPath, nicID),
		targetID:   nicID,
		targetType: sdk.NIC,
	}
	return s.exampleRequest(opts)
}
