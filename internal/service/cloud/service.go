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

// Package cloud offers infra resources services for IONOS Cloud machine reconciliation.
package cloud

import (
	"errors"
	"net/http"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/client"
)

const (
	// unknownValue is a placeholder string, used as defaults when we deref string pointers.
	unknownValue = "UNKNOWN"
)

// Service offers infra resources services for IONOS Cloud machine reconciliation.
type Service struct {
	logger *logr.Logger
	cloud  ionoscloud.Client
}

// NewService returns a new Service.
func NewService(cloud ionoscloud.Client, logger *logr.Logger) (*Service, error) {
	return &Service{
		logger: logger,
		cloud:  cloud,
	}, nil
}

// apiWithDepth is a shortcut for the IONOS Cloud Client with a specific depth.
// It will create a copy of the client with the depth set to the provided value.
func (s *Service) apiWithDepth(depth int32) ionoscloud.Client {
	return client.WithDepth(s.cloud, depth)
}

// isNotFound is a shortcut for checking if an error is a not found error.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	var target sdk.GenericOpenAPIError
	if errors.As(err, &target) {
		return target.StatusCode() == http.StatusNotFound
	}

	return false
}

// ignoreNotFound is a shortcut for ignoring not found errors.
func ignoreNotFound(err error) error {
	if isNotFound(err) {
		return nil
	}
	return err
}
