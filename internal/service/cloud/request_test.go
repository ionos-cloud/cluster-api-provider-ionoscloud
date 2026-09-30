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
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
)

const (
	baseTestURL      = "https://url.tld/path"
	messageEmptyText = "message should be empty"
)

func TestMatcher_MatchByName(t *testing.T) {
	matchByNameFunc := matchByName[*sdk.Server, *sdk.ServerProperties]("test")
	require.False(t, matchByNameFunc(&sdk.Server{}, sdk.Request{}))
	testServer := sdk.Server{
		Properties: &sdk.ServerProperties{
			Name: new("test"),
		},
	}
	require.True(t, matchByNameFunc(&testServer, sdk.Request{}))
	testServer.Properties.Name = new("wrong")
	require.False(t, matchByNameFunc(&testServer, sdk.Request{}))

	// l := (&sdk.Info{}).GetName()
	// the following line generates a compiler error, so validity is checked at compile time
	// matchByNameInvalidFunc := matchByName[*sdk.Server, *sdk.Info]("test")
}

type getRequestStatusSuite struct {
	ServiceTestSuite
}

func TestGetRequestStatusTestSuite(t *testing.T) {
	suite.Run(t, new(getRequestStatusSuite))
}

func (s *getRequestStatusSuite) TestGetRequestStatusMissingMetadata() {
	s.mockCheckRequestStatusCall(baseTestURL).Return(&sdk.RequestStatus{
		Href:     new(baseTestURL),
		Id:       new("12345"),
		Metadata: nil,
	}, nil).Once()

	status, message, err := s.service.GetRequestStatus(s.ctx, baseTestURL)
	s.Error(err, "should return an error but didn't")
	s.Empty(status, "status should be empty")
	s.Empty(message, messageEmptyText)

	s.mockCheckRequestStatusCall(baseTestURL).Return(&sdk.RequestStatus{
		Metadata: &sdk.RequestStatusMetadata{},
	}, nil).Once()

	status, message, err = s.service.GetRequestStatus(s.ctx, baseTestURL)
	s.Error(err, "should return an error but didn't")
	s.Empty(status, "status should be empty")
	s.Empty(message, messageEmptyText)
}

func (s *getRequestStatusSuite) TestGetRequestStatus() {
	s.mockCheckRequestStatusCall(baseTestURL).Return(&sdk.RequestStatus{
		Href: new(baseTestURL),
		Id:   new("12345"),
		Metadata: &sdk.RequestStatusMetadata{
			Status:  new(sdk.RequestStatusFailed),
			Message: new("Failed to do foo and bar"),
		},
	}, nil).Once()

	status, message, err := s.service.GetRequestStatus(s.ctx, baseTestURL)
	s.NoError(err, "should not return an error but did")
	s.Equal(sdk.RequestStatusFailed, status, "status should be FAILED")
	s.Equal("Failed to do foo and bar", message, "message should be 'Failed to do foo and bar'")

	s.mockCheckRequestStatusCall(baseTestURL).Return(&sdk.RequestStatus{
		Href: new(baseTestURL),
		Id:   new("12345"),
		Metadata: &sdk.RequestStatusMetadata{
			Status:  new(sdk.RequestStatusQueued),
			Message: nil,
		},
	}, nil).Once()

	status, message, err = s.service.GetRequestStatus(s.ctx, baseTestURL)
	s.NoError(err, "should not return an error but did")
	s.Equal(sdk.RequestStatusQueued, status, "status should be FAILED")
	s.Empty(message, messageEmptyText)
}

func (s *getRequestStatusSuite) mockCheckRequestStatusCall(
	requestURL string,
) *clienttest.MockClient_CheckRequestStatus_Call {
	return s.ionosClient.EXPECT().CheckRequestStatus(s.ctx, requestURL)
}

type getMatchingRequestSuite struct {
	ServiceTestSuite
}

func TestGetMatchingRequestTestSuite(t *testing.T) {
	suite.Run(t, new(getMatchingRequestSuite))
}

func (s *getMatchingRequestSuite) examplePostRequest(href, status string) sdk.Request {
	// we use a LAN as an example target type here
	opts := requestBuildOptions{
		status:     status,
		method:     http.MethodPost,
		url:        baseTestURL + "/?depth=10",
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.lanName(s.clusterScope.Cluster)),
		href:       href,
		targetID:   "1",
		targetType: sdk.LAN,
	}
	return s.exampleRequest(opts)
}

func (s *getMatchingRequestSuite) TestUnsupportedResourceType() {
	request, err := getMatchingRequest[int](
		s.ctx,
		s.service,
		http.MethodPost,
		"/path",
	)
	s.ErrorContains(err, "unsupported")
	s.Nil(request)
}

func (s *getMatchingRequestSuite) TestMatching() {
	// req1 has a mismatch in its target type
	req1 := s.examplePostRequest("req1", sdk.RequestStatusQueued)
	(*req1.Metadata.RequestStatus.Metadata.Targets)[0].Target.Type = new(sdk.SERVER)

	// req2 has a mismatch in its URL
	req2 := s.examplePostRequest("req2", sdk.RequestStatusQueued)
	*req2.Properties.Url = baseTestURL + "/action?depth=10"

	// req3 doesn't fulfill the matcher function
	req3 := s.examplePostRequest("req3", sdk.RequestStatusQueued)
	renamed := strings.Replace(*req3.Properties.Body, s.service.lanName(s.clusterScope.Cluster), "wrongName", 1)
	req3.Properties.Body = &renamed

	// req4 is the one we want to find
	req4 := s.examplePostRequest("req4", sdk.RequestStatusFailed)

	// req5 would also match, but req4 is found first
	req5 := s.examplePostRequest("req6", sdk.RequestStatusDone)

	s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, "path").
		Return([]sdk.Request{req1, req2, req3, req4, req5}, nil)

	request, err := getMatchingRequest(
		s.ctx,
		s.service,
		http.MethodPost,
		"path?foo=bar&baz=qux",
		func(resource *sdk.Lan, _ sdk.Request) bool {
			return *resource.Properties.Name == s.service.lanName(s.clusterScope.Cluster)
		},
	)
	s.NoError(err)
	s.NotNil(request)
	s.Equal("req4", request.location)
	s.Equal(sdk.RequestStatusFailed, request.status)
}

func TestHasRequestTargetType(t *testing.T) {
	req := sdk.Request{
		Metadata: &sdk.RequestMetadata{
			RequestStatus: &sdk.RequestStatus{
				Metadata: &sdk.RequestStatusMetadata{},
			},
		},
	}
	require.False(t, hasRequestTargetType(req, sdk.LAN))

	req.Metadata.RequestStatus.Metadata.Targets = &[]sdk.RequestTarget{}
	require.False(t, hasRequestTargetType(req, sdk.LAN))

	req.Metadata.RequestStatus.Metadata.Targets = &[]sdk.RequestTarget{
		{
			Target: &sdk.ResourceReference{Type: new(sdk.SERVER)},
		},
	}
	require.False(t, hasRequestTargetType(req, sdk.LAN))

	(*req.Metadata.RequestStatus.Metadata.Targets)[0].Target.Type = new(sdk.LAN)
	require.True(t, hasRequestTargetType(req, sdk.LAN))
}

type findResourceSuite struct {
	ServiceTestSuite
}

func TestFindResourceTestSuite(t *testing.T) {
	suite.Run(t, new(findResourceSuite))
}

func (s *findResourceSuite) TestListingIsEnough() {
	resource, request, err := findResource(
		s.ctx,
		func(_ context.Context) (*int, error) { return new(42), nil },
		func(_ context.Context) (*requestInfo, error) { panic("don't call me") },
	)
	s.NoError(err)
	s.Nil(request)
	s.NotNil(resource)
	s.Equal(42, *resource)
}

func (s *findResourceSuite) TestFoundRequest() {
	wantedRequest := &requestInfo{status: sdk.RequestStatusQueued}

	resource, gotRequest, err := findResource(
		s.ctx,
		func(_ context.Context) (*int, error) { return nil, nil },
		func(_ context.Context) (*requestInfo, error) { return wantedRequest, nil },
	)
	s.NoError(err)
	s.Nil(resource)
	s.NotNil(gotRequest)
	s.Equal(wantedRequest, gotRequest)
}

func (s *findResourceSuite) TestFoundOnSecondListing() {
	listCalls := 0
	resource, gotRequest, err := findResource(
		s.ctx,
		func(_ context.Context) (*int, error) {
			listCalls++
			if listCalls == 1 {
				return nil, nil
			}
			return new(42), nil
		},
		func(_ context.Context) (*requestInfo, error) { return &requestInfo{status: sdk.RequestStatusDone}, nil },
	)
	s.Equal(2, listCalls)
	s.NoError(err)
	s.Nil(gotRequest)
	s.NotNil(resource)
	s.Equal(42, *resource)
}

func (s *findResourceSuite) TestIgnoreNotFound() {
	tests := []struct {
		name          string
		inputStatus   string
		expectedCount int
	}{{
		name:          "Ignore not found on first lookup",
		inputStatus:   sdk.RequestStatusRunning,
		expectedCount: 1,
	}, {
		name:          "Ignore not found on first and second lookup",
		inputStatus:   sdk.RequestStatusDone,
		expectedCount: 2,
	}}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			listCalls := 0
			resource, gotRequest, err := findResource(
				s.ctx,
				func(_ context.Context) (*int, error) {
					listCalls++
					return nil, sdk.NewGenericOpenAPIError("", nil, nil, http.StatusNotFound)
				},
				func(_ context.Context) (*requestInfo, error) { return &requestInfo{status: tt.inputStatus}, nil },
			)

			s.Equal(tt.expectedCount, listCalls)
			s.NoError(err)
			s.NotNil(gotRequest)
			s.Nil(resource)
		})
	}
}

func TestRequestInfo(t *testing.T) {
	req := requestInfo{status: sdk.RequestStatusFailed}
	require.False(t, req.isPending())
	require.False(t, req.isDone())

	req.status = sdk.RequestStatusQueued
	require.True(t, req.isPending())
	require.False(t, req.isDone())

	req.status = sdk.RequestStatusRunning
	require.True(t, req.isPending())
	require.False(t, req.isDone())

	req.status = sdk.RequestStatusDone
	require.False(t, req.isPending())
	require.True(t, req.isDone())
}

func TestMetadataHolder(t *testing.T) {
	lan1 := &sdk.Lan{Metadata: &sdk.DatacenterElementMetadata{State: new("BUSY")}}
	lan2 := &sdk.Lan{Metadata: &sdk.DatacenterElementMetadata{State: new(sdk.Available)}}

	require.False(t, isAvailable(getState(lan1)))
	require.True(t, isAvailable(getState(lan2)))
}
