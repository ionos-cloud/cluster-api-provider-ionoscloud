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
	"strings"
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type getRequestStatusSuite struct {
	ServiceTestSuite
}

func TestGetRequestStatusTestSuite(t *testing.T) {
	suite.Run(t, new(getRequestStatusSuite))
}

func (s *getRequestStatusSuite) TestGetRequestStatusMissingMetadata() {
	const testURL = "https://url.tld/path"
	s.mockCheckRequestStatusCall(testURL).Return(&sdk.RequestStatus{
		Href:     ptr.To("https://url.tld/path"),
		Id:       ptr.To("12345"),
		Metadata: nil,
	}, nil).Once()

	status, message, err := s.service.GetRequestStatus(s.ctx, testURL)
	s.Error(err, "should return an error but didn't")
	s.Empty(status, "status should be empty")
	s.Empty(message, "message should be empty")

	s.mockCheckRequestStatusCall(testURL).Return(&sdk.RequestStatus{
		Metadata: &sdk.RequestStatusMetadata{},
	}, nil).Once()

	status, message, err = s.service.GetRequestStatus(s.ctx, testURL)
	s.Error(err, "should return an error but didn't")
	s.Empty(status, "status should be empty")
	s.Empty(message, "message should be empty")
}

func (s *getRequestStatusSuite) TestGetRequestStatus() {
	const testURL = "https://url.tld/path"
	s.mockCheckRequestStatusCall(testURL).Return(&sdk.RequestStatus{
		Href: ptr.To("https://url.tld/path"),
		Id:   ptr.To("12345"),
		Metadata: &sdk.RequestStatusMetadata{
			Status:  ptr.To(sdk.RequestStatusFailed),
			Message: ptr.To("Failed to do foo and bar"),
		},
	}, nil).Once()

	status, message, err := s.service.GetRequestStatus(s.ctx, testURL)
	s.NoError(err, "should not return an error but did")
	s.Equal(sdk.RequestStatusFailed, status, "status should be FAILED")
	s.Equal("Failed to do foo and bar", message, "message should be 'Failed to do foo and bar'")

	s.mockCheckRequestStatusCall(testURL).Return(&sdk.RequestStatus{
		Href: ptr.To("https://url.tld/path"),
		Id:   ptr.To("12345"),
		Metadata: &sdk.RequestStatusMetadata{
			Status:  ptr.To(sdk.RequestStatusQueued),
			Message: nil,
		},
	}, nil).Once()

	status, message, err = s.service.GetRequestStatus(s.ctx, testURL)
	s.NoError(err, "should not return an error but did")
	s.Equal(sdk.RequestStatusQueued, status, "status should be FAILED")
	s.Empty(message, "message should be empty")
}

func (s *getRequestStatusSuite) mockCheckRequestStatusCall(requestURL string) *clienttest.MockClient_CheckRequestStatus_Call {
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
		url:        "https://url.tld/path/?depth=10",
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.lanName()),
		href:       href,
		targetID:   "1",
		targetType: sdk.LAN,
	}
	return s.exampleRequest(opts)
}

func (s *getMatchingRequestSuite) TestUnsupportedResourceType() {
	request, err := getMatchingRequest[int](
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
	(*req1.Metadata.RequestStatus.Metadata.Targets)[0].Target.Type = ptr.To(sdk.SERVER)

	// req2 has a mismatch in its URL
	req2 := s.examplePostRequest("req2", sdk.RequestStatusQueued)
	*req2.Properties.Url = "https://url.tld/path/action?depth=10"

	// req3 doesn't fulfill the matcher function
	req3 := s.examplePostRequest("req3", sdk.RequestStatusQueued)
	renamed := strings.Replace(*req3.Properties.Body, s.service.lanName(), "wrongName", 1)
	req3.Properties.Body = ptr.To(renamed)

	// req4 is the one we want to find
	req4 := s.examplePostRequest("req4", sdk.RequestStatusFailed)

	// req5 would also match, but req4 is found first
	req5 := s.examplePostRequest("req6", sdk.RequestStatusDone)

	s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodPost, "path").
		Return([]sdk.Request{req1, req2, req3, req4, req5}, nil)

	request, err := getMatchingRequest(
		s.service,
		http.MethodPost,
		"path?foo=bar&baz=qux",
		func(resource sdk.Lan, _ sdk.Request) bool {
			return *resource.Properties.Name == s.service.lanName()
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
			Target: &sdk.ResourceReference{Type: ptr.To(sdk.SERVER)},
		},
	}
	require.False(t, hasRequestTargetType(req, sdk.LAN))

	(*req.Metadata.RequestStatus.Metadata.Targets)[0].Target.Type = ptr.To(sdk.LAN)
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
		func() (*int, error) { return ptr.To(42), nil },
		func() (*requestInfo, error) { panic("don't call me") },
	)
	s.NoError(err)
	s.Nil(request)
	s.NotNil(resource)
	s.Equal(42, *resource)
}

func (s *findResourceSuite) TestFoundRequest() {
	wantedRequest := &requestInfo{status: sdk.RequestStatusQueued}

	resource, gotRequest, err := findResource(
		func() (*int, error) { return nil, nil },
		func() (*requestInfo, error) { return wantedRequest, nil },
	)
	s.NoError(err)
	s.Nil(resource)
	s.NotNil(gotRequest)
	s.Equal(wantedRequest, gotRequest)
}

func (s *findResourceSuite) TestFoundOnSecondListing() {
	listCalls := 0
	resource, gotRequest, err := findResource(
		func() (*int, error) {
			listCalls++
			if listCalls == 1 {
				return nil, nil
			}
			return ptr.To(42), nil
		},
		func() (*requestInfo, error) { return &requestInfo{status: sdk.RequestStatusDone}, nil },
	)
	s.Equal(2, listCalls)
	s.NoError(err)
	s.Nil(gotRequest)
	s.NotNil(resource)
	s.Equal(42, *resource)
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
	lan1 := &sdk.Lan{Metadata: &sdk.DatacenterElementMetadata{State: ptr.To("BUSY")}}
	lan2 := &sdk.Lan{Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(stateAvailable)}}

	require.False(t, isAvailable(getState(lan1)))
	require.True(t, isAvailable(getState(lan2)))
}
