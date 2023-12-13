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

// Package ionosclient contains an implementation of the IonosClient interface defined in the internal client package.
package ionosclient

import (
	"context"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/pkg/ionos"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
)

// IonosClient is a concrete implementation of the IonosClient interface defined in the internal client package, that
// communicates with Cloud API using its sdk.
type IonosClient struct {
	API *ionoscloud.APIClient
}

// NewClient instantiates a usable IonosClient. The client needs the username AND password OR the token to work.
// Failing to provide both will result in an error.
func NewClient(username, password, token, apiURL string) (ionos.Client, error) {
	if token != "" || username != "" && password != "" {
		cfg := ionoscloud.NewConfiguration(username, password, token, apiURL)
		apiClient := ionoscloud.NewAPIClient(cfg)
		return &IonosClient{
			API: apiClient,
		}, nil
	}
	return nil, errors.New("credentials are not valid")
}

// CreateDataCenter creates a new data center with its specification based on provided properties.
func (c *IonosClient) CreateDataCenter(ctx context.Context, properties ionoscloud.DatacenterProperties,
) (*ionoscloud.Datacenter, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	dc := ionoscloud.Datacenter{Properties: &properties}
	dc, _, err := c.API.DataCentersApi.DatacentersPost(ctx).Datacenter(dc).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &dc, nil
}

// GetDataCenter returns a data center that contains the provided id.
func (c *IonosClient) GetDataCenter(ctx context.Context, id string) (*ionoscloud.Datacenter, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if id == "" {
		return nil, errDataCenterIDIsEmpty
	}
	dc, _, err := c.API.DataCentersApi.DatacentersFindById(ctx, id).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &dc, nil
}

// CreateServer creates a new server with provided properties in the specified data center.
func (c *IonosClient) CreateServer(
	ctx context.Context, dataCenterID string, properties ionoscloud.ServerProperties,
) (*ionoscloud.Server, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	server := ionoscloud.Server{
		Properties: &properties,
	}
	s, _, err := c.API.ServersApi.DatacentersServersPost(ctx, dataCenterID).Server(server).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &s, nil
}

// ListServers returns a list with the created servers in the specified data center.
func (c *IonosClient) ListServers(ctx context.Context, dataCenterID string) (*ionoscloud.Servers, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	servers, _, err := c.API.ServersApi.DatacentersServersGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &servers, nil
}

// GetServer returns the server that matches the provided serverID in the specified data center.
func (c *IonosClient) GetServer(ctx context.Context, dataCenterID, serverID string) (*ionoscloud.Server, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if serverID == "" {
		return nil, errServerIDIsEmpty
	}
	server, _, err := c.API.ServersApi.DatacentersServersFindById(ctx, dataCenterID, serverID).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &server, nil
}

// DestroyServer deletes the server that matches the provided serverID in the specified data center.
func (c *IonosClient) DestroyServer(ctx context.Context, dataCenterID, serverID string) error {
	if ctx == nil {
		return errContextIsNil
	}
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if serverID == "" {
		return errServerIDIsEmpty
	}
	_, err := c.API.ServersApi.DatacentersServersDelete(ctx, dataCenterID, serverID).Execute()
	if err != nil {
		return errors.Wrap(err, apiCallErrWrapper)
	}
	return err
}

// CreateLan creates a new LAN with the provided properties in the specified data center.
func (c *IonosClient) CreateLan(ctx context.Context, dataCenterID string, properties ionoscloud.LanPropertiesPost,
) (*ionoscloud.LanPost, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	lanPost := ionoscloud.LanPost{
		Properties: &properties,
	}
	lp, _, err := c.API.LANsApi.DatacentersLansPost(ctx, dataCenterID).Lan(lanPost).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &lp, nil
}

// UpdateLan updates a LAN with the provided properties in the specified data center.
func (c *IonosClient) UpdateLan(
	ctx context.Context, dataCenterID string, lanID string, properties ionoscloud.LanProperties,
) (*ionoscloud.Lan, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	l, _, err := c.API.LANsApi.DatacentersLansPatch(ctx, dataCenterID, lanID).
		Lan(properties).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &l, nil
}

// AttachToLan attaches a provided NIC to a provided LAN in a specified data center.
func (c *IonosClient) AttachToLan(ctx context.Context, dataCenterID, lanID string, nic ionoscloud.Nic,
) (*ionoscloud.Nic, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	n, _, err := c.API.LANsApi.DatacentersLansNicsPost(ctx, dataCenterID, lanID).Nic(nic).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &n, nil
}

// ListLans returns a list of LANs in the specified data center.
func (c *IonosClient) ListLans(ctx context.Context, dataCenterID string) (*ionoscloud.Lans, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	lans, _, err := c.API.LANsApi.DatacentersLansGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &lans, nil
}

// GetLan returns the LAN that matches lanID in the specified data center.
func (c *IonosClient) GetLan(ctx context.Context, dataCenterID, lanID string) (*ionoscloud.Lan, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	lan, _, err := c.API.LANsApi.DatacentersLansFindById(ctx, dataCenterID, lanID).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &lan, nil
}

// ListVolumes returns a list of volumes in a specified data center.
func (c *IonosClient) ListVolumes(ctx context.Context, dataCenterID string,
) (*ionoscloud.Volumes, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	volumes, _, err := c.API.VolumesApi.DatacentersVolumesGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &volumes, nil
}

// GetVolume returns the volume that matches volumeID in the specified data center.
func (c *IonosClient) GetVolume(ctx context.Context, dataCenterID, volumeID string,
) (*ionoscloud.Volume, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if volumeID == "" {
		return nil, errVolumeIDIsEmpty
	}
	volume, _, err := c.API.VolumesApi.DatacentersVolumesFindById(ctx, dataCenterID, volumeID).Execute()
	if err != nil {
		return nil, errors.Wrap(err, apiCallErrWrapper)
	}
	return &volume, nil
}

// DestroyVolume deletes the volume that matches volumeID in the specified data center.
func (c *IonosClient) DestroyVolume(ctx context.Context, dataCenterID, volumeID string) error {
	if ctx == nil {
		return errContextIsNil
	}
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if volumeID == "" {
		return errVolumeIDIsEmpty
	}
	_, err := c.API.VolumesApi.DatacentersVolumesDelete(ctx, dataCenterID, volumeID).Execute()
	if err != nil {
		return errors.Wrap(err, apiCallErrWrapper)
	}
	return nil
}
