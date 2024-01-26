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
	depthRequestsMetadataStatusMetadata = 2 // for LISTing requests and their metadata status metadata
	depthLANEntities                    = 2 // for LISTing LANs and their NICs (w/o NIC details)

	locationHeaderKey = "Location"
)

// IonosCloudClient is a concrete implementation of the Client interface defined in the internal client package that
// communicates with Cloud API using its SDK.
type IonosCloudClient struct {
	API *sdk.APIClient
}

var _ ionoscloud.Client = &IonosCloudClient{}

// NewClient instantiates a usable IonosCloudClient. The client needs the username AND password OR the token to work.
// Failing to provide both will result in an error.
func NewClient(username, password, token, apiURL string) (*IonosCloudClient, error) {
	if err := validate(username, password, token); err != nil {
		return nil, fmt.Errorf("incomplete credentials provided: %w", err)
	}
	cfg := sdk.NewConfiguration(username, password, token, apiURL)
	apiClient := sdk.NewAPIClient(cfg)
	return &IonosCloudClient{
		API: apiClient,
	}, nil
}

func validate(username, password, token string) error {
	if username != "" && password == "" {
		return errors.New("username is set but password is not")
	}
	if username == "" && password != "" {
		return errors.New("password is set but username is not")
	}
	if username == "" && password == "" && token == "" {
		return errors.New("token or username and password need to be set")
	}
	return nil
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
		err = errors.New(apiNoLocationErrMessage)
	}

	return &s, location, err
}

// ListServers returns a list with servers in the specified data center.
func (c *IonosCloudClient) ListServers(ctx context.Context, datacenterID string) (*sdk.Servers, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	servers, _, err := c.API.ServersApi.DatacentersServersGet(ctx, datacenterID).Execute()
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
	server, _, err := c.API.ServersApi.DatacentersServersFindById(ctx, datacenterID, serverID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &server, nil
}

// DestroyServer deletes the server that matches the provided serverID in the specified data center.
func (c *IonosCloudClient) DestroyServer(ctx context.Context, datacenterID, serverID string) error {
	if datacenterID == "" {
		return errDatacenterIDIsEmpty
	}
	if serverID == "" {
		return errServerIDIsEmpty
	}
	_, err := c.API.ServersApi.DatacentersServersDelete(ctx, datacenterID, serverID).Execute()
	if err != nil {
		return fmt.Errorf(apiCallErrWrapper, err)
	}
	return err
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
	return "", errors.New(apiNoLocationErrMessage)
}

// AttachToLAN attaches a provided NIC to a provided LAN in the specified data center.
func (c *IonosCloudClient) AttachToLAN(ctx context.Context, datacenterID, lanID string, nic sdk.Nic,
) (*sdk.Nic, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLANIDIsEmpty
	}
	n, _, err := c.API.LANsApi.DatacentersLansNicsPost(ctx, datacenterID, lanID).Nic(nic).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &n, nil
}

// ListLANs returns a list of LANs in the specified data center.
func (c *IonosCloudClient) ListLANs(ctx context.Context, datacenterID string) (*sdk.Lans, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	lans, _, err := c.API.LANsApi.DatacentersLansGet(ctx, datacenterID).Depth(depthLANEntities).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &lans, nil
}

// GetLAN returns the LAN that matches lanID in the specified data center.
func (c *IonosCloudClient) GetLAN(ctx context.Context, datacenterID, lanID string) (*sdk.Lan, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLANIDIsEmpty
	}
	lan, _, err := c.API.LANsApi.DatacentersLansFindById(ctx, datacenterID, lanID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &lan, nil
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
	return "", errors.New(apiNoLocationErrMessage)
}

// ListVolumes returns a list of volumes in the specified data center.
func (c *IonosCloudClient) ListVolumes(ctx context.Context, datacenterID string,
) (*sdk.Volumes, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	volumes, _, err := c.API.VolumesApi.DatacentersVolumesGet(ctx, datacenterID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &volumes, nil
}

// GetVolume returns the volume that matches volumeID in the specified data center.
func (c *IonosCloudClient) GetVolume(ctx context.Context, datacenterID, volumeID string,
) (*sdk.Volume, error) {
	if datacenterID == "" {
		return nil, errDatacenterIDIsEmpty
	}
	if volumeID == "" {
		return nil, errVolumeIDIsEmpty
	}
	volume, _, err := c.API.VolumesApi.DatacentersVolumesFindById(ctx, datacenterID, volumeID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &volume, nil
}

// DestroyVolume deletes the volume that matches volumeID in the specified data center.
func (c *IonosCloudClient) DestroyVolume(ctx context.Context, datacenterID, volumeID string) error {
	if datacenterID == "" {
		return errDatacenterIDIsEmpty
	}
	if volumeID == "" {
		return errVolumeIDIsEmpty
	}
	_, err := c.API.VolumesApi.DatacentersVolumesDelete(ctx, datacenterID, volumeID).Execute()
	if err != nil {
		return fmt.Errorf(apiCallErrWrapper, err)
	}
	return nil
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
	reqs, _, err := c.API.RequestsApi.RequestsGet(ctx).
		Depth(depthRequestsMetadataStatusMetadata).
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
