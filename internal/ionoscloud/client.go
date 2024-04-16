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
	// CreateServer creates a new server with provided properties in the specified data center.
	CreateServer(ctx context.Context, datacenterID string, properties sdk.ServerProperties, entities sdk.ServerEntities) (
		*sdk.Server, string, error)
	// ListServers returns a list with the servers in the specified data center.
	ListServers(ctx context.Context, datacenterID string) (*sdk.Servers, error)
	// GetServer returns the server that matches the provided serverID in the specified data center.
	GetServer(ctx context.Context, datacenterID, serverID string) (*sdk.Server, error)
	// DeleteServer deletes the server that matches the provided serverID in the specified data center.
	DeleteServer(ctx context.Context, datacenterID, serverID string) (string, error)
	// StartServer starts the server that matches the provided serverID in the specified data center.
	StartServer(ctx context.Context, datacenterID, serverID string) (string, error)
	// CreateLAN creates a new LAN with the provided properties in the specified data center, returning the request path.
	CreateLAN(ctx context.Context, datacenterID string, properties sdk.LanPropertiesPost) (string, error)
	// PatchLAN patches the LAN that matches lanID in the specified data center with the provided properties, returning the request location.
	PatchLAN(ctx context.Context, datacenterID, lanID string, properties sdk.LanProperties) (string, error)
	// ListLANs returns a list of LANs in the specified data center.
	ListLANs(ctx context.Context, datacenterID string) (*sdk.Lans, error)
	// DeleteLAN deletes the LAN that matches the provided lanID in the specified data center, returning the request path.
	DeleteLAN(ctx context.Context, datacenterID, lanID string) (string, error)
	// ReserveIPBlock reserves an IP block with the provided properties in the specified location.
	ReserveIPBlock(ctx context.Context, name, location string, size int32) (requestPath string, err error)
	// GetIPBlock returns the IP block that matches the provided ipBlockID.
	GetIPBlock(ctx context.Context, ipBlockID string) (*sdk.IpBlock, error)
	// ListIPBlocks returns a list of IP blocks.
	ListIPBlocks(ctx context.Context) (*sdk.IpBlocks, error)
	// DeleteIPBlock deletes the IP block that matches the provided ipBlockID.
	DeleteIPBlock(ctx context.Context, ipBlockID string) (requestPath string, err error)
	// CheckRequestStatus checks the status of a provided request identified by requestID
	CheckRequestStatus(ctx context.Context, requestID string) (*sdk.RequestStatus, error)
	// WaitForRequest waits for the completion of the provided request.
	WaitForRequest(ctx context.Context, requestURL string) error
	// GetRequests returns the requests made in the last 24 hours that match the provided method and path.
	GetRequests(ctx context.Context, method, path string) ([]sdk.Request, error)
	// PatchNIC updates the NIC identified by nicID with the provided properties, returning the request location.
	PatchNIC(ctx context.Context, datacenterID, serverID, nicID string, properties sdk.NicProperties) (string, error)
}
