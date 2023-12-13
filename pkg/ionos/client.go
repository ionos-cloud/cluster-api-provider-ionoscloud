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

// Package ionos offers an interface for abstracting Cloud API sdk, making it testable.
package ionos

import (
	"context"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
)

// Client is an interface for abstracting Cloud API sdk, making it possible to create mocks for testing purposes.
type Client interface {
	// CreateDataCenter creates a new data center with its specification based on provided properties.
	CreateDataCenter(ctx context.Context, properties ionoscloud.DatacenterProperties) (
		*ionoscloud.Datacenter, error)
	// GetDataCenter returns a data center that contains the provided id.
	GetDataCenter(ctx context.Context, id string) (*ionoscloud.Datacenter, error)
	// CreateServer creates a new server with provided properties in the specified data center.
	CreateServer(ctx context.Context, dataCenterID string, properties ionoscloud.ServerProperties) (
		*ionoscloud.Server, error)
	// ListServers returns a list with the created servers in the specified data center.
	ListServers(ctx context.Context, dataCenterID string) (*ionoscloud.Servers, error)
	// GetServer returns the server that matches the provided serverID in the specified data center.
	GetServer(ctx context.Context, dataCenterID, serverID string) (*ionoscloud.Server, error)
	// DestroyServer deletes the server that matches the provided serverID in the specified data center.
	DestroyServer(ctx context.Context, dataCenterID, serverID string) error
	// CreateLan creates a new LAN with the provided properties in the specified data center.
	CreateLan(ctx context.Context, dataCenterID string, properties ionoscloud.LanPropertiesPost) (
		*ionoscloud.LanPost, error)
	// UpdateLan updates a LAN with the provided properties in the specified data center.
	UpdateLan(ctx context.Context, dataCenterID string, lanID string, properties ionoscloud.LanProperties) (
		*ionoscloud.Lan, error)
	// AttachToLan attaches a provided NIC to a provided LAN in a specified data center.
	AttachToLan(ctx context.Context, dataCenterID, lanID string, nic ionoscloud.Nic) (
		*ionoscloud.Nic, error)
	// ListLans returns a list of LANs in the specified data center.
	ListLans(ctx context.Context, dataCenterID string) (*ionoscloud.Lans, error)
	// GetLan returns the LAN that matches lanID in the specified data center.
	GetLan(ctx context.Context, dataCenterID, lanID string) (*ionoscloud.Lan, error)
	// ListVolumes returns a list of volumes in a specified data center.
	ListVolumes(ctx context.Context, dataCenterID string) (*ionoscloud.Volumes, error)
	// GetVolume returns the volume that matches volumeID in the specified data center.
	GetVolume(ctx context.Context, dataCenterID, volumeID string) (*ionoscloud.Volume, error)
	// DestroyVolume deletes the volume that matches volumeID in the specified data center.
	DestroyVolume(ctx context.Context, dataCenterID, volumeID string) error
}
