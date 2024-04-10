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

// Package client contains an implementation of the Client interface defined in internal/ionoscloud.
package client

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
)

const (
	locationHeaderKey = "Location"
)

// IonosCloudClient is a concrete implementation of the Client interface defined in the internal client package that
// communicates with Cloud API using its SDK.
type IonosCloudClient struct {
	API          *sdk.APIClient
	requestDepth int32
}

var _ ionoscloud.Client = &IonosCloudClient{}

// NewClient instantiates a usable IonosCloudClient.
// The client needs a token to work. Basic auth is not be supported.
func NewClient(token, apiURL string) (*IonosCloudClient, error) {
	if token == "" {
		return nil, errors.New("token must be set")
	}
	cfg := sdk.NewConfiguration("", "", token, apiURL)
	apiClient := sdk.NewAPIClient(cfg)
	return &IonosCloudClient{
		API: apiClient,
	}, nil
}

// WithDepth creates a temporary copy of the client, where a custom depth can be set.
func WithDepth(client ionoscloud.Client, depth int32) ionoscloud.Client {
	if t, ok := client.(*IonosCloudClient); ok {
		c := clone(t)
		c.requestDepth = depth
		return c
	}

	return client
}

func clone(client *IonosCloudClient) *IonosCloudClient {
	return &IonosCloudClient{
		API:          client.API,
		requestDepth: client.requestDepth,
	}
}

// CreateServer creates a new server with provided properties in the specified data center.
func (c *IonosCloudClient) CreateServer(
	ctx context.Context,
	datacenterID string,
	properties sdk.ServerProperties,
	entities sdk.ServerEntities,
) (*sdk.Server, string, error) {
	if datacenterID == "" {
		return nil, "", errDatacenterIDIsEmpty
	}
	server := sdk.Server{
		Entities:   &entities,
		Properties: &properties,
	}
	s, req, err := c.API.ServersApi.DatacentersServersPost(ctx, datacenterID).Server(server).Execute()
	if err != nil {
		return nil, "", fmt.Errorf(apiCallErrWrapper, err)
	}

	location := req.Header.Get(locationHeaderKey)
	if location == "" {
		err = errLocationHeaderEmpty
	}

	return &s, location, err
}

// ListServers returns a list with servers in the specified data center.
func (c *IonosCloudClient) ListServers(ctx context.Context, datacenterID string) (*sdk.Servers, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	servers, _, err := c.API.ServersApi.
		DatacentersServersGet(ctx, datacenterID).
		Depth(c.requestDepth).
		Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &servers, nil
}

// GetServer returns the server that matches the provided serverID in the specified data center.
func (c *IonosCloudClient) GetServer(ctx context.Context, datacenterID, serverID string) (*sdk.Server, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	if serverID == "" {
		return nil, errServerIDIsEmpty
	}
	server, _, err := c.API.ServersApi.
		DatacentersServersFindById(ctx, datacenterID, serverID).
		Depth(c.requestDepth).
		Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &server, nil
}

// DeleteServer deletes the server that matches the provided serverID in the specified data center.
func (c *IonosCloudClient) DeleteServer(ctx context.Context, datacenterID, serverID string) (string, error) {
	if datacenterID == "" {
		return "", errDatacenterIDIsEmpty
	}
	if serverID == "" {
		return "", errServerIDIsEmpty
	}
	req, err := c.API.ServersApi.
		DatacentersServersDelete(ctx, datacenterID, serverID).
		DeleteVolumes(true).
		Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if location := req.Header.Get(locationHeaderKey); location != "" {
		return location, nil
	}

	return "", errLocationHeaderEmpty
}

// CreateLAN creates a new LAN with the provided properties in the specified data center, returning the request location.
func (c *IonosCloudClient) CreateLAN(ctx context.Context, datacenterID string, properties sdk.LanPropertiesPost,
) (string, error) {
	if datacenterID == "" {
		return "", errDatacenterIDIsEmpty
	}
	lanPost := sdk.LanPost{
		Properties: &properties,
	}
	_, req, err := c.API.LANsApi.DatacentersLansPost(ctx, datacenterID).Lan(lanPost).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if location := req.Header.Get(locationHeaderKey); location != "" {
		return location, nil
	}
	return "", errLocationHeaderEmpty
}

// PatchLAN patches the LAN that matches lanID in the specified data center with the provided properties, returning the request location.
func (c *IonosCloudClient) PatchLAN(ctx context.Context, datacenterID, lanID string, properties sdk.LanProperties) (string, error) {
	if datacenterID == "" {
		return "", errDatacenterIDIsEmpty
	}

	if lanID == "" {
		return "", errLANIDIsEmpty
	}

	_, res, err := c.API.LANsApi.DatacentersLansPatch(ctx, datacenterID, lanID).Lan(properties).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}

	if location := res.Header.Get(locationHeaderKey); location != "" {
		return location, nil
	}

	return "", errLocationHeaderEmpty
}

// ListLANs returns a list of LANs in the specified data center.
func (c *IonosCloudClient) ListLANs(ctx context.Context, datacenterID string) (*sdk.Lans, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	lans, _, err := c.API.LANsApi.DatacentersLansGet(ctx, datacenterID).Depth(c.requestDepth).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &lans, nil
}

// DeleteLAN deletes the LAN that matches the provided lanID in the specified data center, returning the request location.
func (c *IonosCloudClient) DeleteLAN(ctx context.Context, datacenterID, lanID string) (string, error) {
	if datacenterID == "" {
		return "", errDatacenterIDIsEmpty
	}
	if lanID == "" {
		return "", errLANIDIsEmpty
	}
	req, err := c.API.LANsApi.DatacentersLansDelete(ctx, datacenterID, lanID).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if location := req.Header.Get(locationHeaderKey); location != "" {
		return location, nil
	}
	return "", errLocationHeaderEmpty
}

// ReserveIPBlock reserves an IP block with the provided properties in the specified location, returning the request
// path.
func (c *IonosCloudClient) ReserveIPBlock(ctx context.Context, name, location string, size int32) (requestPath string, err error) {
	if location == "" {
		return "", errors.New("location must be set")
	}
	if size <= 0 {
		return "", errors.New("size must be greater than 0")
	}
	if name == "" {
		return "", errors.New("name must be set")
	}
	ipBlock := sdk.IpBlock{
		Properties: &sdk.IpBlockProperties{
			Name:     &name,
			Size:     &size,
			Location: &location,
		},
	}
	_, req, err := c.API.IPBlocksApi.IpblocksPost(ctx).Depth(c.requestDepth).Ipblock(ipBlock).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if requestPath := req.Header.Get(locationHeaderKey); requestPath != "" {
		return requestPath, nil
	}
	return "", errors.New(apiNoLocationErrMessage)
}

// GetIPBlock returns the IP block that matches the provided ipBlockID.
func (c *IonosCloudClient) GetIPBlock(ctx context.Context, ipBlockID string) (*sdk.IpBlock, error) {
	if ipBlockID == "" {
		return nil, errIPBlockIDIsEmpty
	}
	ipBlock, _, err := c.API.IPBlocksApi.IpblocksFindById(ctx, ipBlockID).Depth(c.requestDepth).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &ipBlock, nil
}

// DeleteIPBlock deletes the IP block that matches the provided ipBlockID.
func (c *IonosCloudClient) DeleteIPBlock(ctx context.Context, ipBlockID string) (requestPath string, err error) {
	if ipBlockID == "" {
		return "", errIPBlockIDIsEmpty
	}
	req, err := c.API.IPBlocksApi.IpblocksDelete(ctx, ipBlockID).Depth(c.requestDepth).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if requestPath = req.Header.Get(locationHeaderKey); requestPath != "" {
		return requestPath, nil
	}
	return "", errors.New(apiNoLocationErrMessage)
}

// ListIPBlocks returns a list of IP blocks.
func (c *IonosCloudClient) ListIPBlocks(ctx context.Context) (*sdk.IpBlocks, error) {
	blocks, _, err := c.API.IPBlocksApi.IpblocksGet(ctx).Depth(c.requestDepth).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &blocks, nil
}

// CheckRequestStatus returns the status of a request and an error if checking for it fails.
func (c *IonosCloudClient) CheckRequestStatus(ctx context.Context, requestURL string) (*sdk.RequestStatus, error) {
	if requestURL == "" {
		return nil, errRequestURLIsEmpty
	}
	requestStatus, _, err := c.API.GetRequestStatus(ctx, requestURL)
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return requestStatus, nil
}

// GetRequests returns the requests made in the last 24 hours that match the provided method and path.
func (c *IonosCloudClient) GetRequests(ctx context.Context, method, path string) ([]sdk.Request, error) {
	if path == "" {
		return nil, errors.New("path needs to be provided")
	}
	if method == "" {
		return nil, errors.New("method needs to be provided")
	}

	const lookbackTime = 24 * time.Hour
	lookback := time.Now().Add(-lookbackTime).Format(time.DateTime)

	depth := c.requestDepth
	if depth == 0 {
		depth = 2 // for LISTing requests and their metadata status metadata
	}

	reqs, _, err := c.API.RequestsApi.RequestsGet(ctx).
		Depth(depth).
		FilterMethod(method).
		FilterUrl(path).
		FilterCreatedAfter(lookback).
		Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	if reqs.Items == nil {
		// NOTE(lubedacht): This shouldn't happen, but we shouldn't deref
		// a pointer without a nil check
		return nil, nil
	}

	items := *reqs.Items
	slices.SortFunc(items, func(a, b sdk.Request) int {
		return b.Metadata.CreatedDate.Compare(a.Metadata.CreatedDate.Time)
	})

	return items, nil
}

// WaitForRequest waits for the completion of the provided request.
func (c *IonosCloudClient) WaitForRequest(ctx context.Context, requestURL string) error {
	if requestURL == "" {
		return errRequestURLIsEmpty
	}
	_, err := c.API.WaitForRequest(ctx, requestURL)
	if err != nil {
		return fmt.Errorf(apiCallErrWrapper, err)
	}
	return nil
}

// PatchNIC updates the NIC identified by nicID with the provided properties.
func (c *IonosCloudClient) PatchNIC(ctx context.Context, datacenterID, serverID, nicID string, properties sdk.NicProperties) (string, error) {
	if err := validateNICParameters(datacenterID, serverID, nicID); err != nil {
		return "", err
	}

	_, res, err := c.API.NetworkInterfacesApi.
		DatacentersServersNicsPatch(ctx, datacenterID, serverID, nicID).
		Nic(properties).
		Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}

	if location := res.Header.Get(locationHeaderKey); location != "" {
		return location, nil
	}

	return "", errLocationHeaderEmpty
}

// validateNICParameters validates the parameters for the PatchNIC and DeleteNIC methods.
func validateNICParameters(datacenterID, serverID, nicID string) (err error) {
	if datacenterID == "" {
		return errDatacenterIDIsEmpty
	}

	if serverID == "" {
		return errServerIDIsEmpty
	}

	if nicID == "" {
		return errNICIDIsEmpty
	}

	return nil
}
