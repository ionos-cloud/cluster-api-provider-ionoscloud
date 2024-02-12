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

	sdk "github.com/ionos-cloud/sdk-go/v6"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	controlPlaneEndpointRequestKey = "region-wide"
	ipBlocksPath                   = "ipblocks"
)

// ReconcileControlPlaneEndpoint ensures the control plane endpoint IP block exists.
func (s *Service) ReconcileControlPlaneEndpoint(ctx context.Context, cs *scope.ClusterScope) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileControlPlaneEndpoint")

	ipBlock, request, err := findResource(ctx, s.getIPBlock(cs), s.getLatestIPBlockCreationRequest(cs))
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
		cs.IonosCluster.Spec.ControlPlaneEndpoint.Host = ip
		return false, nil
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

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if ipBlock == nil {
		if err = s.removeIPBlockLeftovers(cs); err != nil {
			return false, err
		}
	}

	request, err = s.getLatestIPBlockDeletionRequest(ctx, *ipBlock.Id)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Deletion request is pending", "location", request.location)
		return true, nil
	}

	err = s.deleteIPBlock(ctx, cs, *ipBlock.Id)
	return err == nil, err
}

// getIPBlock returns a listAndFilterFunc that finds the IP block that matches the expected name and location. An
// error is returned if there are multiple IP blocks that match both the name and location.
func (s *Service) getIPBlock(cs *scope.ClusterScope) listAndFilterFunc[sdk.IpBlock] {
	return func(ctx context.Context) (*sdk.IpBlock, error) {
		blocks, err := s.cloud.ListIPBlocks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list IP blocks: %w", err)
		}

		var (
			expectedName     = cs.DefaultResourceName()
			expectedLocation = string(cs.Region())
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
}

// reserveIPBlock requests for the reservation of an IP block.
func (s *Service) reserveIPBlock(ctx context.Context, cs *scope.ClusterScope) error {
	var err error
	log := s.logger.WithName("reserveIPBlock")

	requestPath, err := s.cloud.ReserveIPBlock(ctx, cs.DefaultResourceName(), string(cs.Region()), 1)
	if err != nil {
		return fmt.Errorf("failed to request the cloud for IP block reservation: %w", err)
	}

	cs.IonosCluster.SetCurrentRequest(controlPlaneEndpointRequestKey, infrav1.NewQueuedRequest(http.MethodPost, requestPath))
	err = cs.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch cluster: %w", err)
	}

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
	cs.IonosCluster.SetCurrentRequest(controlPlaneEndpointRequestKey, infrav1.NewQueuedRequest(http.MethodDelete, requestPath))
	err = cs.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch cluster: %w", err)
	}
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
			path.Join(ipBlocksPath),
			matchByName[*sdk.IpBlock, *sdk.IpBlockProperties](cs.DefaultResourceName()),
			func(r *sdk.IpBlock, _ sdk.Request) bool {
				if r != nil && r.HasProperties() && r.Properties.HasLocation() {
					return *r.Properties.Location == string(cs.Region())
				}
				return false
			},
		)
	}
}

// getLatestIPBlockDeletionRequest returns the latest IP block deletion request.
func (s *Service) getLatestIPBlockDeletionRequest(ctx context.Context, ipBlockID string) (*requestInfo, error) {
	return getMatchingRequest[*sdk.IpBlock](ctx, s, http.MethodDelete, path.Join(ipBlocksPath, ipBlockID))
}

func (s *Service) removeIPBlockLeftovers(cs *scope.ClusterScope) error {
	if cs.IonosCluster.Status.CurrentRequestByDatacenter == nil {
		return nil
	}
	delete(cs.IonosCluster.Status.CurrentRequestByDatacenter, controlPlaneEndpointRequestKey)
	cs.IonosCluster.Spec.ControlPlaneEndpoint.Host = ""
	if err := cs.PatchObject(); err != nil {
		return fmt.Errorf("could not remove leftovers from ionos cluster spec or status: %w", err)
	}
	return nil
}