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

package scope

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/client-go/util/retry"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
)

// LoadBalancerParams is a struct that contains the params used to create a new Machine through NewMachine.
type LoadBalancerParams struct {
	Client       client.Client
	LoadBalancer *infrav1.IonosCloudLoadBalancer

	ClusterScope *Cluster
	Locker       *locker.Locker
}

// LoadBalancer defines a basic loadbalancer context for primary use in IonosCloudLoadBalancerReconciler.
type LoadBalancer struct {
	client      client.Client
	patchHelper *patch.Helper
	Locker      *locker.Locker

	LoadBalancer *infrav1.IonosCloudLoadBalancer

	ClusterScope *Cluster
}

// NewLoadBalancer creates a new LoadBalancer scope.
func NewLoadBalancer(params LoadBalancerParams) (*LoadBalancer, error) {
	if params.Client == nil {
		return nil, errors.New("load balancer scope params lack a client")
	}
	if params.LoadBalancer == nil {
		return nil, errors.New("load balancer scope params lack an IONOS Cloud load balancer")
	}
	if params.ClusterScope == nil {
		return nil, errors.New("load balancer scope params need a IONOS Cloud cluster scope")
	}
	if params.Locker == nil {
		return nil, errors.New("load balancer scope params need a locker")
	}

	helper, err := patch.NewHelper(params.LoadBalancer, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	return &LoadBalancer{
		client:       params.Client,
		patchHelper:  helper,
		Locker:       params.Locker,
		LoadBalancer: params.LoadBalancer,
		ClusterScope: params.ClusterScope,
	}, nil
}

// PatchObject will apply all changes from the IonosCloudLoadBalancer.
// It will also make sure to patch the status subresource.
func (m *LoadBalancer) PatchObject() error {
	conditions.SetSummary(m.LoadBalancer,
		conditions.WithConditions(
			infrav1.IonosCloudLoadBalancerReady))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// We don't accept and forward a context here. This is on purpose: Even if a reconciliation is
	// aborted, we want to make sure that the final patch is applied. Reusing the context from the reconciliation
	// would cause the patch to be aborted as well.
	return m.patchHelper.Patch(
		timeoutCtx,
		m.LoadBalancer,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		}})
}

// Finalize will make sure to apply a patch to the current IonosCloudLoadBalancer.
// It also implements a retry mechanism to increase the chance of success
// in case the patch operation was not successful.
func (m *LoadBalancer) Finalize() error {
	// NOTE(lubedacht) retry is only a way to reduce the failure chance,
	// but in general, the reconciliation logic must be resilient
	// to handle an outdated resource from that API server.
	shouldRetry := func(error) bool { return true }
	return retry.OnError(
		retry.DefaultBackoff,
		shouldRetry,
		m.PatchObject)
}
