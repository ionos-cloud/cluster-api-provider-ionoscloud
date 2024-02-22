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
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	// UnknownValue is a placeholder string, used as default when we deref string pointers.
	UnknownValue = "107df78c-cee8-4902-963a-655cc6ea9865"
)

// Service offers infra resources services for IONOS Cloud machine reconciliation.
type Service struct {
	scope  *scope.MachineScope // Deprecated: pass machine scope explicitly to each method.
	logger *logr.Logger
	cloud  ionoscloud.Client
}

// NewService returns a new Service.
func NewService(ms *scope.MachineScope, cloud ionoscloud.Client) (*Service, error) {
	return &Service{
		scope: ms,
		cloud: cloud,
	}, nil
}

// datacenterID is a shortcut for getting the data center ID used by the IONOS Cloud machine.
func (s *Service) datacenterID(_ *scope.MachineScope) string {
	return s.scope.IonosMachine.Spec.DatacenterID
}

// isNotFound is a shortcut for checking if an error is a not found error.
// TODO(lubedacht) Implement unit tests.
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
