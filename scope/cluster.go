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

// Package scope defines the provider scopes for reconciliation.
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
)

// Cluster defines a basic cluster context for primary use in IonosCloudClusterReconciler.
type Cluster struct {
	patchHelper  *patch.Helper
	Cluster      *clusterv1.Cluster
	IonosCluster *infrav1.IonosCloudCluster
}

// ClusterParams are the parameters, which are used to create a cluster scope.
type ClusterParams struct {
	Client       client.Client
	Cluster      *clusterv1.Cluster
	IonosCluster *infrav1.IonosCloudCluster
}

// NewCluster creates a new cluster scope with the supplied parameters.
// This is meant to be called on each reconciliation.
func NewCluster(params ClusterParams) (*Cluster, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a cluster scope")
	}

	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a cluster scope")
	}

	if params.IonosCluster == nil {
		return nil, errors.New("IonosCluster is required when creating a cluster scope")
	}

	helper, err := patch.NewHelper(params.IonosCluster, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	clusterScope := &Cluster{
		Cluster:      params.Cluster,
		IonosCluster: params.IonosCluster,
		patchHelper:  helper,
	}

	return clusterScope, nil
}

// GetControlPlaneEndpoint returns the endpoint for the IonosCloudCluster.
func (c *Cluster) GetControlPlaneEndpoint() clusterv1.APIEndpoint {
	return c.IonosCluster.Spec.ControlPlaneEndpoint
}

// SetControlPlaneEndpointIPBlockID sets the IP block ID in the IonosCloudCluster status.
func (c *Cluster) SetControlPlaneEndpointIPBlockID(id string) {
	c.IonosCluster.Status.ControlPlaneEndpointIPBlockID = id
}

// Location is a shortcut for getting the location used by the IONOS Cloud cluster IP block.
func (c *Cluster) Location() string {
	return c.IonosCluster.Spec.Location
}

// PatchObject will apply all changes from the IonosCloudCluster.
// It will also make sure to patch the status subresource.
func (c *Cluster) PatchObject() error {
	// always set the ready condition
	conditions.SetSummary(c.IonosCluster,
		conditions.WithConditions(infrav1.IonosCloudClusterReady))

	// NOTE(piepmatz): We don't accept and forward a context here. This is on purpose: Even if a reconciliation is
	//  aborted, we want to make sure that the final patch is applied. Reusing the context from the reconciliation
	//  would cause the patch to be aborted as well.

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return c.patchHelper.Patch(timeoutCtx, c.IonosCluster, patch.WithOwnedConditions{
		Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
		},
	})
}

// Finalize will make sure to apply a patch to the current IonosCloudCluster.
// It also implements a retry mechanism to increase the chance of success
// in case the patch operation was not successful.
func (c *Cluster) Finalize() error {
	// NOTE(lubedacht) retry is only a way to reduce the failure chance,
	// but in general, the reconciliation logic must be resilient
	// to handle an outdated resource from that API server.
	shouldRetry := func(error) bool { return true }
	return retry.OnError(
		retry.DefaultBackoff,
		shouldRetry,
		c.PatchObject)
}
