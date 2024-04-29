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

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
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
	s.mockGetServerCall(exampleServerID).Return(s.defaultServer(s.infraMachine, exampleDHCPIP), nil).Once()

	// no patch request
	s.mockGetLatestNICPatchRequest(exampleServerID, exampleNICID).Return([]sdk.Request{}, nil).Once()
	location := "test/nic/request/path"

	expectedIPs := []string{exampleDHCPIP, exampleEndpointIP}
	s.mockPatchNIC(exampleServerID, exampleNICID, sdk.NicProperties{Ips: &expectedIPs}).Return(location, nil).Once()
	// expect request to be successful
	s.mockWaitForRequestCall(location).Return(nil).Once()

	nic, err := s.service.reconcileNICConfig(s.ctx, s.machineScope, exampleEndpointIP)

	s.NotNil(nic, assertMessageNICIsNil)
	s.NoError(err, assertMessageNICErrorOccurred)
	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest, assertCurrentRequestIsNil)
}

func (s *nicSuite) TestReconcileNICConfigIPIsSet() {
	s.mockGetServerCall(exampleServerID).
		Return(s.defaultServer(s.infraMachine, exampleDHCPIP, exampleEndpointIP), nil).Once()
	nic, err := s.service.reconcileNICConfig(s.ctx, s.machineScope, exampleEndpointIP)

	s.NotNil(nic, assertMessageNICIsNil)
	s.NoError(err, assertMessageNICErrorOccurred)
	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest, assertCurrentRequestIsNil)
}

func (s *nicSuite) TestReconcileNICConfigPatchRequestPending() {
	s.mockGetServerCall(exampleServerID).Return(s.defaultServer(s.infraMachine, exampleDHCPIP), nil).Once()

	patchRequest := s.examplePatchRequest(sdk.RequestStatusQueued, exampleServerID, exampleNICID)

	s.mockGetLatestNICPatchRequest(exampleServerID, exampleNICID).Return(
		[]sdk.Request{patchRequest},
		nil).Once()

	// expect request to be successful
	s.mockWaitForRequestCall(*patchRequest.Metadata.RequestStatus.Href).Return(nil).Once()

	nic, err := s.service.reconcileNICConfig(s.ctx, s.machineScope, exampleEndpointIP)
	s.NotNil(nic, assertMessageNICIsNil)
	s.NoError(err, assertMessageNICErrorOccurred)
	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest, assertCurrentRequestIsNil)
}

func (s *nicSuite) mockPatchNIC(serverID, nicID string, props sdk.NicProperties) *clienttest.MockClient_PatchNIC_Call {
	return s.ionosClient.EXPECT().PatchNIC(s.ctx, s.machineScope.DatacenterID(), serverID, nicID, props)
}

func (s *nicSuite) mockGetLatestNICPatchRequest(serverID, nicID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().
		GetRequests(s.ctx, http.MethodPatch, s.service.nicURL(s.machineScope, serverID, nicID))
}

func (s *nicSuite) examplePatchRequest(status, serverID, nicID string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPatch,
		url:        s.service.nicURL(s.machineScope, serverID, nicID),
		href:       path.Join(exampleRequestPath, nicID),
		targetID:   nicID,
		targetType: sdk.NIC,
	}
	return s.exampleRequest(opts)
}
