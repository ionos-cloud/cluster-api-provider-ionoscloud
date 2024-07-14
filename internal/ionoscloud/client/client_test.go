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
	// taken from crypto/x509 test files
	const rootPEM = `
-----BEGIN CERTIFICATE-----
MIIEBDCCAuygAwIBAgIDAjppMA0GCSqGSIb3DQEBBQUAMEIxCzAJBgNVBAYTAlVT
MRYwFAYDVQQKEw1HZW9UcnVzdCBJbmMuMRswGQYDVQQDExJHZW9UcnVzdCBHbG9i
YWwgQ0EwHhcNMTMwNDA1MTUxNTU1WhcNMTUwNDA0MTUxNTU1WjBJMQswCQYDVQQG
EwJVUzETMBEGA1UEChMKR29vZ2xlIEluYzElMCMGA1UEAxMcR29vZ2xlIEludGVy
bmV0IEF1dGhvcml0eSBHMjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AJwqBHdc2FCROgajguDYUEi8iT/xGXAaiEZ+4I/F8YnOIe5a/mENtzJEiaB0C1NP
VaTOgmKV7utZX8bhBYASxF6UP7xbSDj0U/ck5vuR6RXEz/RTDfRK/J9U3n2+oGtv
h8DQUB8oMANA2ghzUWx//zo8pzcGjr1LEQTrfSTe5vn8MXH7lNVg8y5Kr0LSy+rE
ahqyzFPdFUuLH8gZYR/Nnag+YyuENWllhMgZxUYi+FOVvuOAShDGKuy6lyARxzmZ
EASg8GF6lSWMTlJ14rbtCMoU/M4iarNOz0YDl5cDfsCx3nuvRTPPuj5xt970JSXC
DTWJnZ37DhF5iR43xa+OcmkCAwEAAaOB+zCB+DAfBgNVHSMEGDAWgBTAephojYn7
qwVkDBF9qn1luMrMTjAdBgNVHQ4EFgQUSt0GFhu89mi1dvWBtrtiGrpagS8wEgYD
VR0TAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAQYwOgYDVR0fBDMwMTAvoC2g
K4YpaHR0cDovL2NybC5nZW90cnVzdC5jb20vY3Jscy9ndGdsb2JhbC5jcmwwPQYI
KwYBBQUHAQEEMTAvMC0GCCsGAQUFBzABhiFodHRwOi8vZ3RnbG9iYWwtb2NzcC5n
ZW90cnVzdC5jb20wFwYDVR0gBBAwDjAMBgorBgEEAdZ5AgUBMA0GCSqGSIb3DQEB
BQUAA4IBAQA21waAESetKhSbOHezI6B1WLuxfoNCunLaHtiONgaX4PCVOzf9G0JY
/iLIa704XtE7JW4S615ndkZAkNoUyHgN7ZVm2o6Gb4ChulYylYbc3GrKBIxbf/a/
zG+FA1jDaFETzf3I93k9mTXwVqO94FntT0QJo544evZG0R0SnU++0ED8Vf4GXjza
HFa9llF7b1cq26KqltyMdMKVvvBulRP/F/A8rLIQjcxz++iPAsbw+zOzlTvjwsto
WHPbqCRiOwY1nQ2pM714A5AuTHhdUDqB1O6gyHA43LL5Z/qHQF1hwFGPa4NrzQU6
yuGnBXj8ytqU0CwIPX4WecigUCAkVDNx
-----END CERTIFICATE-----`

	const set = "SET"

	tests := []struct {
		name       string
		token      string
		apiURL     string
		caBundle   []byte
		shouldPass bool
	}{
		{
			"token set",
			set,
			"",
			nil,
			true,
		},
		{
			"token and URL set",
			set,
			set,
			nil,
			true,
		},
		{
			"token missing",
			"",
			set,
			nil,
			false,
		},
		{
			"token and CA bundle set",
			set,
			"",
			[]byte(rootPEM),
			true,
		},
		{
			"invalid CA bundle",
			set,
			"",
			[]byte("invalid"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.token, tt.apiURL, tt.caBundle)
			if tt.shouldPass {
				require.NotNil(t, c, "NewClient returned a nil IonosCloudClient")
				require.NoError(t, err, "NewClient returned an error")
				cfg := c.API.GetConfig()
				require.Equal(t, tt.token, cfg.Token, "token didn't match")
				require.Equal(t, tt.apiURL, cfg.Host, "apiURL didn't match")
				if tt.caBundle != nil {
					require.NotNil(t, cfg.HTTPClient, "HTTP client is nil")
					require.NotNil(t, cfg.HTTPClient.Transport, "HTTP client lacks a custom transport")
					require.IsType(t, &http.Transport{}, cfg.HTTPClient.Transport, "transport is not an http.Transport")
					require.NotNil(t, cfg.HTTPClient.Transport.(*http.Transport).TLSClientConfig,
						"TLSClientConfig is nil")
					require.NotNil(t, cfg.HTTPClient.Transport.(*http.Transport).TLSClientConfig.RootCAs,
						"RootCAs is nil")
				} else {
					require.Equal(t, http.DefaultClient, cfg.HTTPClient, "HTTP client is not the default client")
				}
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
	s.client, err = NewClient("token", "localhost", nil)
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

func (s *IonosCloudClientTestSuite) TestGetImage() {
	httpmock.RegisterResponder(
		http.MethodGet,
		catchAllMockURL,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{}),
	)
	image, err := s.client.GetImage(s.ctx, exampleID)
	s.NoError(err)
	s.NotNil(image)
}

func (s *IonosCloudClientTestSuite) TestListLabels() {
	httpmock.RegisterResponder(
		http.MethodGet,
		catchAllMockURL,
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
			"items": []map[string]any{
				{},
			},
		}),
	)
	labels, err := s.client.ListLabels(s.ctx)
	s.NoError(err)
	s.NotEmpty(labels)
}
