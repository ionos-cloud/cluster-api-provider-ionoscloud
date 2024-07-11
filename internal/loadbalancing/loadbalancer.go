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

package loadbalancing

import (
	"context"
	"fmt"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// LoadBalancer is an interface for managing the provisioning of and cleanup of various types of load balancers.
type LoadBalancer interface {
	// PrepareEnvironment is responsible for setting the preconditions for the load balancer to be created.
	PrepareEnvironment(ctx context.Context, loadBalancerScope *scope.LoadBalancer) (requeue bool, err error)
	// ProvisionLoadBalancer is responsible for creating the load balancer.
	ProvisionLoadBalancer(ctx context.Context, loadBalancerScope *scope.LoadBalancer) (requeue bool, err error)

	// DestroyLoadBalancer is responsible for deleting the load balancer.
	DestroyLoadBalancer(ctx context.Context, loadBalancerScope *scope.LoadBalancer) (requeue bool, err error)
	// CleanupResources is responsible for cleaning up any resources associated with the load balancer.
	CleanupResources(ctx context.Context, loadBalancerScope *scope.LoadBalancer) (requeue bool, err error)
}

// NewProvisioner creates a new load balancer provisioner, based on the load balancer type.
func NewProvisioner(_ ionoscloud.Client, lbType infrav1.LoadBalancerType) (LoadBalancer, error) {
	switch lbType {
	case infrav1.LoadBalancerTypeHA:
	case infrav1.LoadBalancerTypeNLB:
	case infrav1.LoadBalancerTypeExternal:
	}
	return nil, fmt.Errorf("unknown load balancer type %q", lbType)
}
