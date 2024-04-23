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

	sdk "github.com/ionos-cloud/sdk-go/v6"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

func (*Service) nicURL(ms *scope.Machine, serverID, nicID string) string {
	return path.Join("datacenters", ms.DatacenterID(), "servers", serverID, "nics", nicID)
}

// reconcileNICConfig ensures that the primary NIC contains the endpoint IP address.
func (s *Service) reconcileNICConfig(ctx context.Context, ms *scope.Machine, endpointIP string) (*sdk.Nic, error) {
	log := s.logger.WithName("reconcileNICConfig")

	log.V(4).Info("Reconciling NIC config")
	// Get current state of the server
	server, err := s.getServer(ctx, ms)
	if err != nil {
		return nil, err
	}

	// Find the primary NIC and ensure that the endpoint IP address is added to the NIC.
	nic, err := s.findPrimaryNIC(ms.IonosMachine, server)
	if err != nil {
		return nil, err
	}

	// if the NIC already contains the endpoint IP address, we can return
	if nicHasIP(nic, endpointIP) {
		log.V(4).Info("Primary NIC contains endpoint IP address. Reconcile successful.")
		return nic, nil
	}

	serverID := ptr.Deref(server.GetId(), "")
	nicID := ptr.Deref(nic.GetId(), "")
	// check if there is a pending patch request for the NIC
	ri, err := s.getLatestNICPatchRequest(ctx, ms, serverID, nicID)
	if err != nil {
		return nil, fmt.Errorf("unable to check for pending NIC patch request: %w", err)
	}

	if ri != nil && ri.isPending() {
		log.Info("Found pending NIC request. Waiting for it to be finished")

		if err := s.ionosClient.WaitForRequest(ctx, ri.location); err != nil {
			return nil, fmt.Errorf("failed to wait for pending NIC request: %w", err)
		}

		return nic, nil
	}

	log.V(4).Info("Unable to find endpoint IP address in primary NIC. Patching NIC.")
	nicIPs := ptr.Deref(nic.GetProperties().GetIps(), []string{})
	nicIPs = append(nicIPs, endpointIP)

	if err := s.patchNIC(ctx, ms, serverID, nic, sdk.NicProperties{Ips: &nicIPs}); err != nil {
		return nil, err
	}

	log.V(4).Info("Successfully patched NIC. Finished reconciling NIC config.")
	// As we are waiting for the request to finish this time, we can assume that the request was successful
	// Therefore we can remove the current request
	ms.IonosMachine.DeleteCurrentRequest()

	return nic, nil
}

func (s *Service) findPrimaryNIC(m *infrav1.IonosCloudMachine, server *sdk.Server) (*sdk.Nic, error) {
	serverNICs := ptr.Deref(server.GetEntities().GetNics().GetItems(), []sdk.Nic{})
	for _, nic := range serverNICs {
		if s.isPrimaryNIC(m, &nic) {
			return &nic, nil
		}
	}

	return nil, fmt.Errorf("could not find primary NIC with name %s", s.nicName(m))
}

func (s *Service) isPrimaryNIC(m *infrav1.IonosCloudMachine, nic *sdk.Nic) bool {
	if name := ptr.Deref(nic.GetProperties().GetName(), ""); name == s.nicName(m) {
		return true
	}

	return false
}

func (s *Service) patchNIC(
	ctx context.Context, ms *scope.Machine, serverID string, nic *sdk.Nic, props sdk.NicProperties,
) error {
	log := s.logger.WithName("patchNIC")

	nicID := ptr.Deref(nic.GetId(), "")
	log.V(4).Info("Patching NIC", "id", nicID)

	location, err := s.ionosClient.PatchNIC(ctx, ms.DatacenterID(), serverID, nicID, props)
	if err != nil {
		return fmt.Errorf("failed to patch NIC %s: %w", nicID, err)
	}

	// set the current request in case the WaitForRequest function fails.
	ms.IonosMachine.SetCurrentRequest(http.MethodPatch, sdk.RequestStatusQueued, location)
	log.V(4).Info("Successfully patched NIC", "location", location)
	// In this case, we want to wait for the request to be finished as we need to configure the
	// failover group
	return s.ionosClient.WaitForRequest(ctx, location)
}

func (s *Service) getLatestNICPatchRequest(
	ctx context.Context, ms *scope.Machine, serverID string, nicID string,
) (*requestInfo, error) {
	return getMatchingRequest[sdk.Nic](
		ctx,
		s,
		http.MethodPatch,
		s.nicURL(ms, serverID, nicID),
	)
}

// nicHasIP returns true if the NIC contains the given IP address.
func nicHasIP(nic *sdk.Nic, expectedIP string) bool {
	ips := ptr.Deref(nic.GetProperties().GetIps(), []string{})
	return slices.Contains(ips, expectedIP)
}

func (*Service) nicName(m *infrav1.IonosCloudMachine) string {
	return fmt.Sprintf("k8s-nic-%s-%s", m.Namespace, m.Name)
}
