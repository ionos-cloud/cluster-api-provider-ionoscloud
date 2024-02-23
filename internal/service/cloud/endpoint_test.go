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

func (s *EndpointTestSuite) TestGetIPBlock_MultipleMatches() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleName),
					Location: ptr.To(exampleLocation),
				},
			},
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To(exampleName),
					Location: ptr.To(exampleLocation),
				},
			},
		},
	}, nil).Once()
	blocks, err := s.service.getIPBlock(s.clusterScope)(s.ctx)
	s.Error(err)
	s.Nil(blocks)
}

func (s *EndpointTestSuite) TestGetIPBlock_SingleMatch() {
	name := ptr.To(exampleName)
	location := ptr.To(exampleLocation)
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     name,
					Location: location,
				},
			},
			{
				Properties: &sdk.IpBlockProperties{
					Name:     name,
					Location: ptr.To("es/vit"),
				},
			},
		},
	}, nil).Once()
	block, err := s.service.getIPBlock(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.NotNil(block)
	s.Equal(name, block.Properties.Name, "IP block name does not match")
	s.Equal(location, block.Properties.Location, "IP block location does not match")
}

func (s *EndpointTestSuite) TestGetIPBlock_UserSetIP() {
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = exampleIP
	name := ptr.To("random name")
	location := ptr.To(exampleLocation)
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     name,
					Location: location,
					Ips: &[]string{
						exampleIP,
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
	block, err := s.service.getIPBlock(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.Contains(*block.GetProperties().GetIps(), exampleIP)
}

func (s *EndpointTestSuite) TestGetIPBlock_PreviouslySetID() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointProviderID = exampleID
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
		Id: ptr.To(exampleID),
	}, nil).Once()
	block, err := s.service.getIPBlock(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.Equal(exampleID, *block.Id)
}

func (s *EndpointTestSuite) TestGetIPBlock_NoMatch() {
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
	block, err := s.service.getIPBlock(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.Nil(block)
}

func (s *EndpointTestSuite) TestReserveIPBlock_RequestSuccess() {
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

func (s *EndpointTestSuite) TestDeleteIPBlock_RequestSuccess() {
	requestPath := exampleRequestPath
	s.mockDeleteIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.deleteIPBlock(s.ctx, s.clusterScope, exampleID)
	s.NoError(err)
	req := s.clusterScope.IonosCluster.Status.CurrentClusterRequest
	s.NotNil(req)
	s.Equal(http.MethodDelete, req.Method)
	s.Equal(sdk.RequestStatusQueued, req.State)
	s.Equal(requestPath, req.RequestPath)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockCreationRequest_NoRequest() {
	s.mockGetRequestsCallPost().Return(make([]sdk.Request, 0), nil).Once()
	req, err := s.service.getLatestIPBlockCreationRequest(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.Nil(req)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockCreationRequest_Request() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodPost, "")
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallPost().Return(reqs, nil).Once()
	info, err := s.service.getLatestIPBlockCreationRequest(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.NotNil(info)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockDeletionRequest_NoRequest() {
	s.mockGetRequestsCallDelete(exampleID).Return(make([]sdk.Request, 0), nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleID)
	s.NoError(err)
	s.Nil(info)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockDeletionRequest_Request() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodDelete, exampleID)
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallDelete(exampleID).Return(reqs, nil).Once()
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleID)
	s.NoError(err)
	s.NotNil(info)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpoint_UserSetIP() {
	s.clusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = exampleIP
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Id: ptr.To(exampleID),
				Metadata: &sdk.DatacenterElementMetadata{
					State: ptr.To(sdk.Available),
				},
				Properties: &sdk.IpBlockProperties{
					Location: ptr.To(exampleLocation),
					Ips: &[]string{
						"another IP",
						exampleIP,
					},
				},
			},
		},
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Equal(s.clusterScope.GetControlPlaneEndpoint().Host, exampleIP)
	s.Equal(s.clusterScope.IonosCluster.Status.ControlPlaneEndpointProviderID, fmt.Sprintf("ionos://%s", exampleID))
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpoint_PendingRequest() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{
		s.buildRequest(sdk.RequestStatusRunning, http.MethodPost, ""),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath, exampleRequestPath)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.State, sdk.RequestStatusRunning)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.Method, http.MethodPost)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpoint_RequestNewIPBlock() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{}, nil).Once()
	s.mockReserveIPBlockCall().Return(exampleRequestPath, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpoint(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath, exampleRequestPath)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletion_CreationPendingRequest() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{}, nil).Once()
	s.mockGetRequestsCallPost().Return([]sdk.Request{
		s.buildRequest(sdk.RequestStatusRunning, http.MethodPost, ""),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath, exampleRequestPath)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.State, sdk.RequestStatusRunning)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.Method, http.MethodPost)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletion_UserSetIP() {
	s.clusterScope.IonosCluster.Status.ControlPlaneEndpointProviderID = exampleID
	s.mockGetIPBlockByIDCall().Return(&sdk.IpBlock{
		Id: ptr.To(exampleID),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletion_NoBlockLeft() {
	s.clusterScope.IonosCluster.SetCurrentClusterRequest(http.MethodPost, sdk.RequestStatusRunning, exampleRequestPath)
	s.mockListIPBlockCall().Return(nil, nil).Once()
	s.mockGetRequestsCallPost().Return(nil, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.False(requeue)
	s.NoError(err)
	s.Nil(s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletion_DeletionPendingRequest() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{
		{
			Id: ptr.To(exampleID),
			Properties: &sdk.IpBlockProperties{
				Name:     ptr.To(exampleName),
				Location: ptr.To(exampleLocation),
			},
		},
	}}, nil).Once()
	s.mockGetRequestsCallDelete(exampleID).Return([]sdk.Request{
		s.buildRequest(sdk.RequestStatusRunning, http.MethodDelete, exampleID),
	}, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.State, sdk.RequestStatusRunning)
}

func (s *EndpointTestSuite) TestReconcileControlPlaneEndpointDeletion_RequestNewDeletion() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{Items: &[]sdk.IpBlock{
		{
			Id: ptr.To(exampleID),
			Properties: &sdk.IpBlockProperties{
				Name:     ptr.To(exampleName),
				Location: ptr.To(exampleLocation),
			},
		},
	}}, nil).Once()
	s.mockGetRequestsCallDelete(exampleID).Return(nil, nil).Once()
	s.mockDeleteIPBlockCall().Return(exampleRequestPath, nil).Once()
	requeue, err := s.service.ReconcileControlPlaneEndpointDeletion(s.ctx, s.clusterScope)
	s.True(requeue)
	s.NoError(err)
	s.Equal(s.clusterScope.IonosCluster.Status.CurrentClusterRequest.RequestPath, exampleRequestPath)
}

func (s *EndpointTestSuite) mockReserveIPBlockCall() *clienttest.MockClient_ReserveIPBlock_Call {
	return s.ionosClient.
		EXPECT().
		ReserveIPBlock(s.ctx, s.service.ipBlockName(s.clusterScope), s.clusterScope.Location(), 1)
}

func (s *EndpointTestSuite) mockGetIPBlockByIDCall() *clienttest.MockClient_GetIPBlock_Call {
	return s.ionosClient.EXPECT().GetIPBlock(s.ctx, exampleID)
}

func (s *EndpointTestSuite) mockListIPBlockCall() *clienttest.MockClient_ListIPBlocks_Call {
	return s.ionosClient.EXPECT().ListIPBlocks(s.ctx)
}

func (s *EndpointTestSuite) mockDeleteIPBlockCall() *clienttest.MockClient_DeleteIPBlock_Call {
	return s.ionosClient.EXPECT().DeleteIPBlock(s.ctx, exampleID)
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
