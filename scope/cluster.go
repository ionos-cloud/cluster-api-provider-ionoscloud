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
	"net"
	"net/netip"
	"slices"
	"time"

	"k8s.io/client-go/util/retry"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
)

// resolver is able to look up IP addresses from a given host name.
// The net.Resolver type (found at net.DefaultResolver) implements this interface.
// This is intended for testing.
type resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// Cluster defines a basic cluster context for primary use in IonosCloudClusterReconciler.
type Cluster struct {
	client       client.Client
	patchHelper  *patch.Helper
	resolver     resolver
	Locker       *locker.Locker
	Cluster      *clusterv1.Cluster
	IonosCluster *infrav1.IonosCloudCluster
}

// ClusterParams are the parameters, which are used to create a cluster scope.
type ClusterParams struct {
	Client       client.Client
	Cluster      *clusterv1.Cluster
	IonosCluster *infrav1.IonosCloudCluster
	Locker       *locker.Locker
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

	if params.Locker == nil {
		return nil, errors.New("locker is required when creating a cluster scope")
	}

	helper, err := patch.NewHelper(params.IonosCluster, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	clusterScope := &Cluster{
		client:       params.Client,
		Cluster:      params.Cluster,
		IonosCluster: params.IonosCluster,
		patchHelper:  helper,
		resolver:     net.DefaultResolver,
		Locker:       params.Locker,
	}

	return clusterScope, nil
}

// GetControlPlaneEndpoint returns the endpoint for the IonosCloudCluster.
func (c *Cluster) GetControlPlaneEndpoint() clusterv1.APIEndpoint {
	return c.IonosCluster.Spec.ControlPlaneEndpoint
}

// GetControlPlaneEndpointIP returns the endpoint IP for the IonosCloudCluster.
// If the endpoint host is unset (neither an IP nor an FQDN), it will return an empty string.
func (c *Cluster) GetControlPlaneEndpointIP(ctx context.Context) (string, error) {
	host := c.GetControlPlaneEndpoint().Host
	if host == "" {
		return "", nil
	}

	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.String(), nil
	}

	// If the host is not an IP, try to resolve it.
	ips, err := c.resolver.LookupNetIP(ctx, "ip4", host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve control plane endpoint IP: %w", err)
	}

	// Sort IPs to deal with random order intended for load balancing.
	slices.SortFunc(ips, func(a, b netip.Addr) int { return a.Compare(b) })

	return ips[0].String(), nil
}

// SetControlPlaneEndpointIPBlockID sets the IP block ID in the IonosCloudCluster status.
func (c *Cluster) SetControlPlaneEndpointIPBlockID(id string) {
	c.IonosCluster.Status.ControlPlaneEndpointIPBlockID = id
}

// ListMachines returns a list of IonosCloudMachines in the same namespace and with the same cluster label.
// With machineLabels, additional search labels can be provided.
func (c *Cluster) ListMachines(
	ctx context.Context,
	machineLabels client.MatchingLabels,
) ([]infrav1.IonosCloudMachine, error) {
	if machineLabels == nil {
		machineLabels = client.MatchingLabels{}
	}

	machineLabels[clusterv1.ClusterNameLabel] = c.Cluster.Name
	listOpts := []client.ListOption{client.InNamespace(c.Cluster.Namespace), machineLabels}

	machineList := &infrav1.IonosCloudMachineList{}
	if err := c.client.List(ctx, machineList, listOpts...); err != nil {
		return nil, err
	}
	return machineList.Items, nil
}

// Location is a shortcut for getting the location used by the IONOS Cloud cluster IP block.
func (c *Cluster) Location() string {
	return c.IonosCluster.Spec.Location
}

// IsDeleted checks if the cluster was requested for deletion.
func (c *Cluster) IsDeleted() bool {
	return !c.Cluster.DeletionTimestamp.IsZero() || !c.IonosCluster.DeletionTimestamp.IsZero()
}

func (c *Cluster) currentRequestByDatacenterLockKey() string {
	return string(c.Cluster.UID) + "/currentRequestByDatacenter"
}

// GetCurrentRequestByDatacenter returns the current provisioning request for the given data center and a boolean
// indicating if it exists.
func (c *Cluster) GetCurrentRequestByDatacenter(datacenterID string) (infrav1.ProvisioningRequest, bool) {
	lockKey := c.currentRequestByDatacenterLockKey()
	_ = c.Locker.Lock(context.Background(), lockKey)
	defer c.Locker.Unlock(lockKey)
	req, ok := c.IonosCluster.Status.CurrentRequestByDatacenter[datacenterID]
	return req, ok
}

// SetCurrentRequestByDatacenter sets the current provisioning request for the given data center.
// This function makes sure that the map is initialized before setting the request.
func (c *Cluster) SetCurrentRequestByDatacenter(datacenterID, method, status, requestPath string) {
	lockKey := c.currentRequestByDatacenterLockKey()
	_ = c.Locker.Lock(context.Background(), lockKey)
	defer c.Locker.Unlock(lockKey)
	if c.IonosCluster.Status.CurrentRequestByDatacenter == nil {
		c.IonosCluster.Status.CurrentRequestByDatacenter = map[string]infrav1.ProvisioningRequest{}
	}
	c.IonosCluster.Status.CurrentRequestByDatacenter[datacenterID] = infrav1.ProvisioningRequest{
		Method:      method,
		RequestPath: requestPath,
		State:       status,
	}
}

// DeleteCurrentRequestByDatacenter deletes the current provisioning request for the given data center.
func (c *Cluster) DeleteCurrentRequestByDatacenter(datacenterID string) {
	lockKey := c.currentRequestByDatacenterLockKey()
	_ = c.Locker.Lock(context.Background(), lockKey)
	defer c.Locker.Unlock(lockKey)
	delete(c.IonosCluster.Status.CurrentRequestByDatacenter, datacenterID)
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
