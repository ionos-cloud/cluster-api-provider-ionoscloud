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
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

const (
	testNICID  = "f3b3f8e4-3b6d-4b6d-8f1d-3e3e6e3e3e3e"
	testDHCPIP = "198.51.100.1"
)

type nicSuite struct {
	ServiceTestSuite
}

func TestNICSuite(t *testing.T) {
	suite.Run(t, new(nicSuite))
}

func (n *nicSuite) TestReconcileNICConfig() {
	n.mockGetServer(testServerID).Return(n.defaultServer(testDHCPIP), nil).Once()

	// no patch request
	n.mockGetLatestNICPatchRequest(testServerID, testNICID).Return([]sdk.Request{}, nil).Once()
	location := "test/nic/request/path"

	expectedIPs := []string{testDHCPIP, endpointIP}
	n.mockPatchNIC(testServerID, testNICID, sdk.NicProperties{Ips: &expectedIPs}).Return(location, nil).Once()
	// expect request to be successful
	n.mockWaitForRequest(location).Return(nil).Once()

	nic, err := n.service.reconcileNICConfig(endpointIP)

	n.NotNil(nic, "reconcileNICConfig() should return a NIC")
	n.NoError(err, "reconcileNICConfig() should not return an error")
	n.Nil(n.service.scope.IonosMachine.Status.CurrentRequest, "currentRequest should be nil")
}

func (n *nicSuite) TestReconcileNICConfigIPIsSet() {
	n.mockGetServer(testServerID).Return(n.defaultServer(testDHCPIP, endpointIP), nil).Once()
	n.mockGetLatestNICPatchRequest(testServerID, testNICID).Return([]sdk.Request{}, nil).Once()
	nic, err := n.service.reconcileNICConfig(endpointIP)

	n.NotNil(nic, "reconcileNICConfig() should return a NIC")
	n.NoError(err, "reconcileNICConfig() should not return an error")
	n.Nil(n.service.scope.IonosMachine.Status.CurrentRequest, "currentRequest should be nil")
}

func (n *nicSuite) TestReconcileNICConfigPatchRequestPending() {
	n.mockGetServer(testServerID).Return(n.defaultServer(testDHCPIP), nil).Once()

	patchRequest := n.examplePatchRequest(sdk.RequestStatusQueued, testServerID, testNICID)

	n.mockGetLatestNICPatchRequest(testServerID, testNICID).Return(
		[]sdk.Request{patchRequest},
		nil).Once()

	// expect request to be successful
	n.mockWaitForRequest(*patchRequest.Metadata.RequestStatus.Href).Return(nil).Once()

	nic, err := n.service.reconcileNICConfig(endpointIP)
	n.NotNil(nic, "reconcileNICConfig() should return a NIC")
	n.NoError(err, "reconcileNICConfig() should not return an error")
	n.Nil(n.service.scope.IonosMachine.Status.CurrentRequest, "currentRequest should be nil")
}

func (n *nicSuite) mockGetServer(serverID string) *clienttest.MockClient_GetServer_Call {
	return n.ionosClient.EXPECT().GetServer(n.ctx, n.service.datacenterID(), serverID)
}

func (n *nicSuite) mockPatchNIC(serverID, nicID string, props sdk.NicProperties) *clienttest.MockClient_PatchNIC_Call {
	return n.ionosClient.EXPECT().PatchNIC(n.ctx, n.service.datacenterID(), serverID, nicID, props)
}

func (n *nicSuite) mockGetLatestNICPatchRequest(serverID, nicID string) *clienttest.MockClient_GetRequests_Call {
	return n.ionosClient.EXPECT().GetRequests(n.ctx, http.MethodPatch, n.service.nicURL(serverID, nicID))
}

func (n *nicSuite) mockWaitForRequest(location string) *clienttest.MockClient_WaitForRequest_Call {
	return n.ionosClient.EXPECT().WaitForRequest(n.ctx, location)
}

func (n *nicSuite) examplePatchRequest(status, serverID, nicID string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodDelete,
		url:        n.service.nicURL(serverID, nicID),
		href:       path.Join(reqPath, nicID),
		targetID:   testNICID,
		targetType: sdk.NIC,
	}
	return n.exampleRequest(opts)
}

func (n *nicSuite) defaultServer(ips ...string) *sdk.Server {
	return &sdk.Server{
		Id: ptr.To(testServerID),
		Entities: &sdk.ServerEntities{
			Nics: &sdk.Nics{
				Items: &[]sdk.Nic{{
					Entities: nil,
					Id:       ptr.To(testNICID),
					Properties: &sdk.NicProperties{
						Dhcp: ptr.To(true),
						Name: ptr.To(n.service.serverName()),
						Ips:  &ips,
					},
				}},
			},
		},
	}
}
