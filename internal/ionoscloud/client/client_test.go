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

package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	exampleID       = "814fcf5c-41dd-45e1-b9dc-1192844848a7"
	examplePath     = "/a/b/c/d/e"
	exampleLocation = "de/fkb"

	// catchAllMockURL is a regex that matches all URLs, so no one needs to write
	// the proper path for each test that uses httpmock.
	catchAllMockURL = "=~^.*$"
)

func TestNewClient(t *testing.T) {
	w := "SET"
	tests := []struct {
		token      string
		apiURL     string
		shouldPass bool
	}{
		{w, w, true},
		{w, "", true},
		{"", w, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("token=%s apiURL=%s (shouldPass=%t)",
			tt.token, tt.apiURL, tt.shouldPass), func(t *testing.T) {
			c, err := NewClient(tt.token, tt.apiURL)
			if tt.shouldPass {
				require.NotNil(t, c, "NewClient returned a nil IonosCloudClient")
				require.NoError(t, err, "NewClient returned an error")
				cfg := c.API.GetConfig()
				require.Equal(t, tt.token, cfg.Token, "token didn't match")
				require.Equal(t, tt.apiURL, cfg.Host, "apiURL didn't match")
			} else {
				require.Nil(t, c, "NewClient returned a non-nil client")
				require.Error(t, err, "NewClient did not return an error")
			}
		})
	}
}

type IonosCloudClientTestSuite struct {
	*require.Assertions
	suite.Suite

	ctx    context.Context
	client *IonosCloudClient
}

func TestIonosCloudClientTestSuite(t *testing.T) {
	suite.Run(t, new(IonosCloudClientTestSuite))
}

func (s *IonosCloudClientTestSuite) SetupSuite() {
	s.Assertions = s.Require()
	s.ctx = context.Background()

	var err error
	s.client, err = NewClient("token", "localhost")
	s.NoError(err)

	httpmock.Activate()
}

func (*IonosCloudClientTestSuite) TearDownSuite() {
	httpmock.Deactivate()
}

func (*IonosCloudClientTestSuite) TearDownTest() {
	httpmock.Reset()
}

func (s *IonosCloudClientTestSuite) TestReserveIPBlockSuccess() {
	header := http.Header{}
	header.Add(locationHeaderKey, examplePath)
	responder := httpmock.NewJsonResponderOrPanic(http.StatusAccepted, map[string]any{}).HeaderSet(header)
	httpmock.RegisterResponder(
		http.MethodPost,
		catchAllMockURL,
		responder,
	)
	requestLocation, err := s.client.ReserveIPBlock(s.ctx, "test", "de/fkb", 1)
	s.NoError(err)
	s.Equal(examplePath, requestLocation)
}

func (s *IonosCloudClientTestSuite) TestReserveIPBlockFailure() {
	tcs := []struct {
		testName string
		name     string
		location string
		size     int32
	}{
		{"empty name", "", exampleLocation, 1},
		{"empty location", "test", "", 1},
		{"invalid size (zero)", "test", exampleLocation, 0},
		{"invalid size (negative)", "test", "exampleLocation", -1},
	}

	for _, test := range tcs {
		s.Run(test.testName, func() {
			requestLocation, err := s.client.ReserveIPBlock(s.ctx, test.name, test.location, test.size)
			s.Error(err)
			s.Empty(requestLocation)
		})
	}
}

func (s *IonosCloudClientTestSuite) TestGetIPBlockSuccess() {
	httpmock.RegisterResponder(
		http.MethodGet,
		catchAllMockURL,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{}),
	)
	ipBlock, err := s.client.GetIPBlock(s.ctx, exampleID)
	s.NoError(err)
	s.NotNil(ipBlock)
}

func (s *IonosCloudClientTestSuite) TestGetIPBlockFailureEmptyID() {
	ipBlock, err := s.client.GetIPBlock(s.ctx, "")
	s.Error(err)
	s.Nil(ipBlock)
}

func (s *IonosCloudClientTestSuite) TestListIPBlocksSuccess() {
	httpmock.RegisterResponder(
		http.MethodGet,
		catchAllMockURL,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{}),
	)
	ipBlocks, err := s.client.ListIPBlocks(s.ctx)
	s.NoError(err)
	s.NotNil(ipBlocks)
}

func (s *IonosCloudClientTestSuite) TestDeleteIPBlockSuccess() {
	header := http.Header{}
	header.Set(locationHeaderKey, examplePath)
	responder := httpmock.NewJsonResponderOrPanic(http.StatusAccepted, map[string]any{}).HeaderSet(header)
	httpmock.RegisterResponder(http.MethodDelete, catchAllMockURL, responder)
	requestLocation, err := s.client.DeleteIPBlock(s.ctx, exampleID)
	s.NoError(err)
	s.Equal(examplePath, requestLocation)
}

func (s *IonosCloudClientTestSuite) TestDeleteIPBlockFailureEmptyID() {
	requestLocation, err := s.client.DeleteIPBlock(s.ctx, "")
	s.Error(err)
	s.Empty(requestLocation)
}

func TestWithDepth(t *testing.T) {
	tests := []struct {
		depth int32
	}{
		{1},
		{2},
		{3},
		{4},
		{5},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("depth=%d", tt.depth), func(t *testing.T) {
			t.Parallel()
			c := &IonosCloudClient{}
			n, ok := WithDepth(c, tt.depth).(*IonosCloudClient)
			require.True(t, ok, "WithDepth didn't return an IonosCloudClient")
			require.Equal(t, tt.depth, n.requestDepth, "depth didn't match")
			require.NotEqualf(t, c, n, "WithDepth returned the same client")
		})
	}
}
