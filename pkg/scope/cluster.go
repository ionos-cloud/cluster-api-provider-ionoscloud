/*
Copyright 2023 IONOS Cloud.

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

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
)

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	*logr.Logger

	client      client.Client
	patchHelper *patch.Helper

	Cluster      *clusterv1.Cluster
	IonosCluster *infrav1.IonosCloudCluster

	IonosClient ionoscloud.Client
}

// ClusterScopeParams are the parameters, which are used to create a cluster scope.
type ClusterScopeParams struct {
	Client       client.Client
	Logger       *logr.Logger
	Cluster      *clusterv1.Cluster
	IonosCluster *infrav1.IonosCloudCluster
	IonosClient  ionoscloud.Client
}

// NewClusterScope creates a new scope for the supplied parameters.
// This is meant to be called on each reconciliation.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a ClusterScope")
	}

	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a ClusterScope")
	}

	if params.IonosCluster == nil {
		return nil, errors.New("IonosCluster is required when creating a ClusterScope")
	}
	if params.Client == nil {
		return nil, errors.New("IonosCluster is required when creating a ClusterScope")
	}

	if params.Logger == nil {
		logger := log.FromContext(context.Background())
		params.Logger = &logger
	}

	helper, err := patch.NewHelper(params.IonosCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	clusterScope := &ClusterScope{
		Logger:       params.Logger,
		Cluster:      params.Cluster,
		IonosCluster: params.IonosCluster,
		IonosClient:  params.IonosClient,
		client:       params.Client,
		patchHelper:  helper,
	}

	return clusterScope, nil
}

// PatchObject will apply all changes from the IonosCloudCluster.
// It will also make sure to patch the status subresource.
func (c *ClusterScope) PatchObject() error {
	// always set the ready condition
	conditions.SetSummary(c.IonosCluster,
		conditions.WithConditions(infrav1.IonosCloudClusterReady))

	return c.patchHelper.Patch(context.TODO(), c.IonosCluster)
}

// Finalize will make sure to apply a patch to the current IonosCloudCluster.
func (c *ClusterScope) Finalize() error {
	return c.PatchObject()
}