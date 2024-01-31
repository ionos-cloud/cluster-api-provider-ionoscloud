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
	"fmt"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

const ControlPlaneEndpointRequestKey = "region-wide"

// ReconcileControlPlaneEndpoint ensures the control plane endpoint IP block exists, creating one if it doesn't.
func (s *Service) ReconcileControlPlaneEndpoint() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileControlPlaneEndpoint")

	ipBlock, request, err := findResource(s.getIPBlock, s.getLatestIPBlockCreationRequest)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	if ipBlock != nil {
		if state := getState(ipBlock); !isAvailable(state) {
			log.Info("IP block is not available yet", "state", state)
			return true, nil
		}
		ip := (*ipBlock.Properties.Ips)[0]
		s.scope.ClusterScope.IonosCluster.Spec.ControlPlaneEndpoint.Host = ip
		return false, nil
	}

	log.V(4).Info("No IP block was found. Creating new IP block")
	if err := s.reserveIPBlock(); err != nil {
		return false, err
	}
	return true, nil
}

// reserveIPBlock requests for the reservation of an IP block.
func (s *Service) reserveIPBlock() error {
	log := s.scope.Logger.WithName("createIPBlock")

	request, err := s.api().ReserveIPBlock(s.ctx, s.lanName(), s.ipBlockLocation(), 1)
	if err != nil {
		return fmt.Errorf("failed to request IP block reservation: %w", err)
	}
	s.scope.ClusterScope.IonosCluster.SetCurrentRequest(
		ControlPlaneEndpointRequestKey, infrav1.NewQueuedRequest(http.MethodPost, request))
	err = s.scope.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch cluster: %w", err)
	}
	log.Info("Successfully requested for IP block reservation", "requestPath", request)
	return nil
}

// getLatestIPBlockCreationRequest returns the latest IP block creation request.
func (s *Service) getLatestIPBlockCreationRequest() (*requestInfo, error) {
	return getMatchingRequest(
		s,
		http.MethodPost,
		path.Join("ipblocks"),
		matchByName[*sdk.IpBlock, *sdk.IpBlockProperties](s.ipBlockName()),
		func(r *sdk.IpBlock, _ sdk.Request) bool {
			if r != nil && r.HasProperties() && r.Properties.HasLocation() {
				return *r.Properties.Location == s.ipBlockLocation()
			}
			return false
		},
	)
}

// getIPBlock returns the IP block that matches the expected name and location. An error is returned if there are
// multiple IP blocks that match both the name and location.
func (s *Service) getIPBlock() (*sdk.IpBlock, error) {
	blocks, err := s.api().ListIPBlocks(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list IP blocks: %w", err)
	}

	var (
		expectedName     = s.ipBlockName()
		expectedLocation = s.ipBlockLocation()
		count            = 0
		foundBlock       *sdk.IpBlock
	)
	if len(blocks) > 0 {
		for _, block := range blocks {
			if props := block.Properties; props != nil {
				if *props.Name == expectedName && *props.Location == expectedLocation {
					count++
					foundBlock = ptr.To(block)
				}
			}
			if count > 1 {
				return nil, fmt.Errorf(
					"cannot determine IP block for Control Plane Endpoint as there are multiple IP blocks with the name %s",
					expectedName)
			}
		}
		return foundBlock, nil
	}
	return nil, nil
}

// ReconcileControlPlaneEndpointDeletion ensures the control plane endpoint IP block is deleted.
func (s *Service) ReconcileControlPlaneEndpointDeletion() (bool, error) {
	log := s.scope.Logger.WithName("ReconcileControlPlaneEndpointDeletion")

	// Try to retrieve the CPE IP Block or even check if it's currently still being created.
	ipBlock, request, err := findResource(s.getIPBlock, s.getLatestIPBlockCreationRequest)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if ipBlock == nil {
		err = s.removeIPBlockPendingRequestFromCluster()
		return err != nil, err
	}

	// If we found an IP Block, we check if there is already a deletion in progress.
	request, err = s.getLatestIPBlockDeletionRequest(*ipBlock.Id)
	if err != nil {
		return false, err
	}
	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Deletion request is pending", "location", request.location)
		return true, nil
	}

	// Request for IP Block deletion
	err = s.deleteIPBlock(*ipBlock.Id)
	return err == nil, err
}

// getLatestIPBlockDeletionRequest returns the latest IP block deletion request.
func (s *Service) getLatestIPBlockDeletionRequest(ipBlockID string) (*requestInfo, error) {
	return getMatchingRequest[sdk.IpBlock](
		s,
		http.MethodDelete,
		path.Join("ipblocks", ipBlockID),
	)
}

// deleteIPBlock requests for the deletion of the IP block with the given ID.
func (s *Service) deleteIPBlock(ipBlockID string) error {
	log := s.scope.Logger.WithName("deleteIPBlock")

	request, err := s.api().DeleteIPBlock(s.ctx, ipBlockID)
	if err != nil {
		return fmt.Errorf("failed to request IP block deletion: %w", err)
	}
	s.scope.ClusterScope.IonosCluster.SetCurrentRequest(
		ControlPlaneEndpointRequestKey, infrav1.NewQueuedRequest(http.MethodDelete, request))
	err = s.scope.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch cluster: %w", err)
	}
	log.Info("Successfully requested for IP block deletion", "requestPath", request)
	return nil
}

// ipBlockName returns the name of the cluster IP block.
func (s *Service) ipBlockName() string {
	return fmt.Sprintf("k8s-%s-%s", s.scope.ClusterScope.Cluster.Namespace, s.scope.ClusterScope.Cluster.Name)
}

// ipBlockLocation returns the location of the cluster IP block.
func (s *Service) ipBlockLocation() string {
	return "LOCATION_PLACEHOLDER"
}

// removeIPBlockPendingRequestFromCluster removes the IP block pending request from the cluster.
func (s *Service) removeIPBlockPendingRequestFromCluster() error {
	s.scope.ClusterScope.IonosCluster.DeleteCurrentRequest(ControlPlaneEndpointRequestKey)
	if err := s.scope.ClusterScope.PatchObject(); err != nil {
		return fmt.Errorf("could not remove stale IP Block pending request from cluster: %w", err)
	}
	return nil
}
