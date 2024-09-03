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

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

type nlbProvisioner struct {
	svc *cloud.Service
}

// Provision is responsible for creating the Network Load Balancer.
func (n *nlbProvisioner) Provision(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	/*
		Required:
		* public LAN for incoming traffic
		* private LAN for outgoing traffic
		* control plane nodes need to be in private LAN and in public LAN
		* NLB with the target LAN and probably the private IPs

		c.API.NetworkLoadBalancersApi.DatacentersNetworkloadbalancersPost(ctx, datacenterID).NetworkLoadBalancer(sdk.NetworkLoadBalancer{
			Properties: &sdk.NetworkLoadBalancerProperties{
				Name:           nil,
				ListenerLan:    nil,
				Ips:            nil,
				TargetLan:      nil,
				LbPrivateIps:   nil,
				CentralLogging: nil,
				LoggingFormat:  nil,
			},
		}).Execute()
	*/

	requeue, err = n.svc.ReconcileLoadBalancerNetworks(ctx, lb)
	if err != nil || requeue {
		return requeue, err
	}

	// Reconcile NLB and attach it to both LANs
	return n.svc.ReconcileNLB(ctx, lb)
}

func (n *nlbProvisioner) Destroy(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	// Destroy NLB

	// Destroy LANs
	return n.svc.ReconcileLoadBalancerNetworksDeletion(ctx, lb)
}
