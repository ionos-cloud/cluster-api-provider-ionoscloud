package cloud

import (
	"fmt"
	"net/http"
	"path"
	"slices"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

func (s *Service) nicURL(id string) string {
	return path.Join("datacenters", s.datacenterID(), "nics", id)
}

// reconcileNICConfig ensures that the primary NIC contains the endpoint IP address.
func (s *Service) reconcileNICConfig(endpointIP string) (*sdk.Nic, error) {
	log := s.scope.Logger.WithName("reconcileNICConfig")

	log.V(4).Info("Reconciling NIC config")
	// Get current state of the server
	server, err := s.getServer()
	if err != nil {
		return nil, err
	}

	// Find the primary NIC and ensure that the endpoint IP address is added to the NIC.
	nic, err := s.findPrimaryNIC(server)
	if err != nil {
		return nil, err
	}

	// if the NIC already contains the endpoint IP address, we can return
	if nicHasIP(nic, endpointIP) {
		log.V(4).Info("Primary NIC contains endpoint IP address. Reconcile successful.")
		return nic, nil
	}

	nicID := ptr.Deref(server.GetId(), "")
	// check if there is a pending patch request for the NIC
	ri, err := s.getLatestNICPatchRequest(nicID)
	if err != nil {
		return nil, fmt.Errorf("unable to check for pending NIC patch request: %w", err)
	}

	if ri != nil && ri.isPending() {
		log.Info("Found pending NIC request. Waiting for it to be finished")

		if err := s.api().WaitForRequest(s.ctx, ri.location); err != nil {
			return nil, fmt.Errorf("failed to wait for pending NIC request: %w", err)
		}

		return nic, nil
	}

	log.V(4).Info("Unable to find endpoint IP address in primary NIC. Patching NIC.")
	nicIPs := ptr.Deref(nic.GetProperties().GetIps(), []string{})
	nicIPs = append(nicIPs, endpointIP)

	if err := s.patchNIC(nicID, nic, sdk.NicProperties{Ips: &nicIPs}); err != nil {
		return nil, err
	}

	log.V(4).Info("Successfully patched NIC. Finished reconciling NIC config.")
	// As we are waiting for the request to finish this time, we can assume that the request was successful
	// Therefore we can remove the current request
	s.scope.IonosMachine.Status.CurrentRequest = nil

	return nic, nil
}

func (s *Service) findPrimaryNIC(server *sdk.Server) (*sdk.Nic, error) {
	serverNICs := ptr.Deref(server.GetEntities().GetNics().GetItems(), []sdk.Nic{})
	for _, nic := range serverNICs {
		if name := ptr.Deref(nic.GetProperties().GetName(), ""); name == s.serverName() {
			return &nic, nil
		}
	}
	return nil, fmt.Errorf("could not find primary NIC with name %s", s.serverName())
}

func (s *Service) patchNIC(serverID string, nic *sdk.Nic, props sdk.NicProperties) error {
	log := s.scope.Logger.WithName("patchNIC")

	nicID := ptr.Deref(nic.GetId(), "")
	log.V(4).Info("Patching NIC", "id", nicID)

	location, err := s.api().PatchNIC(s.ctx, s.datacenterID(), serverID, nicID, props)
	if err != nil {
		return fmt.Errorf("failed to patch NIC %s: %w", nicID, err)
	}

	// set the current request in case the WaitForRequest function fails.
	s.scope.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodPatch, location))

	log.V(4).Info("Successfully patched NIC", "location", location)
	// In this case, we want to wait for the request to be finished as we need to configure the
	// failover group
	return s.api().WaitForRequest(s.ctx, location)
}

func (s *Service) getLatestNICPatchRequest(nicID string) (*requestInfo, error) {
	return getMatchingRequest[sdk.Lan](
		s,
		http.MethodPatch,
		s.nicURL(nicID),
	)
}

// nicHasIP returns true if the NIC contains the given IP address.
func nicHasIP(nic *sdk.Nic, expectedIP string) bool {
	ips := ptr.Deref(nic.GetProperties().GetIps(), []string{})
	return slices.Contains(ips, expectedIP)
}
