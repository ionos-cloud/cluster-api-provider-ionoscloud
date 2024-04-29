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
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/suite"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type ipBlockTestSuite struct {
	ServiceTestSuite
}

func TestIPBlockTestSuite(t *testing.T) {
	suite.Run(t, new(ipBlockTestSuite))
}

const (
	exampleIPBlockName = "k8s-ipb-default-test-cluster"
)

func (s *ipBlockTestSuite) TestGetControlPlaneEndpointIPBlockMultipleMatches() {
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			*exampleIPBlock(),
			*exampleIPBlock(),
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(exampleIPBlock(), nil).Twice()
	block, err := s.service.getControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.Error(err)
	s.Nil(block)
}

func (s *ipBlockTestSuite) TestGetControlPlaneEndpointIPBlockSingleMatch() {
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			*exampleIPBlock(),
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleIPBlockName),
					Location: ptr.To("es/vit"),
				},
			},
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(exampleIPBlock(), nil).Once()
	block, err := s.service.getControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.NotNil(block)
	s.Equal(exampleIPBlockName, *block.Properties.Name, "IP block name does not match")
	s.Equal(exampleLocation, *block.Properties.Location, "IP block location does not match")
	s.Equal(sdk.Available, *block.Metadata.State, "IP block state does not match")
}

func (s *ipBlockTestSuite) TestGetControlPlaneEndpointIPBlockUserSetIP() {
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	name := ptr.To("random name")
	location := ptr.To(exampleLocation)
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Id: ptr.To(exampleIPBlockID),
				Properties: &sdk.IpBlockProperties{
					Name:     name,
					Location: location,
					Ips: &[]string{
						exampleEndpointIP,
					},
				},
			},
			{
				Properties: &sdk.IpBlockProperties{
					Name:     name,
					Location: location,
				},
			},
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(&sdk.IpBlock{
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Id: ptr.To(exampleIPBlockID),
		Properties: &sdk.IpBlockProperties{
			Ips: &[]string{
				exampleEndpointIP,
			},
		},
	}, nil).Once()
	block, err := s.service.getControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Contains(*block.GetProperties().GetIps(), exampleEndpointIP)
	s.Equal(sdk.Available, *block.Metadata.State, "IP block state does not match")
}

func (s *ipBlockTestSuite) TestGetControlPlaneEndpointIPBlockPreviouslySetID() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(&sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
	}, nil).Once()

	block, err := s.service.getControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Equal(exampleIPBlockID, *block.Id)
}

func (s *ipBlockTestSuite) TestGetControlPlaneEndpointIPBlockPreviouslySetIDNotFound() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(
		nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404),
	).Once()

	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{},
	}, nil).Once()
	block, err := s.service.getControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.ErrorAs(err, &sdk.GenericOpenAPIError{})
	s.Nil(block)
}

func (s *ipBlockTestSuite) TestGetControlPlaneEndpointIPBlockNoMatch() {
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleIPBlockName),
					Location: ptr.To("de/fra"),
				},
			},
		},
	}, nil).Once()
	block, err := s.service.getControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Nil(block)
}

func (s *ipBlockTestSuite) TestReserveControlPlaneEndpointIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	s.mockReserveIPBlockCall(
		s.service.controlPlaneEndpointIPBlockName(s.clusterScope),
		s.clusterScope.Location(),
	).Return(requestPath, nil).Once()

	err := s.service.reserveControlPlaneEndpointIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.validateRequest(http.MethodPost, sdk.RequestStatusQueued, requestPath, req)
}

func (s *ipBlockTestSuite) TestReserveMachineDeploymentIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	labels := s.machineScope.IonosMachine.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[clusterv1.MachineDeploymentNameLabel] = "test-deployment"

	s.mockReserveIPBlockCall(
		s.service.failoverIPBlockName(s.machineScope),
		s.clusterScope.Location(),
	).Return(requestPath, nil).Once()

	err := s.service.reserveMachineDeploymentFailoverIPBlock(s.ctx, s.machineScope)
	s.NoError(err)
	req := s.machineScope.IonosMachine.Status.CurrentRequest
	s.validateRequest(http.MethodPost, sdk.RequestStatusQueued, requestPath, req)
}

func (s *ipBlockTestSuite) validateRequest(method, state, path string, req *infrav1.ProvisioningRequest) {
	s.T().Helper()

	s.NotNil(req)
	s.Equal(method, req.Method)
	s.Equal(state, req.State)
	s.Equal(path, req.RequestPath)
}

func (s *ipBlockTestSuite) TestDeleteControlPlaneEndpointIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	s.mockDeleteIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.deleteControlPlaneEndpointIPBlock(s.ctx, s.clusterScope, exampleIPBlockID)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.NotNil(req)
	s.Equal(http.MethodDelete, req.Method)
	s.Equal(sdk.RequestStatusQueued, req.State)
	s.Equal(requestPath, req.RequestPath)
}

func (s *ipBlockTestSuite) TestGetLatestControlPlaneEndpointIPBlockCreationRequestNoRequest() {
	s.mockGetIPBlocksRequestsPostCall().Return(make([]sdk.Request, 0), nil).Once()
	req, err := s.service.getLatestControlPlaneEndpointIPBlockCreationRequest(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Nil(req)
}

func (s *ipBlockTestSuite) TestGetLatestControlPlaneEndpointIPBlockCreationRequestSuccessfulRequest() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodPost, "")
	reqs := []sdk.Request{req}
	s.mockGetIPBlocksRequestsPostCall().Return(reqs, nil).Once()
	info, err := s.service.getLatestControlPlaneEndpointIPBlockCreationRequest(s.ctx, s.clusterScope)
	s.NoError(err)
	s.NotNil(info)
}

func (s *ipBlockTestSuite) TestGetLatestIPBlockDeletionRequestNoRequest() {
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return(make([]sdk.Request, 0), nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleIPBlockID)
	s.NoError(err)
	s.Nil(info)
}

func (s *ipBlockTestSuite) TestGetLatestIPBlockDeletionRequestRequest() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodDelete, exampleIPBlockID)
	reqs := []sdk.Request{req}
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return(reqs, nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleIPBlockID)
	s.NoError(err)
	s.NotNil(info)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointUserSetIP() {
	block := exampleIPBlock()
	block.Properties.Name = ptr.To("asdf")
	block.Properties.Ips = &[]string{
		"another IP",
		exampleEndpointIP,
	}
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Port = 0
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{*block},
	}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(block, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal(exampleEndpointIP, s.clusterScope.GetControlPlaneEndpoint().Host)
	s.Equal(defaultControlPlaneEndpointPort, s.clusterScope.GetControlPlaneEndpoint().Port)
	s.Equal(exampleIPBlockID, s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointUserSetPort() {
	var port int32 = 9999
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = ""
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Port = port
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(exampleIPBlock(), nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal(exampleEndpointIP, s.clusterScope.GetControlPlaneEndpoint().Host)
	s.Equal(port, s.clusterScope.GetControlPlaneEndpoint().Port)
	s.Equal(exampleIPBlockID, s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointPendingRequest() {
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetIPBlocksRequestsPostCall().Return([]sdk.Request{
		s.buildRequest(sdk.RequestStatusRunning, http.MethodPost, ""),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
	s.Equal(sdk.RequestStatusRunning, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.State)
	s.Equal(http.MethodPost, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.Method)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointRequestNewIPBlock() {
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetIPBlocksRequestsPostCall().Return([]sdk.Request{}, nil).Once()
	s.mockReserveIPBlockCall(
		s.service.controlPlaneEndpointIPBlockName(s.clusterScope),
		s.clusterScope.Location(),
	).Return(exampleRequestPath, nil).Once()

	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionCreationPendingRequest() {
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetIPBlocksRequestsPostCall().Return([]sdk.Request{
		s.buildRequest(sdk.RequestStatusRunning, http.MethodPost, ""),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
	s.Equal(sdk.RequestStatusRunning, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.State)
	s.Equal(http.MethodPost, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.Method)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionUserSetIPWithIPBlockID() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(&sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionUserSetIPWithoutIPBlockID() {
	block := exampleIPBlock()
	block.Properties.Name = ptr.To("aaaa")
	block.Properties.Ips = &[]string{
		"an IP",
		exampleEndpointIP,
	}
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			*block,
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(block, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionNoBlockLeft() {
	s.infraCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	s.mockListIPBlocksCall().Return(nil, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Nil(s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionDeletionPendingRequest() {
	block := exampleIPBlock()
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{
		*block,
	}}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(block, nil).Once()
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return([]sdk.Request{
		s.buildRequest(sdk.RequestStatusRunning, http.MethodDelete, exampleIPBlockID),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(sdk.RequestStatusRunning, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.State)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionRequestNewDeletion() {
	block := exampleIPBlock()
	block.Metadata.State = ptr.To(sdk.Busy)
	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{
		*block,
	}}, nil).Once()
	blockUpdated := exampleIPBlock()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(blockUpdated, nil).Once()
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return(nil, nil).Once()
	s.mockDeleteIPBlockCall().Return(exampleRequestPath, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletion() {
	s.infraMachine.Spec.FailoverIP = infrav1.CloudResourceConfigAuto
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))

	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(ipBlock, nil).Once()
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return(nil, nil).Once()
	s.mockDeleteIPBlockCall().Return(exampleRequestPath, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionSkipped() {
	s.infraMachine.Spec.FailoverIP = infrav1.CloudResourceConfigAuto
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))
	lan := s.exampleLAN()
	lan.Properties.IpFailover = &[]sdk.IPFailover{{
		Ip:      ptr.To(exampleEndpointIP),
		NicUuid: nil,
	}}

	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(ipBlock, nil).Once()
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return(nil, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionPendingCreation() {
	s.infraMachine.Spec.FailoverIP = infrav1.CloudResourceConfigAuto

	s.mockListIPBlocksCall().Return(nil, nil).Once()
	s.mockGetIPBlocksRequestsPostCall().Return([]sdk.Request{
		s.buildIPBlockRequestWithName(
			s.service.failoverIPBlockName(s.machineScope), sdk.RequestStatusQueued, http.MethodPost, ""),
	}, nil,
	).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionPendingDeletion() {
	s.infraMachine.Spec.FailoverIP = infrav1.CloudResourceConfigAuto
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))

	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(ipBlock, nil).Once()

	deleteRequest := s.buildIPBlockRequestWithName(
		s.service.failoverIPBlockName(s.machineScope),
		sdk.RequestStatusQueued,
		http.MethodDelete,
		exampleIPBlockID,
	)

	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return([]sdk.Request{deleteRequest}, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
	s.Equal(
		*deleteRequest.GetMetadata().GetRequestStatus().GetHref(),
		s.machineScope.IonosMachine.Status.CurrentRequest.RequestPath,
	)

	s.Equal(http.MethodDelete, s.machineScope.IonosMachine.Status.CurrentRequest.Method)
	s.Equal(sdk.RequestStatusQueued, s.machineScope.IonosMachine.Status.CurrentRequest.State)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionDeletionFinished() {
	s.infraMachine.Spec.FailoverIP = infrav1.CloudResourceConfigAuto
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))

	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall(exampleIPBlockID).Return(ipBlock, nil).Once()

	deleteRequest := s.buildIPBlockRequestWithName("", sdk.RequestStatusDone, http.MethodDelete, exampleIPBlockID)
	s.mockGetIPBlocksRequestsDeleteCall(exampleIPBlockID).Return([]sdk.Request{deleteRequest}, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest)
}

func (s *ipBlockTestSuite) mockDeleteIPBlockCall() *clienttest.MockClient_DeleteIPBlock_Call {
	return s.ionosClient.EXPECT().DeleteIPBlock(s.ctx, exampleIPBlockID)
}

func (s *ipBlockTestSuite) buildRequest(status string, method, id string) sdk.Request {
	return s.buildIPBlockRequestWithName(s.service.controlPlaneEndpointIPBlockName(s.clusterScope), status, method, id)
}

// exampleIPBlock returns a new sdk.IpBlock instance for testing. The IPs need to be set.
func exampleIPBlock() *sdk.IpBlock {
	return exampleIPBlockWithName(exampleIPBlockName)
}

func exampleIPBlockWithName(name string) *sdk.IpBlock {
	return &sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Properties: &sdk.IpBlockProperties{
			Name:     ptr.To(name),
			Location: ptr.To(exampleLocation),
			Ips: &[]string{
				exampleEndpointIP,
			},
		},
	}
}
