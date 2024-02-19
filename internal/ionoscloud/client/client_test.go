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

	// catchAllMockUrl is a regex that matches all URLs, so no one needs to write
	// the proper path for each test that uses httpmock.
	catchAllMockUrl = "=~^.*$"
)

func TestNewClient(t *testing.T) {
	w := "SET"
	tests := []struct {
		username   string
		password   string
		token      string
		apiURL     string
		shouldPass bool
	}{
		{w, w, w, w, true},
		{w, w, w, "", true},
		{"", "", w, w, true},
		{"", "", w, "", true},
		{w, w, "", w, true},
		{w, "", w, w, false},
		{w, "", "", w, false},
		{"", w, w, w, false},
		{"", w, "", w, false},
		{"", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("username=%s password=%s token=%s apiURL=%s (shouldPass=%t)",
			tt.username, tt.password, tt.token, tt.apiURL, tt.shouldPass), func(t *testing.T) {
			c, err := NewClient(tt.username, tt.password, tt.token, tt.apiURL)
			if tt.shouldPass {
				require.NotNil(t, c, "NewClient returned a nil IonosCloudClient")
				require.NoError(t, err, "NewClient returned an error")
				cfg := c.API.GetConfig()
				require.Equal(t, tt.username, cfg.Username, "username didn't match")
				require.Equal(t, tt.password, cfg.Password, "password didn't match")
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
	client *IonosCloudClient
}

func TestIonosCloudClientTestSuite(t *testing.T) {
	suite.Run(t, new(IonosCloudClientTestSuite))
}

func (s *IonosCloudClientTestSuite) SetupSuite() {
	s.Assertions = s.Require()
	var err error
	s.client, err = NewClient("username", "password", "", "localhost")
	s.NoError(err)
	httpmock.Activate()
}

func (s *IonosCloudClientTestSuite) TearDownSuite() {
	httpmock.Deactivate()
}

func (s *IonosCloudClientTestSuite) TearDownTest() {
	httpmock.Reset()
}

func (s *IonosCloudClientTestSuite) TestReserveIPBlock_Success() {
	header := http.Header{}
	header.Add(locationHeaderKey, examplePath)
	responder := httpmock.NewJsonResponderOrPanic(http.StatusAccepted, map[string]any{}).HeaderSet(header)
	httpmock.RegisterResponder(
		http.MethodPost,
		catchAllMockUrl,
		responder,
	)
	requestLocation, err := s.client.ReserveIPBlock(nil, "test", "de/fkb", 1)
	s.NoError(err)
	s.Equal(examplePath, requestLocation)
}

func (s *IonosCloudClientTestSuite) TestReserveIPBlock_Failure() {
	tcs := []struct {
		testName string
		name     string
		location string
		size     int
	}{
		{"empty name", "", exampleLocation, 1},
		{"empty location", "test", "", 1},
		{"invalid size (zero)", "test", exampleLocation, 0},
		{"invalid size (negative)", "test", "exampleLocation", -1},
	}

	for _, test := range tcs {
		s.Run(test.testName, func() {
			requestLocation, err := s.client.ReserveIPBlock(nil, test.name, test.location, test.size)
			s.Error(err)
			s.Empty(requestLocation)
		})
	}
}

func (s *IonosCloudClientTestSuite) TestGetIPBlock_Success() {
	httpmock.RegisterResponder(
		http.MethodGet,
		catchAllMockUrl,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{}),
	)
	ipBlock, err := s.client.GetIPBlock(nil, exampleID)
	s.NoError(err)
	s.NotNil(ipBlock)
}

func (s *IonosCloudClientTestSuite) TestGetIPBlock_Failure_EmptyID() {
	ipBlock, err := s.client.GetIPBlock(nil, "")
	s.Error(err)
	s.Nil(ipBlock)
}

func (s *IonosCloudClientTestSuite) TestListIPBlocks_Success() {
	httpmock.RegisterResponder(
		http.MethodGet,
		catchAllMockUrl,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{}),
	)
	ipBlocks, err := s.client.ListIPBlocks(nil)
	s.NoError(err)
	s.NotNil(ipBlocks)
}

func (s *IonosCloudClientTestSuite) TestDeleteIPBlock_Success() {
	header := http.Header{}
	header.Set(locationHeaderKey, examplePath)
	responder := httpmock.NewJsonResponderOrPanic(http.StatusAccepted, map[string]any{}).HeaderSet(header)
	httpmock.RegisterResponder(http.MethodDelete, catchAllMockUrl, responder)
	requestLocation, err := s.client.DeleteIPBlock(nil, exampleID)
	s.NoError(err)
	s.Equal(examplePath, requestLocation)
}

func (s *IonosCloudClientTestSuite) TestDeleteIPBlock_Failure_EmptyID() {
	requestLocation, err := s.client.DeleteIPBlock(nil, "")
	s.Error(err)
	s.Empty(requestLocation)
}
