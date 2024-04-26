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
	"github.com/stretchr/testify/suite"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type ipBlockTestSuite struct {
	ServiceTestSuite
}

func TestEndpointTestSuite(t *testing.T) {
	suite.Run(t, new(ipBlockTestSuite))
}

const (
	exampleIPBlockName = "k8s-ipb-default-test-cluster"
)

func (s *ipBlockTestSuite) TestGetIPBlockFuncMultipleMatches() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			*exampleIPBlock(),
			*exampleIPBlock(),
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(exampleIPBlock(), nil).Twice()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.Error(err)
	s.Nil(block)
}

func (s *ipBlockTestSuite) TestGetIPBlockFuncSingleMatch() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
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
	s.mockGetIPBlockByIDCall().Return(exampleIPBlock(), nil).Once()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.NotNil(block)
	s.Equal(exampleIPBlockName, *block.Properties.Name, "IP block name does not match")
	s.Equal(exampleLocation, *block.Properties.Location, "IP block location does not match")
	s.Equal(sdk.Available, *block.Metadata.State, "IP block state does not match")
}

func (s *ipBlockTestSuite) TestGetIPBlockFuncUserSetIP() {
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	name := ptr.To("random name")
	location := ptr.To(exampleLocation)
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
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
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
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
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Contains(*block.GetProperties().GetIps(), exampleEndpointIP)
	s.Equal(sdk.Available, *block.Metadata.State, "IP block state does not match")
}

func (s *ipBlockTestSuite) TestGetIPBlockPreviouslySetID() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
	}, nil).Once()

	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Equal(exampleIPBlockID, *block.Id)
}

func (s *ipBlockTestSuite) TestGetIPBlockPreviouslySetIDNotFound() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall().Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404)).Once()
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{},
	}, nil).Once()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.ErrorAs(err, &sdk.GenericOpenAPIError{})
	s.Nil(block)
}

func (s *ipBlockTestSuite) TestGetIPBlockNoMatch() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleIPBlockName),
					Location: ptr.To("de/fra"),
				},
			},
		},
	}, nil).Once()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Nil(block)
}

func (s *ipBlockTestSuite) TestReserveIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	s.mockReserveIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.reserveClusterIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.NotNil(req)
	s.Equal(http.MethodPost, req.Method)
	s.Equal(sdk.RequestStatusQueued, req.State)
	s.Equal(requestPath, req.RequestPath)
}

func (s *ipBlockTestSuite) TestDeleteIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	s.mockDeleteIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.deleteClusterIPBlock(s.ctx, s.clusterScope, exampleIPBlockID)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.NotNil(req)
	s.Equal(http.MethodDelete, req.Method)
	s.Equal(sdk.RequestStatusQueued, req.State)
	s.Equal(requestPath, req.RequestPath)
}

func (s *ipBlockTestSuite) TestGetLatestIPBlockCreationRequestNoRequest() {
	s.mockGetRequestsCallPost().Return(make([]sdk.Request, 0), nil).Once()
	req, err := s.service.getLatestIPBlockCreationRequest(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Nil(req)
}

func (s *ipBlockTestSuite) TestGetLatestIPBlockCreationRequestRequest() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodPost, "")
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallPost().Return(reqs, nil).Once()
	info, err := s.service.getLatestIPBlockCreationRequest(s.ctx, s.clusterScope)
	s.NoError(err)
	s.NotNil(info)
}

func (s *ipBlockTestSuite) TestGetLatestIPBlockDeletionRequestNoRequest() {
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(make([]sdk.Request, 0), nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleIPBlockID)
	s.NoError(err)
	s.Nil(info)
}

func (s *ipBlockTestSuite) TestGetLatestIPBlockDeletionRequestRequest() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodDelete, exampleIPBlockID)
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(reqs, nil).Once()
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
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{*block},
	}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(block, nil).Once()
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
	s.mockGetIPBlockByIDCall().Return(exampleIPBlock(), nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal(exampleEndpointIP, s.clusterScope.GetControlPlaneEndpoint().Host)
	s.Equal(port, s.clusterScope.GetControlPlaneEndpoint().Port)
	s.Equal(exampleIPBlockID, s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointPendingRequest() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{
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
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{}, nil).Once()
	s.mockReserveIPBlockCall().Return(exampleRequestPath, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionCreationPendingRequest() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{
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
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
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
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			*block,
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(block, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionNoBlockLeft() {
	s.infraCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	s.mockListIPBlockCall().Return(nil, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Nil(s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
}

func (s *ipBlockTestSuite) TestReconcileControlPlaneEndpointDeletionDeletionPendingRequest() {
	block := exampleIPBlock()
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{
		*block,
	}}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(block, nil).Once()
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return([]sdk.Request{
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
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{
		*block,
	}}, nil).Once()
	blockUpdated := exampleIPBlock()
	s.mockGetIPBlockByIDCall().Return(blockUpdated, nil).Once()
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(nil, nil).Once()
	s.mockDeleteIPBlockCall().Return(exampleRequestPath, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletion() {
	s.infraMachine.Spec.MachineFailoverIP = ptr.To(infrav1.CloudResourceConfigAuto)
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))

	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(ipBlock, nil).Once()
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(nil, nil).Once()
	s.mockDeleteIPBlockCall().Return(exampleRequestPath, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionSkipped() {
	s.infraMachine.Spec.MachineFailoverIP = ptr.To(infrav1.CloudResourceConfigAuto)
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))
	lan := s.exampleLAN()
	lan.Properties.IpFailover = &[]sdk.IPFailover{{
		Ip:      ptr.To(exampleEndpointIP),
		NicUuid: nil,
	}}

	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(ipBlock, nil).Once()
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(nil, nil).Once()
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionPendingCreation() {
	s.infraMachine.Spec.MachineFailoverIP = ptr.To(infrav1.CloudResourceConfigAuto)

	s.mockListIPBlockCall().Return(nil, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{
		s.buildRequestWithName(
			s.service.failoverIPBlockName(s.machineScope), sdk.RequestStatusQueued, http.MethodPost, ""),
	}, nil,
	).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *ipBlockTestSuite) TestReconcileFailoverIPBlockDeletionPendingDeletion() {
	s.infraMachine.Spec.MachineFailoverIP = ptr.To(infrav1.CloudResourceConfigAuto)
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))

	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(ipBlock, nil).Once()

	deleteRequest := s.buildRequestWithName(
		s.service.failoverIPBlockName(s.machineScope),
		sdk.RequestStatusQueued,
		http.MethodDelete,
		exampleIPBlockID,
	)

	s.mockGetRequestsCallDelete(exampleIPBlockID).Return([]sdk.Request{deleteRequest}, nil).Once()

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

func (s *lanSuite) TestReconcileFailoverIPBlockDeletionDeletionFinished() {
	s.infraMachine.Spec.MachineFailoverIP = ptr.To(infrav1.CloudResourceConfigAuto)
	ipBlock := exampleIPBlockWithName(s.service.failoverIPBlockName(s.machineScope))

	s.mockListIPBlocksCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{*ipBlock}}, nil).Once()
	s.mockGetIPBlockCall(exampleIPBlockID).Return(ipBlock, nil).Once()

	deleteRequest := s.exampleIPBlockDeleteRequest(sdk.RequestStatusDone)
	s.mockDeleteIPBlockRequestCall(exampleIPBlockID).Return(deleteRequest, nil).Once()

	requeue, err := s.service.ReconcileFailoverIPBlockDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.machineScope.IonosMachine.Status.CurrentRequest)
}

func (s *ipBlockTestSuite) mockReserveIPBlockCall() *clienttest.MockClient_ReserveIPBlock_Call {
	return s.ionosClient.
		EXPECT().
		ReserveIPBlock(s.ctx, s.service.ipBlockName(s.clusterScope), s.clusterScope.Location(), int32(1))
}

func (s *ipBlockTestSuite) mockGetIPBlockByIDCall() *clienttest.MockClient_GetIPBlock_Call {
	return s.ionosClient.EXPECT().GetIPBlock(s.ctx, exampleIPBlockID)
}

func (s *ipBlockTestSuite) mockListIPBlockCall() *clienttest.MockClient_ListIPBlocks_Call {
	return s.ionosClient.EXPECT().ListIPBlocks(s.ctx)
}

func (s *ipBlockTestSuite) mockDeleteIPBlockCall() *clienttest.MockClient_DeleteIPBlock_Call {
	return s.ionosClient.EXPECT().DeleteIPBlock(s.ctx, exampleIPBlockID)
}

func (s *ipBlockTestSuite) mockGetRequestsCallPost() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, ipBlocksPath)
}

func (s *ipBlockTestSuite) mockGetRequestsCallDelete(id string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodDelete, path.Join(ipBlocksPath, id))
}

func (s *ipBlockTestSuite) buildRequest(status string, method, id string) sdk.Request {
	return s.buildRequestWithName(s.service.ipBlockName(s.clusterScope), status, method, id)
}

func (s *ipBlockTestSuite) mockListLANsCall() *clienttest.MockClient_ListLANs_Call {
	return s.ionosClient.EXPECT().ListLANs(s.ctx, s.machineScope.DatacenterID())
}

func (s *ipBlockTestSuite) exampleLAN() sdk.Lan {
	return sdk.Lan{
		Id: ptr.To(exampleLANID),
		Properties: &sdk.LanProperties{
			Name: ptr.To(s.service.lanName(s.clusterScope.Cluster)),
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

func (s *ipBlockTestSuite) buildRequestWithName(name, status, method, id string) sdk.Request {
	opts := requestBuildOptions{
		status:     status,
		method:     method,
		url:        ipBlocksPath,
		href:       exampleRequestPath,
		targetType: sdk.IPBLOCK,
	}
	if id != "" {
		opts.url = path.Join(opts.url, id)
		opts.targetID = id
	}
	if method == http.MethodPost {
		opts.body = fmt.Sprintf(`{"properties":{"location":"%s","name":"%s","size":1}}`,
			s.clusterScope.Location(), name)
	}
	return s.exampleRequest(opts)
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
