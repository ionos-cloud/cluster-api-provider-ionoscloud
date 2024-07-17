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

// Package loadbalancing provides the load balancer provisioner interface and its implementations.
// The provisioner is responsible for managing the provisioning of and cleanup of various types of load balancers.
package loadbalancing

import (
	"context"
	"fmt"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// Provisioner is an interface for managing the provisioning and cleanup of various types of load balancers.
type Provisioner interface {
	// Provision is responsible for creating the load balancer.
	Provision(ctx context.Context, loadBalancerScope *scope.LoadBalancer) (requeue bool, err error)

	// Destroy is responsible for deleting the load balancer.
	Destroy(ctx context.Context, loadBalancerScope *scope.LoadBalancer) (requeue bool, err error)
}

// NewProvisioner creates a new load balancer provisioner, based on the load balancer type.
func NewProvisioner(_ *cloud.Service, source infrav1.LoadBalancerSource) (Provisioner, error) {
	switch {
	case source.KubeVIP != nil:
		return &kubeVIPProvisioner{}, nil
	case source.NLB != nil:
		return &nlbProvisioner{}, nil
	}
	return nil, fmt.Errorf("unknown load balancer config %#v", source)
}
