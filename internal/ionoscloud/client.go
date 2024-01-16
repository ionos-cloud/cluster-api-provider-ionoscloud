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

// Package ionoscloud offers an interface for abstracting Cloud API SDK, making it testable.
package ionoscloud

import (
	"context"

	sdk "github.com/ionos-cloud/sdk-go/v6"
)

// Client is an interface for abstracting Cloud API SDK, making it possible to create mocks for testing purposes.
type Client interface {
	// CreateDataCenter creates a new data center with its specification based on provided properties.
	CreateDataCenter(ctx context.Context, properties sdk.DatacenterProperties) (
		*sdk.Datacenter, error)
	// GetDataCenter returns the data center that matches the provided data center ID.
	GetDataCenter(ctx context.Context, dataCenterID string) (*sdk.Datacenter, error)
	// CreateServer creates a new server with provided properties in the specified data center.
	CreateServer(ctx context.Context, dataCenterID string, properties sdk.ServerProperties) (
		*sdk.Server, error)
	// ListServers returns a list with the servers in the specified data center.
	ListServers(ctx context.Context, dataCenterID string) (*sdk.Servers, error)
	// GetServer returns the server that matches the provided serverID in the specified data center.
	GetServer(ctx context.Context, dataCenterID, serverID string) (*sdk.Server, error)
	// DestroyServer deletes the server that matches the provided serverID in the specified data center.
	DestroyServer(ctx context.Context, dataCenterID, serverID string) error
	// CreateLAN creates a new LAN with the provided properties in the specified data center, returning the request location.
	CreateLAN(ctx context.Context, dataCenterID string, properties sdk.LanPropertiesPost) (string, error)
	// UpdateLAN updates a LAN with the provided properties in the specified data center.
	UpdateLAN(ctx context.Context, dataCenterID string, lanID string, properties sdk.LanProperties) (
		*sdk.Lan, error)
	// AttachToLAN attaches a provided NIC to a provided LAN in a specified data center.
	AttachToLAN(ctx context.Context, dataCenterID, lanID string, nic sdk.Nic) (
		*sdk.Nic, error)
	// ListLANs returns a list of LANs in the specified data center.
	ListLANs(ctx context.Context, dataCenterID string) (*sdk.Lans, error)
	// GetLAN returns the LAN that matches lanID in the specified data center.
	GetLAN(ctx context.Context, dataCenterID, lanID string) (*sdk.Lan, error)
	// DeleteLAN deletes the LAN that matches the provided lanID in the specified data center, returning the request location.
	DeleteLAN(ctx context.Context, dataCenterID, lanID string) (string, error)
	// ListVolumes returns a list of volumes in a specified data center.
	ListVolumes(ctx context.Context, dataCenterID string) (*sdk.Volumes, error)
	// GetVolume returns the volume that matches volumeID in the specified data center.
	GetVolume(ctx context.Context, dataCenterID, volumeID string) (*sdk.Volume, error)
	// DestroyVolume deletes the volume that matches volumeID in the specified data center.
	DestroyVolume(ctx context.Context, dataCenterID, volumeID string) error
	// CheckRequestStatus checks the status of a provided request identified by requestID
	CheckRequestStatus(ctx context.Context, requestID string) (*sdk.RequestStatus, error)
	// WaitForRequest waits for the completion of the provided request.
	WaitForRequest(ctx context.Context, requestURL string) error
	// GetRequests returns the requests made in the last 24 hours that match the provided method and path.
	GetRequests(ctx context.Context, method, path string) ([]sdk.Request, error)
}
