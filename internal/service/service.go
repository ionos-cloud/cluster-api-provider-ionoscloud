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

// Package service offers infra resources services for IONOS Cloud machine reconciliation.
package service

import (
	"context"
	"errors"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// MachineService offers infra resources services for IONOS Cloud machine reconciliation.
type MachineService struct {
	scope *scope.MachineScope
	ctx   context.Context
}

// NewMachineService returns a new MachineService.
func NewMachineService(ctx context.Context, s *scope.MachineScope) (*MachineService, error) {
	if s == nil {
		return nil, errors.New("machine service cannot use a nil machine scope")
	}
	return &MachineService{
		scope: s,
		ctx:   ctx,
	}, nil
}

// API is a shortcut for the IONOS Cloud Client.
func (s *MachineService) API() ionoscloud.Client {
	return s.scope.ClusterScope.IonosClient
}
