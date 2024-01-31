package cloud

import (
	"fmt"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

func (s *Service) ipblockfn() (requeue bool, err error) {
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

func (s *Service) reserveIPBlock() error {
	log := s.scope.Logger.WithName("createIPBlock")

	request, err := s.api().ReserveIPBlock(s.ctx, s.lanName(), s.ipBlockLocation(), 1)
	if err != nil {
		return fmt.Errorf("failed to request IP block reservation: %w", err)
	}
	s.scope.ClusterScope.IonosCluster.SetCurrentRequest("cluster-wide", infrav1.NewQueuedRequest(http.MethodPost, request))
	log.Info("Successfully requested for IP block reservation", "requestPath", request)
	return nil
}

func (s *Service) getLatestIPBlockCreationRequest() (*requestInfo, error) {
	return getMatchingRequest(
		s,
		http.MethodPost,
		path.Join("ips", "blocks"),
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
					"cannot determine IP block for Control Plane Endpoint as there are multiple IP blocks with the name %s", expectedName)
			}
		}
		return foundBlock, nil
	}
	return nil, nil
}

func (s *Service) deleteIPBlock(resourceUUID string) error {
	log := s.scope.Logger.WithName("deleteIPBlock")

	request, err := s.api().DeleteIPBlock(s.ctx, resourceUUID)
	if err != nil {
		return fmt.Errorf("failed to request IP block deletion: %w", err)
	}
	s.scope.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodDelete, request))
	log.Info("Successfully requested for IP block deletion", "requestPath", request)
	return nil
}

// ipBlockName returns the name of the cluster IP block.
func (s *Service) ipBlockName() string {
	return fmt.Sprintf("k8s-%s-%s", s.scope.ClusterScope.Cluster.Namespace, s.scope.ClusterScope.Cluster.Name)
}

func (s *Service) ipBlockLocation() string {
	return "LOCATION_PLACEHOLDER"
}
