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
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	ipBlocksPath = "ipblocks"
)

// ReconcileControlPlaneEndpoint ensures the control plane endpoint IP block exists.
func (s *Service) ReconcileControlPlaneEndpoint(ctx context.Context, cs *scope.ClusterScope) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileControlPlaneEndpoint")

	ipBlock, request, err := findResource(ctx, s.getIPBlock(cs), s.getLatestIPBlockCreationRequest(cs))
	if err != nil {
		return false, err
	}

	if ipBlock != nil {
		if state := getState(ipBlock); !isAvailable(state) {
			log.Info("IP block is not available yet", "state", state)
			return true, nil
		}
		if cs.IonosCluster.Spec.ControlPlaneEndpoint.Host == "" {
			ip := (*ipBlock.Properties.Ips)[0]
			cs.IonosCluster.Spec.ControlPlaneEndpoint.Host = ip
		}
		cs.SetControlPlaneEndpointProviderID(*ipBlock.Id)
		return false, nil
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		cs.IonosCluster.SetCurrentClusterRequest(http.MethodPost, request.status, request.location)
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	log.V(4).Info("No IP block was found. Creating new IP block")
	if err := s.reserveIPBlock(ctx, cs); err != nil {
		return false, err
	}
	return true, nil
}

// ReconcileControlPlaneEndpointDeletion ensures the control plane endpoint IP block is deleted.
func (s *Service) ReconcileControlPlaneEndpointDeletion(ctx context.Context, cs *scope.ClusterScope) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileControlPlaneEndpointDeletion")

	// Try to retrieve the cluster IP Block or even check if it's currently still being created.
	ipBlock, request, err := findResource(ctx, s.getIPBlock(cs), s.getLatestIPBlockCreationRequest(cs))
	if err != nil {
		return false, err
	}

	// NOTE: this check covers the case where customers have set the control plane endpoint IP themselves.
	// If this is the case we don't request for the deletion of the IP block.
	if ipBlock != nil && ptr.Deref(ipBlock.GetProperties().GetName(), unknownValue) != s.ipBlockName(cs) {
		log.Info("Control Plane Endpoint was created externally by the user. Skipping deletion")
		return false, nil
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		cs.IonosCluster.SetCurrentClusterRequest(http.MethodPost, request.status, request.location)
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if ipBlock == nil {
		cs.IonosCluster.DeleteCurrentClusterRequest()
		return false, nil
	}

	request, err = s.getLatestIPBlockDeletionRequest(ctx, *ipBlock.Id)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		cs.IonosCluster.SetCurrentClusterRequest(http.MethodDelete, request.status, request.location)
		log.Info("Deletion request is pending", "location", request.location)
		return true, nil
	}

	err = s.deleteIPBlock(ctx, cs, *ipBlock.Id)
	return err == nil, err
}

// getIPBlock returns a listAndFilterFunc that finds the IP block that matches the expected name and location. An
// error is returned if there are multiple IP blocks that match both the name and location.
func (s *Service) getIPBlock(cs *scope.ClusterScope) tryLookupResourceFunc[sdk.IpBlock] {
	return func(ctx context.Context) (*sdk.IpBlock, error) {
		ipBlock, err := s.getIPBlockByID(ctx, cs)
		if ipBlock != nil || ignoreNotFound(err) != nil {
			return ipBlock, err
		}

		s.logger.Info("IP block not found by ID, trying to find by listing IP blocks instead")
		blocks, listErr := s.apiWithDepth(1).ListIPBlocks(ctx)
		if listErr != nil {
			return nil, fmt.Errorf("failed to list IP blocks: %w", listErr)
		}

		var (
			expectedName     = s.ipBlockName(cs)
			expectedLocation = cs.Location()
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
				// NOTE(gfariasalves): Change this to &block after we move to go1.22
				foundBlock = ptr.To(block)
			case s.checkIfUserSetBlock(cs, props):
				// NOTE: this is for when customers set IPs for the control plane endpoint themselves.
				return ptr.To(block), nil
			}
			if count > 1 {
				return nil, fmt.Errorf(
					"cannot determine IP block for Control Plane Endpoint as there are multiple IP blocks with the name %s",
					expectedName)
			}
		}
		if count == 0 && cs.GetControlPlaneEndpoint().Host != "" {
			return nil, fmt.Errorf("could not find any IP block for the already set control plane endpoint")
		}
		if foundBlock != nil {
			return foundBlock, nil
		}
		// if we still can't find a server we return the potential
		// initial not found error.
		return nil, err
	}
}

func (s *Service) checkIfUserSetBlock(cs *scope.ClusterScope, props *sdk.IpBlockProperties) bool {
	ip := cs.GetControlPlaneEndpoint().Host
	ips := ptr.Deref(props.GetIps(), nil)
	return ip != "" && slices.Contains(ips, ip)
}

func (s *Service) getIPBlockByID(ctx context.Context, cs *scope.ClusterScope) (*sdk.IpBlock, error) {
	id := strings.TrimPrefix(cs.IonosCluster.Status.ControlPlaneEndpointProviderID, "ionos://")
	if id == "" {
		s.logger.Info("Could not find any IP block by ID as the provider ID is not set.")
		return nil, nil
	}
	ipBlock, err := s.cloud.GetIPBlock(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get IP block by ID using the API: %w", err)
	}
	return ipBlock, nil
}

// reserveIPBlock requests for the reservation of an IP block.
func (s *Service) reserveIPBlock(ctx context.Context, cs *scope.ClusterScope) error {
	var err error
	log := s.logger.WithName("reserveIPBlock")

	requestPath, err := s.cloud.ReserveIPBlock(ctx, s.ipBlockName(cs), cs.Location(), 1)
	if err != nil {
		return fmt.Errorf("failed to request the cloud for IP block reservation: %w", err)
	}

	cs.IonosCluster.SetCurrentClusterRequest(http.MethodPost, sdk.RequestStatusQueued, requestPath)
	log.Info("Successfully requested for IP block reservation", "requestPath", requestPath)
	return nil
}

// deleteIPBlock requests for the deletion of the IP block with the given ID.
func (s *Service) deleteIPBlock(ctx context.Context, cs *scope.ClusterScope, id string) error {
	log := s.logger.WithName("deleteIPBlock")

	requestPath, err := s.cloud.DeleteIPBlock(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to requestPath IP block deletion: %w", err)
	}
	cs.IonosCluster.SetCurrentClusterRequest(http.MethodDelete, sdk.RequestStatusQueued, requestPath)
	log.Info("Successfully requested for IP block deletion", "requestPath", requestPath)
	return nil
}

// getLatestIPBlockCreationRequest returns the latest IP block creation request.
func (s *Service) getLatestIPBlockCreationRequest(cs *scope.ClusterScope) checkQueueFunc {
	return func(ctx context.Context) (*requestInfo, error) {
		return getMatchingRequest(
			ctx,
			s,
			http.MethodPost,
			ipBlocksPath,
			matchByName[*sdk.IpBlock, *sdk.IpBlockProperties](s.ipBlockName(cs)),
			func(r *sdk.IpBlock, _ sdk.Request) bool {
				return ptr.Deref(r.GetProperties().GetLocation(), unknownValue) == cs.Location()
			},
		)
	}
}

// getLatestIPBlockDeletionRequest returns the latest IP block deletion request.
func (s *Service) getLatestIPBlockDeletionRequest(ctx context.Context, ipBlockID string) (*requestInfo, error) {
	return getMatchingRequest[*sdk.IpBlock](ctx, s, http.MethodDelete, path.Join(ipBlocksPath, ipBlockID))
}

// removeIPBlockLeftovers removes the current cluster request, and deletes the host from the spec.

// ipBlockName returns the name that should be used for cluster context resources.
func (s *Service) ipBlockName(cs *scope.ClusterScope) string {
	return fmt.Sprintf("k8s-%s-%s", cs.Cluster.Namespace, cs.Cluster.Name)
}
