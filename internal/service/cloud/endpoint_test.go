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
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type EndpointTestSuite struct {
	ServiceTestSuite
}

func TestEndpointTestSuite(t *testing.T) {
	suite.Run(t, new(EndpointTestSuite))
}

func (s *EndpointTestSuite) TestGetIPBlock_MultipleMatches() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To("k8s-default-test-cluster"),
					Location: ptr.To(exampleLocation),
				},
			},
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To("k8s-default-test-cluster"),
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
	name := ptr.To("k8s-default-test-cluster")
	location := ptr.To(exampleLocation)
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
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
	s.NotNil(block)
	s.Equal(name, block.Properties.Name, "IP block name does not match")
	s.Equal(location, block.Properties.Location, "IP block location does not match")
}

func (s *EndpointTestSuite) TestGetIPBlock_NoMatch() {
	s.mockListIPBlockCall().Return(&sdk.IpBlocks{
		Items: &[]sdk.IpBlock{
			{
				Properties: &sdk.IpBlockProperties{
					Name:     ptr.To("k8s-default-test-cluster"),
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
	s.NotNil(s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
	req := infrav1.NewQueuedRequest(http.MethodPost, requestPath)
	s.Equal(req, *s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
}

func (s *EndpointTestSuite) TestDeleteIPBlock_RequestSuccess() {
	requestPath := exampleRequestPath
	s.mockDeleteIPBlockCall().Return(requestPath, nil).Once()
	err := s.service.deleteIPBlock(s.ctx, s.clusterScope, exampleID)
	s.NoError(err)
	s.NotNil(s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
	req := infrav1.NewQueuedRequest(http.MethodDelete, requestPath)
	s.Equal(req, *s.clusterScope.IonosCluster.Status.CurrentClusterRequest)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockCreationRequest_NoRequest() {
	s.mockGetRequestsCallPost().Return(make([]sdk.Request, 0), nil)
	req, err := s.service.getLatestIPBlockCreationRequest(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.Nil(req)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockCreationRequest_Request() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodPost, "")
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallPost().Return(reqs, nil)
	info, err := s.service.getLatestIPBlockCreationRequest(s.clusterScope)(s.ctx)
	s.NoError(err)
	s.NotNil(info)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockDeletionRequest_NoRequest() {
	s.mockGetRequestsCallDelete(exampleID).Return(make([]sdk.Request, 0), nil)
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleID)
	s.NoError(err)
	s.Nil(info)
}

func (s *EndpointTestSuite) TestGetLatestIPBlockDeletionRequest_Request() {
	req := s.buildRequest(sdk.RequestStatusQueued, http.MethodDelete, exampleID)
	reqs := []sdk.Request{req}
	s.mockGetRequestsCallDelete(exampleID).Return(reqs, nil)
	info, err := s.service.getLatestIPBlockDeletionRequest(s.ctx, exampleID)
	s.NoError(err)
	s.NotNil(info)
}

func (s *EndpointTestSuite) mockReserveIPBlockCall() *clienttest.MockClient_ReserveIPBlock_Call {
	return s.ionosClient.
		EXPECT().
		ReserveIPBlock(s.ctx, s.service.defaultResourceName(s.clusterScope), s.clusterScope.Location(), 1)
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
			s.clusterScope.Location(), s.service.defaultResourceName(s.clusterScope))
	}
	return s.exampleRequest(opts)
}
