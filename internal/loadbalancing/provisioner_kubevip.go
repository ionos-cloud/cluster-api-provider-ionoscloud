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

type kubeVIPProvisioner struct {
	service *cloud.Service
}

func (k *kubeVIPProvisioner) Provision(ctx context.Context, ls *scope.LoadBalancer) (requeue bool, err error) {
	requeue, err = k.service.ReconcileControlPlaneEndpoint(ctx, ls)
	if requeue || err != nil {
		return requeue, err
	}

	return false, nil
}

func (k *kubeVIPProvisioner) Destroy(ctx context.Context, ls *scope.LoadBalancer) (requeue bool, err error) {
	requeue, err = k.service.ReconcileControlPlaneEndpointDeletion(ctx, ls)
	if requeue || err != nil {
		return requeue, err
	}

	return false, nil
}
