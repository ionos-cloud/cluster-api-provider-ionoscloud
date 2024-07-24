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

package cloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"slices"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	ipBlocksPath = "ipblocks"

	// listIPBlocksDepth is the depth needed for getting properties of each IP block.
	listIPBlocksDepth = 1

	defaultControlPlaneEndpointPort int32 = 6443
)

var errUserSetIPNotFound = errors.New("could not find any IP block for the already set control plane endpoint")

// ReconcileControlPlaneEndpoint ensures the control plane endpoint IP block exists.
func (s *Service) ReconcileControlPlaneEndpoint(ctx context.Context, ls *scope.LoadBalancer) (requeue bool, err error) {
	// If the endpoints match, there is nothing we have to do
	if ls.Endpoint().IsValid() && ls.InfraClusterEndpoint() == ls.Endpoint() {
		return false, nil
	}

	log := s.logger.WithName("ReconcileControlPlaneEndpoint")

	ipBlock, request, err := scopedFindResource(
		ctx, ls,
		s.getControlPlaneEndpointIPBlock,
		s.getLatestControlPlaneEndpointIPBlockCreationRequest,
	)
	if err != nil {
		return false, err
	}

	if ipBlock != nil {
		if state := getState(ipBlock); !isAvailable(state) {
			log.Info("IP block is not available yet", "state", state)
			return true, nil
		}

		if ls.Endpoint().Host == "" {
			ip := (*ipBlock.Properties.Ips)[0]
			ls.LoadBalancer.Spec.LoadBalancerEndpoint.Host = ip
		}

		if ls.Endpoint().Port == 0 {
			ls.LoadBalancer.Spec.LoadBalancerEndpoint.Port = defaultControlPlaneEndpointPort
		}

		ls.LoadBalancer.Status.LoadBalancerEndpointIPBlockID = *ipBlock.Id
		return false, nil
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		ls.LoadBalancer.SetCurrentRequest(http.MethodPost, request.status, request.location)
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	log.V(4).Info("No IP block was found. Creating new IP block")
	if err := s.reserveControlPlaneEndpointIPBlock(ctx, ls); err != nil {
		return false, err
	}
	return true, nil
}

// ReconcileControlPlaneEndpointDeletion ensures the control plane endpoint IP block is deleted.
func (s *Service) ReconcileControlPlaneEndpointDeletion(
	ctx context.Context, ls *scope.LoadBalancer,
) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileControlPlaneEndpointDeletion")

	// Try to retrieve the cluster IP Block or even check if it's currently still being created.
	ipBlock, request, err := scopedFindResource(
		ctx, ls,
		s.getControlPlaneEndpointIPBlock,
		s.getLatestControlPlaneEndpointIPBlockCreationRequest,
	)

	if ignoreErrUserSetIPNotFound(ignoreNotFound(err)) != nil {
		return false, err
	}
	// NOTE(gfariasalves): we ignore the error if it is a "user set IP not found" error, because it doesn't matter here.
	// This error is only relevant when we are trying to create a new IP block. If it shows up here, it means that:
	// a) this IP block was created by the user, and they have deleted it, or,
	// b) the IP block was created by the controller, we have already requested its deletion, this is the second
	// part of the reconciliation loop, and the resource is now gone.
	// For both cases this means success, so we can return early with no error.
	if errors.Is(err, errUserSetIPNotFound) || ipBlock == nil && request == nil {
		ls.LoadBalancer.DeleteCurrentRequest()
		return false, nil
	}

	// NOTE: this check covers the case where customers have set the control plane endpoint IP themselves.
	// If this is the case we don't request for the deletion of the IP block.
	if ipBlock != nil && ptr.Deref(ipBlock.GetProperties().GetName(), unknownValue) != s.controlPlaneEndpointIPBlockName(ls.ClusterScope) {
		log.Info("Control Plane Endpoint was created externally by the user. Skipping deletion")
		return false, nil
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		ls.LoadBalancer.SetCurrentRequest(http.MethodPost, request.status, request.location)
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	request, err = s.getLatestIPBlockDeletionRequest(ctx, *ipBlock.Id)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		ls.LoadBalancer.SetCurrentRequest(http.MethodDelete, request.status, request.location)
		log.Info("Deletion request is pending", "location", request.location)
		return true, nil
	}

	err = s.deleteControlPlaneEndpointIPBlock(ctx, ls, *ipBlock.Id)
	return err == nil, err
}

// ReconcileFailoverIPBlockDeletion ensures that the IP block is deleted.
func (s *Service) ReconcileFailoverIPBlockDeletion(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileFailoverIPBlockDeletion")
	if foIP := ms.IonosMachine.Spec.FailoverIP; foIP == nil || *foIP != infrav1.CloudResourceConfigAuto {
		log.V(4).Info("Failover IP block is not managed by the provider, skipping deletion", "failoverIP", foIP)
		return false, nil
	}

	lockKey := s.failoverIPBlockLockKey(ms)
	if err := ms.Locker.Lock(ctx, lockKey); err != nil {
		return false, err
	}
	defer ms.Locker.Unlock(lockKey)

	// Check if the IP block is currently in creation. We need to wait for it to be finished
	// before we can trigger the deletion.
	ipBlock, request, err := scopedFindResource(
		ctx,
		ms,
		s.getFailoverIPBlock,
		s.getLatestFailoverIPBlockCreateRequest,
	)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		ms.IonosMachine.SetCurrentRequest(http.MethodPost, sdk.RequestStatusQueued, request.location)
		return true, nil
	}

	if ipBlock == nil {
		ms.IonosMachine.DeleteCurrentRequest()
		return false, nil
	}

	request, err = s.getLatestIPBlockDeletionRequest(ctx, *ipBlock.GetId())
	if err != nil {
		return false, err
	}

	if request != nil {
		if request.isPending() {
			ms.IonosMachine.SetCurrentRequest(http.MethodDelete, sdk.RequestStatusQueued, request.location)
			return true, nil
		}

		if request.isDone() {
			ms.IonosMachine.DeleteCurrentRequest()
			return false, nil
		}
	}

	// Check if IPBlock is still being used
	lan, err := s.getLAN(ctx, ms)
	if err != nil {
		return false, err
	}

	ipSet := sets.New(ptr.Deref(ipBlock.GetProperties().GetIps(), []string{})...)

	if shouldSkipIPBlockDeletion(lan, ipSet) {
		log.V(4).Info("Failover IP block is still in use, skipping deletion")
		return false, nil
	}

	return true, s.deleteFailoverIPBlock(ctx, ms, *ipBlock.GetId())
}

func shouldSkipIPBlockDeletion(lan *sdk.Lan, ipSet sets.Set[string]) bool {
	if lan == nil {
		return false
	}

	// Check if the failover IP is still in use
	failoverConfigs := ptr.Deref(lan.GetProperties().GetIpFailover(), []sdk.IPFailover{})
	for _, failoverConfig := range failoverConfigs {
		ip := ptr.Deref(failoverConfig.GetIp(), unknownValue)
		if ipSet.Has(ip) {
			return true
		}
	}

	return false
}

func (s *Service) getFailoverIPBlock(ctx context.Context, ms *scope.Machine) (*sdk.IpBlock, error) {
	blocks, err := s.apiWithDepth(listIPBlocksDepth).ListIPBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list IP blocks: %w", err)
	}

	location, err := s.getLocation(ctx, ms)
	if err != nil {
		return nil, err
	}

	for _, block := range ptr.Deref(blocks.GetItems(), nil) {
		props := block.GetProperties()
		if ptr.Deref(props.GetLocation(), "") != location {
			continue
		}
		if ptr.Deref(props.GetName(), "") == s.failoverIPBlockName(ms) {
			return &block, nil
		}
	}

	return nil, nil
}

// getControlPlaneEndpointIPBlock finds the IP block that matches the expected name and location.
// An error is returned if there are multiple IP blocks that match both the name and location.
func (s *Service) getControlPlaneEndpointIPBlock(ctx context.Context, ls *scope.LoadBalancer) (*sdk.IpBlock, error) {
	ipBlock, err := s.getIPBlockByID(ctx, ls.LoadBalancer.Status.LoadBalancerEndpointIPBlockID)
	if ipBlock != nil || ignoreNotFound(err) != nil {
		return ipBlock, err
	}

	notFoundError := err

	blocks, err := s.apiWithDepth(listIPBlocksDepth).ListIPBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list IP blocks: %w", err)
	}

	controlPlaneEndpointIP, err := ls.ClusterScope.ResolveIPAddressForHost(ctx, ls.Endpoint().Host)
	if err != nil {
		return nil, err
	}

	var (
		expectedName     = s.controlPlaneEndpointIPBlockName(ls.ClusterScope)
		expectedLocation = ls.ClusterScope.Location()
		count            = 0
		foundBlock       *sdk.IpBlock
	)

	for _, block := range ptr.Deref(blocks.GetItems(), nil) {
		props := block.GetProperties()
		switch {
		case ptr.Deref(props.GetLocation(), "") != expectedLocation:
			continue
		case ptr.Deref(props.GetName(), "") == expectedName:
			count++
			foundBlock = &block
		case s.checkIfUserSetBlock(controlPlaneEndpointIP, props):
			// NOTE: this is for when customers set IPs for the control plane endpoint themselves.
			return &block, nil
		}
		if count > 1 {
			return nil, fmt.Errorf(
				"cannot determine IP block for Control Plane Endpoint, as there are multiple IP blocks with the name %s",
				expectedName)
		}
	}
	if count == 0 && controlPlaneEndpointIP != "" {
		return nil, errUserSetIPNotFound
	}
	if foundBlock != nil {
		return foundBlock, nil
	}
	// if we still can't find an IP block we return the potential
	// initial not found error.
	return nil, notFoundError
}

func (*Service) checkIfUserSetBlock(controlPlaneEndpointIP string, props *sdk.IpBlockProperties) bool {
	ips := ptr.Deref(props.GetIps(), nil)
	return controlPlaneEndpointIP != "" && slices.Contains(ips, controlPlaneEndpointIP)
}

func (s *Service) getIPBlockByID(ctx context.Context, ipBlockID string) (*sdk.IpBlock, error) {
	if ipBlockID == "" {
		s.logger.Info("Could not find any IP block by ID, as the provider ID is not set.")
		return nil, nil
	}
	ipBlock, err := s.ionosClient.GetIPBlock(ctx, ipBlockID)
	if err != nil {
		return nil, fmt.Errorf("could not get IP block by ID using the API: %w", err)
	}
	return ipBlock, nil
}

// reserveControlPlaneEndpointIPBlock requests for the reservation of an IP block for the control plane.
func (s *Service) reserveControlPlaneEndpointIPBlock(ctx context.Context, ls *scope.LoadBalancer) error {
	log := s.logger.WithName("reserveControlPlaneEndpointIPBlock")
	return s.reserveIPBlock(
		ctx, s.controlPlaneEndpointIPBlockName(ls.ClusterScope),
		ls.ClusterScope.Location(), log,
		ls.LoadBalancer.SetCurrentRequest,
	)
}

func (s *Service) reserveMachineDeploymentFailoverIPBlock(ctx context.Context, ms *scope.Machine) error {
	log := s.logger.WithName("reserveMachineDeploymentFailoverIPBlock")
	location, err := s.getLocation(ctx, ms)
	if err != nil {
		return err
	}

	ms.IonosMachine.Status.Location = location

	return s.reserveIPBlock(
		ctx, s.failoverIPBlockName(ms),
		location, log,
		ms.IonosMachine.SetCurrentRequest,
	)
}

func (s *Service) reserveIPBlock(
	ctx context.Context,
	ipBlockName,
	location string,
	log logr.Logger,
	setRequestStatusFunc func(string, string, string),
) error {
	requestPath, err := s.ionosClient.ReserveIPBlock(ctx, ipBlockName, location, 1)
	if err != nil {
		return fmt.Errorf("failed to request the cloud for IP block reservation: %w", err)
	}

	setRequestStatusFunc(http.MethodPost, sdk.RequestStatusQueued, requestPath)
	log.Info("Successfully requested for IP block reservation", "requestPath", requestPath)

	return nil
}

// deleteControlPlaneEndpointIPBlock requests for the deletion of the control plane IP block with the given ID.
func (s *Service) deleteControlPlaneEndpointIPBlock(ctx context.Context, ls *scope.LoadBalancer, ipBlockID string) error {
	log := s.logger.WithName("deleteControlPlaneEndpointIPBlock")
	return s.deleteIPBlock(ctx, log, ipBlockID, ls.LoadBalancer.SetCurrentRequest)
}

// deleteFailoverIPBlock requests for the deletion of the failover IP block with the given ID.
func (s *Service) deleteFailoverIPBlock(ctx context.Context, ms *scope.Machine, ipBlockID string) error {
	log := s.logger.WithName("deleteFailoverIPBlock")
	return s.deleteIPBlock(ctx, log, ipBlockID, ms.IonosMachine.SetCurrentRequest)
}

func (s *Service) deleteIPBlock(
	ctx context.Context,
	log logr.Logger,
	ipBlockID string,
	setRequestStatusFunc func(string, string, string),
) error {
	requestPath, err := s.ionosClient.DeleteIPBlock(ctx, ipBlockID)
	if err != nil {
		return fmt.Errorf("failed to request IP block deletion: %w", err)
	}

	setRequestStatusFunc(http.MethodDelete, sdk.RequestStatusQueued, requestPath)
	log.Info("Successfully requested for IP block deletion", "requestPath", requestPath)
	return nil
}

// getLatestControlPlaneEndpointIPBlockCreationRequest returns the latest IP block creation request.
func (s *Service) getLatestControlPlaneEndpointIPBlockCreationRequest(
	ctx context.Context,
	ls *scope.LoadBalancer,
) (*requestInfo, error) {
	return s.getLatestIPBlockRequestByNameAndLocation(
		ctx, http.MethodPost,
		s.controlPlaneEndpointIPBlockName(ls.ClusterScope),
		ls.ClusterScope.Location(),
	)
}

// getLatestFailoverIPBlockCreateRequest returns the latest failover IP block creation request.
func (s *Service) getLatestFailoverIPBlockCreateRequest(ctx context.Context, ms *scope.Machine) (*requestInfo, error) {
	location, err := s.getLocation(ctx, ms)
	if err != nil {
		return nil, err
	}
	return s.getLatestIPBlockRequestByNameAndLocation(
		ctx, http.MethodPost,
		s.failoverIPBlockName(ms),
		location,
	)
}

// getLatestIPBlockRequestByNameAndLocation returns the latest IP block creation request by a given name and location.
func (s *Service) getLatestIPBlockRequestByNameAndLocation(
	ctx context.Context,
	method,
	ipBlockName,
	location string,
) (*requestInfo, error) {
	return getMatchingRequest(
		ctx,
		s,
		method,
		ipBlocksPath,
		matchByName[*sdk.IpBlock, *sdk.IpBlockProperties](ipBlockName),
		func(r *sdk.IpBlock, _ sdk.Request) bool {
			return ptr.Deref(r.GetProperties().GetLocation(), unknownValue) == location
		},
	)
}

// getLatestIPBlockDeletionRequest returns the latest IP block deletion request.
func (s *Service) getLatestIPBlockDeletionRequest(ctx context.Context, ipBlockID string) (*requestInfo, error) {
	return getMatchingRequest[*sdk.IpBlock](ctx, s, http.MethodDelete, path.Join(ipBlocksPath, ipBlockID))
}

// controlPlaneEndpointIPBlockName returns the name that should be used for cluster context resources.
func (*Service) controlPlaneEndpointIPBlockName(cs *scope.Cluster) string {
	return fmt.Sprintf("ipb-%s-%s", cs.Cluster.Namespace, cs.Cluster.Name)
}

func (*Service) failoverIPBlockName(ms *scope.Machine) string {
	return fmt.Sprintf("fo-ipb-%s-%s",
		ms.IonosMachine.Namespace,
		ms.IonosMachine.Labels[clusterv1.MachineDeploymentNameLabel],
	)
}

func (*Service) failoverIPBlockLockKey(ms *scope.Machine) string {
	// Failover IPs are shared across machines within the same failover group.
	// When reserving the corresponding IP block, we must avoid duplicate reservations caused by concurrent machine
	// reconciliations. So we lock when performing write operations.
	// As the failover group corresponds with the MachineDeployment the machines belong to, we use the MachineDeployment
	// namespace and name as part of the key used for locking. That's more fine-grained than using the machine's
	// datacenter ID and allows working on distinct failover groups within the same datacenter in parallel.
	return "fo-ipb/" + ms.IonosMachine.Namespace + "/" + ms.IonosMachine.Labels[clusterv1.MachineDeploymentNameLabel]
}

func ignoreErrUserSetIPNotFound(err error) error {
	if errors.Is(err, errUserSetIPNotFound) {
		return nil
	}
	return err
}

// getLocation checks if the location of the machine is already set in the status.
// If not, it queries Cloud API to get the location of the datacenter.
func (s *Service) getLocation(ctx context.Context, ms *scope.Machine) (string, error) {
	if ms.IonosMachine.Status.Location == "" {
		location, err := s.ionosClient.GetDatacenterLocationByID(ctx, ms.IonosMachine.Spec.DatacenterID)
		if err != nil {
			return "", fmt.Errorf("failed to get location of datacenter: %w", err)
		}
		ms.IonosMachine.Status.Location = location
		return location, nil
	}
	return ms.IonosMachine.Status.Location, nil
}
