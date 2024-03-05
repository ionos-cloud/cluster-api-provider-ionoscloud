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

	"github.com/go-logr/logr"
	"k8s.io/client-go/util/retry"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
)

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	*logr.Logger // Deprecated

	client      client.Client
	patchHelper *patch.Helper

	Cluster                        *clusterv1.Cluster
	IonosCluster                   *infrav1.IonosCloudCluster
	KubeadmControlPlane            *controlplanev1.KubeadmControlPlane
	kubeadmControlPlanePatchHelper *patch.Helper
	IonosClient                    ionoscloud.Client // Deprecated
}

// ClusterScopeParams are the parameters, which are used to create a cluster scope.
type ClusterScopeParams struct {
	Client              client.Client
	Logger              *logr.Logger
	Cluster             *clusterv1.Cluster
	IonosCluster        *infrav1.IonosCloudCluster
	IonosClient         ionoscloud.Client
	KubeadmControlPlane *controlplanev1.KubeadmControlPlane
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
	if params.IonosClient == nil {
		return nil, errors.New("IonosClient is required when creating a ClusterScope")
	}

	if params.Logger == nil {
		logger := ctrl.Log
		params.Logger = &logger
	}

	helper, err := patch.NewHelper(params.IonosCluster, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	clusterScope := &ClusterScope{
		Logger:       params.Logger,
		Cluster:      params.Cluster,
		IonosCluster: params.IonosCluster,
		IonosClient:  params.IonosClient,
		client:       params.Client,
		patchHelper:  helper,
	}

	if params.KubeadmControlPlane != nil {
		kcpHelper, err := patch.NewHelper(params.KubeadmControlPlane, params.Client)
		if err != nil {
			return nil, fmt.Errorf("failed to init kubeadm control plane patch kcpHelper: %w", err)
		}
		clusterScope.KubeadmControlPlane = params.KubeadmControlPlane
		clusterScope.kubeadmControlPlanePatchHelper = kcpHelper
	}

	return clusterScope, nil
}

// GetControlPlaneEndpoint returns the endpoint for the IonosCloudCluster.
func (c *ClusterScope) GetControlPlaneEndpoint() clusterv1.APIEndpoint {
	return c.IonosCluster.Spec.ControlPlaneEndpoint
}

// SetControlPlaneEndpointIPBlockID sets the IP block ID in the IonosCloudCluster status.
func (c *ClusterScope) SetControlPlaneEndpointIPBlockID(id string) {
	c.IonosCluster.Status.ControlPlaneEndpointIPBlockID = id
}

// Location is a shortcut for getting the location used by the IONOS Cloud cluster IP block.
func (c *ClusterScope) Location() string {
	return c.IonosCluster.Spec.Location
}

// PatchObject will apply all changes from the IonosCloudCluster.
// It will also make sure to patch the status subresource.
func (c *ClusterScope) PatchObject() error {
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

// PatchKubeadmControlPlane will apply all changes from the KubeadmControlPlane.
func (c *ClusterScope) PatchKubeadmControlPlane() error {
	if c.kubeadmControlPlanePatchHelper == nil {
		return errors.New("kubeadmControlPlanePatchHelper is not initialized")
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return c.kubeadmControlPlanePatchHelper.Patch(timeoutCtx, c.KubeadmControlPlane)
}

// Finalize will make sure to apply a patch to the current IonosCloudCluster.
// It also implements a retry mechanism to increase the chance of success
// in case the patch operation was not successful.
func (c *ClusterScope) Finalize() error {
	// NOTE(lubedacht) retry is only a way to reduce the failure chance,
	// but in general, the reconciliation logic must be resilient
	// to handle an outdated resource from that API server.
	shouldRetry := func(error) bool { return true }
	return retry.OnError(
		retry.DefaultBackoff,
		shouldRetry,
		c.PatchObject)
}
