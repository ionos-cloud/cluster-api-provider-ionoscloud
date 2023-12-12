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

// Package ionoscloud contains an implementation of the Client interface defined in the internal client package.
package ionoscloud

import (
	"context"
	"os"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/client"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrlRuntime "sigs.k8s.io/controller-runtime/pkg/client"
)

// Client is a concrete implementation of the Client interface defined in the internal client package, that
// communicates with Cloud API using its sdk.
type Client struct {
	api *ionoscloud.APIClient
}

// NewClient instantiates a usable IonosCloudClient. If `r` and `secretRef` are nil, IonosCloudClient will
// load the sdk configuration from env, otherwise it will fetch the values from a provided secret. If the secret does
// not exist, an error will be thrown.
func NewClient(ctx context.Context, r ctrlRuntime.Client, secretRef *corev1.SecretReference,
) (client.Client, error) {
	var cfg *ionoscloud.Configuration
	if secretRef == nil {
		cfg = ionoscloud.NewConfigurationFromEnv()
	} else {
		secret := &corev1.Secret{}
		objKey := types.NamespacedName{Namespace: secretRef.Namespace, Name: secretRef.Name}
		err := r.Get(ctx, objKey, secret)
		if err != nil {
			return nil, err
		}
		username := string(secret.Data["username"])
		password := string(secret.Data["password"])
		token := string(secret.Data["token"])
		cfg = ionoscloud.NewConfiguration(username, password, token, os.Getenv(ionoscloud.IonosApiUrlEnvVar))
	}
	apiClient := ionoscloud.NewAPIClient(cfg)
	return &Client{
		api: apiClient,
	}, nil
}

// CreateDataCenter creates a new data center with its specification based on provided properties.
func (c *Client) CreateDataCenter(ctx context.Context, properties ionoscloud.DatacenterProperties,
) (*ionoscloud.Datacenter, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	dc := ionoscloud.Datacenter{Properties: &properties}
	dc, _, err := c.api.DataCentersApi.DatacentersPost(ctx).Datacenter(dc).Execute()
	if err != nil {
		return nil, err
	}
	return &dc, nil
}

// GetDataCenter returns a data center that contains the provided id.
func (c *Client) GetDataCenter(ctx context.Context, id string) (*ionoscloud.Datacenter, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if id == "" {
		return nil, errDataCenterIDIsEmpty
	}
	dc, _, err := c.api.DataCentersApi.DatacentersFindById(ctx, id).Execute()
	if err != nil {
		return nil, err
	}
	return &dc, nil
}

// CreateServer creates a new server with provided properties in the specified data center.
func (c *Client) CreateServer(
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
	s, _, err := c.api.ServersApi.DatacentersServersPost(ctx, dataCenterID).Server(server).Execute()
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListServers returns a list with the created servers in the specified data center.
func (c *Client) ListServers(ctx context.Context, dataCenterID string) (*ionoscloud.Servers, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	servers, _, err := c.api.ServersApi.DatacentersServersGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, err
	}
	return &servers, nil
}

// GetServer returns the server that matches the provided serverID in the specified data center.
func (c *Client) GetServer(ctx context.Context, dataCenterID, serverID string) (*ionoscloud.Server, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if serverID == "" {
		return nil, errServerIDIsEmpty
	}
	server, _, err := c.api.ServersApi.DatacentersServersFindById(ctx, dataCenterID, serverID).Execute()
	if err != nil {
		return nil, err
	}
	return &server, nil
}

// DestroyServer deletes the server that matches the provided serverID in the specified data center.
func (c *Client) DestroyServer(ctx context.Context, dataCenterID, serverID string) error {
	if ctx == nil {
		return errContextIsNil
	}
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if serverID == "" {
		return errServerIDIsEmpty
	}
	_, err := c.api.ServersApi.DatacentersServersDelete(ctx, dataCenterID, serverID).Execute()
	return err
}

// CreateLan creates a new LAN with the provided properties in the specified data center.
func (c *Client) CreateLan(ctx context.Context, dataCenterID string, properties ionoscloud.LanPropertiesPost,
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
	lp, _, err := c.api.LANsApi.DatacentersLansPost(ctx, dataCenterID).Lan(lanPost).Execute()
	if err != nil {
		return nil, err
	}
	return &lp, nil
}

// UpdateLan updates a LAN with the provided properties in the specified data center.
func (c *Client) UpdateLan(
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
	l, _, err := c.api.LANsApi.DatacentersLansPatch(ctx, dataCenterID, lanID).
		Lan(properties).Execute()
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// AttachToLan attaches a provided NIC to a provided LAN in a specified data center.
func (c *Client) AttachToLan(ctx context.Context, dataCenterID, lanID string, nic ionoscloud.Nic,
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
	n, _, err := c.api.LANsApi.DatacentersLansNicsPost(ctx, dataCenterID, lanID).Nic(nic).Execute()
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// ListLans returns a list of LANs in the specified data center.
func (c *Client) ListLans(ctx context.Context, dataCenterID string) (*ionoscloud.Lans, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	lans, _, err := c.api.LANsApi.DatacentersLansGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, err
	}
	return &lans, nil
}

// GetLan returns the LAN that matches lanID in the specified data center.
func (c *Client) GetLan(ctx context.Context, dataCenterID, lanID string) (*ionoscloud.Lan, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	if lanID == "" {
		return nil, errLanIDIsEmpty
	}
	lan, _, err := c.api.LANsApi.DatacentersLansFindById(ctx, dataCenterID, lanID).Execute()
	if err != nil {
		return nil, err
	}
	return &lan, nil
}

// ListVolumes returns a list of volumes in a specified data center.
func (c *Client) ListVolumes(ctx context.Context, dataCenterID string,
) (*ionoscloud.Volumes, error) {
	if ctx == nil {
		return nil, errContextIsNil
	}
	if dataCenterID == "" {
		return nil, errDataCenterIDIsEmpty
	}
	volumes, _, err := c.api.VolumesApi.DatacentersVolumesGet(ctx, dataCenterID).Execute()
	if err != nil {
		return nil, err
	}
	return &volumes, nil
}

// GetVolume returns the volume that matches volumeID in the specified data center.
func (c *Client) GetVolume(ctx context.Context, dataCenterID, volumeID string,
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
	volume, _, err := c.api.VolumesApi.DatacentersVolumesFindById(ctx, dataCenterID, volumeID).Execute()
	if err != nil {
		return nil, err
	}
	return &volume, nil
}

// DestroyVolume deletes the volume that matches volumeID in the specified data center.
func (c *Client) DestroyVolume(ctx context.Context, dataCenterID, volumeID string) error {
	if ctx == nil {
		return errContextIsNil
	}
	if dataCenterID == "" {
		return errDataCenterIDIsEmpty
	}
	if volumeID == "" {
		return errVolumeIDIsEmpty
	}
	_, err := c.api.VolumesApi.DatacentersVolumesDelete(ctx, dataCenterID, volumeID).Execute()
	if err != nil {
		return err
	}
	return nil
}
