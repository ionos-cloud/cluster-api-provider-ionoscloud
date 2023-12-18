/*
Copyright 2023 IONOS Cloud.

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

// Package client contains an implementation of the IonosCloudClient interface defined in internal/ionoscloud.
package client

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
)

// IonosCloudClient is a concrete implementation of the Client interface defined in the internal client package, that
// communicates with Cloud API using its SDK.
type IonosCloudClient struct {
	API *sdk.APIClient
}

var _ ionoscloud.Client = &IonosCloudClient{}

// NewClient instantiates a usable Client. The client needs the username AND password OR the token to work.
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

// CreateDataCenter creates a new data center with its specification based on provided properties.
func (c *IonosCloudClient) CreateDataCenter(ctx context.Context, properties sdk.DatacenterProperties,
) (*sdk.Datacenter, error) {
	dc := sdk.Datacenter{Properties: &properties}
	dc, _, err := c.API.DataCentersApi.DatacentersPost(ctx).Datacenter(dc).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &dc, nil
}

// GetDataCenter returns the data center that matches the provided datacenterID.
func (c *IonosCloudClient) GetDataCenter(ctx context.Context, id string) (*sdk.Datacenter, error) {
	if id == "" {
		return nil, errDataCenterIDIsEmpty
	}
	dc, _, err := c.API.DataCentersApi.DatacentersFindById(ctx, id).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &dc, nil
}

// CreateServer creates a new server with provided properties in the specified data center.
func (c *IonosCloudClient) CreateServer(
	ctx context.Context, dataCenterID string, properties sdk.ServerProperties,
) (*sdk.Server, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	server := sdk.Server{
		Properties: &properties,
	}
	s, _, err := c.API.ServersApi.DatacentersServersPost(ctx, dataCenterID).Server(server).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &s, nil
}

// ListServers returns a list with servers in the specified data center.
func (c *IonosCloudClient) ListServers(ctx context.Context, dataCenterID string) (*sdk.Servers, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	servers, _, err := c.API.ServersApi.DatacentersServersGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &servers, nil
}

// GetServer returns the server that matches the provided serverID in the specified data center.
func (c *IonosCloudClient) GetServer(ctx context.Context, dataCenterID, serverID string) (*sdk.Server, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if serverID == "" {
		return nil, errServerIDIsEmpty
	}
	server, _, err := c.API.ServersApi.DatacentersServersFindById(ctx, dataCenterID, serverID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &server, nil
}

// DestroyServer deletes the server that matches the provided serverID in the specified data center.
func (c *IonosCloudClient) DestroyServer(ctx context.Context, dataCenterID, serverID string) error {
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if serverID == "" {
		return errServerIDIsEmpty
	}
	_, err := c.API.ServersApi.DatacentersServersDelete(ctx, dataCenterID, serverID).Execute()
	if err != nil {
		return fmt.Errorf(apiCallErrWrapper, err)
	}
	return err
}

// CreateLAN creates a new LAN with the provided properties in the specified data center.
func (c *IonosCloudClient) CreateLAN(ctx context.Context, dataCenterID string, properties sdk.LanPropertiesPost,
) (*sdk.LanPost, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	lanPost := sdk.LanPost{
		Properties: &properties,
	}
	lp, _, err := c.API.LANsApi.DatacentersLansPost(ctx, dataCenterID).Lan(lanPost).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &lp, nil
}

// UpdateLAN updates a LAN with the provided properties in the specified data center.
func (c *IonosCloudClient) UpdateLAN(
	ctx context.Context, dataCenterID string, lanID string, properties sdk.LanProperties,
) (*sdk.Lan, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	l, _, err := c.API.LANsApi.DatacentersLansPatch(ctx, dataCenterID, lanID).
		Lan(properties).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &l, nil
}

// AttachToLAN attaches a provided NIC to a provided LAN in the specified data center.
func (c *IonosCloudClient) AttachToLAN(ctx context.Context, dataCenterID, lanID string, nic sdk.Nic,
) (*sdk.Nic, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	n, _, err := c.API.LANsApi.DatacentersLansNicsPost(ctx, dataCenterID, lanID).Nic(nic).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &n, nil
}

// ListLANs returns a list of LANs in the specified data center.
func (c *IonosCloudClient) ListLANs(ctx context.Context, dataCenterID string) (*sdk.Lans, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	lans, _, err := c.API.LANsApi.DatacentersLansGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &lans, nil
}

// GetLAN returns the LAN that matches lanID in the specified data center.
func (c *IonosCloudClient) GetLAN(ctx context.Context, dataCenterID, lanID string) (*sdk.Lan, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	lan, _, err := c.API.LANsApi.DatacentersLansFindById(ctx, dataCenterID, lanID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &lan, nil
}

// DestroyLAN deletes the LAN that matches the provided lanID in the specified data center.
func (c *IonosCloudClient) DestroyLAN(ctx context.Context, dataCenterID, lanID string) error {
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return errLanIDIsEmpty
	}
	_, err := c.API.LANsApi.DatacentersLansDelete(ctx, dataCenterID, lanID).Execute()
	if err != nil {
		return fmt.Errorf(apiCallErrWrapper, err)
	}
	return nil
}

// ListVolumes returns a list of volumes in the specified data center.
func (c *IonosCloudClient) ListVolumes(ctx context.Context, dataCenterID string,
) (*sdk.Volumes, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	volumes, _, err := c.API.VolumesApi.DatacentersVolumesGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &volumes, nil
}

// GetVolume returns the volume that matches volumeID in the specified data center.
func (c *IonosCloudClient) GetVolume(ctx context.Context, dataCenterID, volumeID string,
) (*sdk.Volume, error) {
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if volumeID == "" {
		return nil, errVolumeIDIsEmpty
	}
	volume, _, err := c.API.VolumesApi.DatacentersVolumesFindById(ctx, dataCenterID, volumeID).Execute()
	if err != nil {
		return nil, fmt.Errorf(apiCallErrWrapper, err)
	}
	return &volume, nil
}

// DestroyVolume deletes the volume that matches volumeID in the specified data center.
func (c *IonosCloudClient) DestroyVolume(ctx context.Context, dataCenterID, volumeID string) error {
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if volumeID == "" {
		return errVolumeIDIsEmpty
	}
	_, err := c.API.VolumesApi.DatacentersVolumesDelete(ctx, dataCenterID, volumeID).Execute()
	if err != nil {
		return fmt.Errorf(apiCallErrWrapper, err)
	}
	return nil
}
