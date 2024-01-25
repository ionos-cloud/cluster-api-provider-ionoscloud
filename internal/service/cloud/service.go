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
	"context"
	"errors"
	"net/http"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// Service offers infra resources services for IONOS Cloud machine reconciliation.
type Service struct {
	scope *scope.MachineScope
	ctx   context.Context
}

// NewService returns a new Service.
func NewService(ctx context.Context, s *scope.MachineScope) (*Service, error) {
	if s == nil {
		return nil, errors.New("cloud service cannot use a nil machine scope")
	}
	return &Service{
		scope: s,
		ctx:   ctx,
	}, nil
}

// api is a shortcut for the IONOS Cloud Client.
func (s *Service) api() ionoscloud.Client {
	return s.scope.ClusterScope.IonosClient
}

// datacenterID is a shortcut for getting the data center ID used by the IONOS Cloud machine.
func (s *Service) datacenterID() string {
	return s.scope.IonosMachine.Spec.DatacenterID
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
