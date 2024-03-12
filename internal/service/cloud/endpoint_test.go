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

	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type EndpointTestSuite struct {
	ServiceTestSuite
}

func TestEndpointTestSuite(t *testing.T) {
	suite.Run(t, new(EndpointTestSuite))
}

const (
	exampleName = "k8s-default-test-cluster"
)

func (s *EndpointTestSuite) TestGetIPBlockFuncMultipleMatches() {
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

func (s *EndpointTestSuite) TestGetIPBlockFuncSingleMatch() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			*exampleIPBlock(),
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleName),
					Location: ptr.To("es/vit"),
				},
			},
		},
	}, nil).Once()
	s.mockGetIPBlockByIDCall().Return(exampleIPBlock(), nil).Once()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.NotNil(block)
	s.Equal(exampleName, *block.Properties.Name, "IP block name does not match")
	s.Equal(exampleLocation, *block.Properties.Location, "IP block location does not match")
	s.Equal(sdk.Available, *block.Metadata.State, "IP block state does not match")
}

func (s *EndpointTestSuite) TestGetIPBlockFuncUserSetIP() {
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

func (s *EndpointTestSuite) TestGetIPBlockPreviouslySetID() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
	}, nil).Once()

	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Equal(exampleIPBlockID, *block.Id)
}

func (s *EndpointTestSuite) TestGetIPBlockPreviouslySetIDNotFound() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall().Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, 404)).Once()
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{},
	}, nil).Once()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.ErrorAs(err, &sdk.GenericOpenAPIError{})
	s.Nil(block)
}

func (s *EndpointTestSuite) TestGetIPBlockNoMatch() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleName),
					Location: ptr.To("de/fra"),
				},
			},
		},
	}, nil).Once()
	block, err := s.service.getIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Nil(block)
}

func (s *EndpointTestSuite) TestReserveIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	s.mockReserveIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.reserveIPBlock(s.ctx, s.clusterScope)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.NotNil(req)
	s.Equal(http.MethodPost, req.Method)
	s.Equal(sdk.RequestStatusQueued, req.State)
	s.Equal(requestPath, req.RequestPath)
}

func (s *EndpointTestSuite) TestDeleteIPBlockRequestSuccess() {
	requestPath := exampleRequestPath
	s.mockDeleteIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.deleteIPBlock(s.ctx, s.clusterScope, exampleIPBlockID)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.NotNil(req)
	s.Equal(http.MethodDelete, req.Method)
	s.Equal(sdk.RequestStatusQueued, req.State)
	s.Equal(requestPath, req.RequestPath)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockCreationRequestNoRequest() {
	s.mockGetRequestsCallPost().Return(make([]sdk.Request, 0), nil).Once()
	req, err := s.service.getLatestIPBlockCreationRequest(s.ctx, s.clusterScope)
	s.NoError(err)
	s.Nil(req)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockCreationRequestRequest() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodPost, "")
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallPost().Return(reqs, nil).Once()
	info, err := s.service.getLatestIPBlockCreationRequest(s.ctx, s.clusterScope)
	s.NoError(err)
	s.NotNil(info)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockDeletionRequestNoRequest() {
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(make([]sdk.Request, 0), nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleIPBlockID)
	s.NoError(err)
	s.Nil(info)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockDeletionRequestRequest() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodDelete, exampleIPBlockID)
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallDelete(exampleIPBlockID).Return(reqs, nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleIPBlockID)
	s.NoError(err)
	s.NotNil(info)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointUserSetIP() {
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

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointUserSetPort() {
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

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointPendingRequest() {
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

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointRequestNewIPBlock() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{}, nil).Once()
	s.mockReserveIPBlockCall().Return(exampleRequestPath, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(exampleRequestPath, s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletionCreationPendingRequest() {
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

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletionUserSetIPWithIPBlockID() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointIPBlockID = exampleIPBlockID
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletionUserSetIPWithoutIPBlockID() {
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

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletionNoBlockLeft() {
	s.infraCluster.Spec.ControlPlaneEndpoint.Host = exampleEndpointIP
	s.mockListIPBlockCall().Return(nil, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Nil(s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletionDeletionPendingRequest() {
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

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletionRequestNewDeletion() {
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

func (s *EndpointTestSuite) mockReserveIPBlockCall() *clienttest.MockClient_ReserveIPBlock_Call {
	return s.ionosClient.
		EXPECT().
		ReserveIPBlock(s.ctx, s.service.ipBlockName(s.clusterScope), s.clusterScope.Location(), int32(1))
}

func (s *EndpointTestSuite) mockGetIPBlockByIDCall() *clienttest.MockClient_GetIPBlock_Call {
	return s.ionosClient.EXPECT().GetIPBlock(s.ctx, exampleIPBlockID)
}

func (s *EndpointTestSuite) mockListIPBlockCall() *clienttest.MockClient_ListIPBlocks_Call {
	return s.ionosClient.EXPECT().ListIPBlocks(s.ctx)
}

func (s *EndpointTestSuite) mockDeleteIPBlockCall() *clienttest.MockClient_DeleteIPBlock_Call {
	return s.ionosClient.EXPECT().DeleteIPBlock(s.ctx, exampleIPBlockID)
}

func (s *EndpointTestSuite) mockGetRequestsCallPost() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, ipBlocksPath)
}

func (s *EndpointTestSuite) mockGetRequestsCallDelete(id string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodDelete, path.Join(ipBlocksPath, id))
}

func (s *EndpointTestSuite) buildRequest(status string, method, id string) sdk.Request {
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
			s.clusterScope.Location(), s.service.ipBlockName(s.clusterScope))
	}
	return s.exampleRequest(opts)
}

// exampleIPBlock returns a new sdk.IpBlock instance for testing. The IPs need to be set.
func exampleIPBlock() *sdk.IpBlock {
	return &sdk.IpBlock{
		Id: ptr.To(exampleIPBlockID),
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Properties: &sdk.IpBlockProperties{
			Name:     ptr.To(exampleName),
			Location: ptr.To(exampleLocation),
			Ips: &[]string{
				exampleEndpointIP,
			},
		},
	}
}
