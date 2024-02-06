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
	"sigs.k8s.io/cluster-api/util"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// lanName returns the name of the cluster LAN.
func (s *Service) lanName() string {
	return fmt.Sprintf(
		"k8s-%s-%s",
		s.scope.ClusterScope.Cluster.Namespace,
		s.scope.ClusterScope.Cluster.Name)
}

func (s *Service) lanURL(id string) string {
	return path.Join("datacenters", s.datacenterID(), "lans", id)
}

func (s *Service) lansURL() string {
	return path.Join("datacenters", s.datacenterID(), "lans")
}

// ReconcileLAN ensures the cluster LAN exist, creating one if it doesn't.
func (s *Service) ReconcileLAN() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileLAN")

	lan, request, err := findResource(s.getLAN, s.getLatestLANCreationRequest)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	if lan != nil {
		if state := getState(lan); !isAvailable(state) {
			log.Info("LAN is not available yet", "state", state)
			return true, nil
		}
		return false, nil
	}

	log.V(4).Info("No LAN was found. Creating new LAN")
	if err := s.createLAN(); err != nil {
		return false, err
	}

	// after creating the LAN, we want to requeue and let the request be finished
	return true, nil
}

// ReconcileLANDeletion ensures there's no cluster LAN available, requesting for deletion (if no other resource
// uses it) otherwise.
func (s *Service) ReconcileLANDeletion() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileLANDeletion")

	// Try to retrieve the cluster LAN or even check if it's currently still being created.
	lan, request, err := findResource(s.getLAN, s.getLatestLANCreationRequest)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if lan == nil {
		err = s.removeLANPendingRequestFromCluster()
		return err != nil, err
	}

	// If we found a LAN, we check if there is a deletion already in progress.
	request, err = s.getLatestLANDeletionRequest(*lan.Id)
	if err != nil {
		return false, err
	}
	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Deletion request is pending", "location", request.location)
		return true, nil
	}

	if len(*lan.Entities.Nics.Items) > 0 {
		log.Info("The cluster LAN is still being used by another resource. Skipping deletion.")
		return false, nil
	}

	// Request for LAN deletion
	err = s.deleteLAN(*lan.Id)
	return err == nil, err
}

// getLAN tries to retrieve the cluster-related LAN in the data center.
func (s *Service) getLAN() (*sdk.Lan, error) {
	// check if the LAN exists
	lans, err := s.api().ListLANs(s.ctx, s.datacenterID())
	if err != nil {
		return nil, fmt.Errorf("could not list LANs in data center %s: %w", s.datacenterID(), err)
	}

	var (
		expectedName = s.lanName()
		lanCount     = 0
		foundLAN     *sdk.Lan
	)

	for _, l := range *lans.Items {
		if l.Properties.HasName() && *l.Properties.Name == expectedName {
			l := l
			foundLAN = &l
			lanCount++
		}

		// If there are multiple LANs with the same name, we should return an error.
		// Our logic won't be able to proceed as we cannot select the correct LAN.
		if lanCount > 1 {
			return nil, fmt.Errorf("found multiple LANs with the name: %s", expectedName)
		}
	}

	return foundLAN, nil
}

func (s *Service) createLAN() error {
	log := s.scope.Logger.WithName("createLAN")

	requestPath, err := s.api().CreateLAN(s.ctx, s.datacenterID(), sdk.LanPropertiesPost{
		Name:   ptr.To(s.lanName()),
		Public: ptr.To(true),
	})
	if err != nil {
		return fmt.Errorf("unable to create LAN in data center %s: %w", s.datacenterID(), err)
	}

	s.scope.ClusterScope.IonosCluster.SetCurrentRequest(
		s.datacenterID(),
		infrav1.NewQueuedRequest(http.MethodPost, requestPath),
	)

	err = s.scope.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch the cluster: %w", err)
	}

	log.Info("Successfully requested for LAN creation", "requestPath", requestPath)

	return nil
}

func (s *Service) deleteLAN(lanID string) error {
	log := s.scope.Logger.WithName("deleteLAN")

	requestPath, err := s.api().DeleteLAN(s.ctx, s.datacenterID(), lanID)
	if err != nil {
		return fmt.Errorf("unable to request LAN deletion in data center: %w", err)
	}

	if s.scope.ClusterScope.IonosCluster.Status.CurrentRequestByDatacenter == nil {
		s.scope.ClusterScope.IonosCluster.Status.CurrentRequestByDatacenter = make(map[string]infrav1.ProvisioningRequest)
	}
	s.scope.ClusterScope.IonosCluster.Status.CurrentRequestByDatacenter[s.datacenterID()] = infrav1.ProvisioningRequest{
		Method:      http.MethodDelete,
		RequestPath: requestPath,
		State:       infrav1.RequestStatusQueued,
	}

	err = s.scope.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch cluster: %w", err)
	}
	log.WithValues("requestPath", requestPath).Info("Successfully requested for LAN deletion")
	return nil
}

func (s *Service) getLatestLANCreationRequest() (*requestInfo, error) {
	return getMatchingRequest(
		s,
		http.MethodPost,
		s.lansURL(),
		matchByName[*sdk.Lan, *sdk.LanProperties](s.lanName()),
	)
}

func (s *Service) getLatestLANDeletionRequest(lanID string) (*requestInfo, error) {
	return getMatchingRequest[sdk.Lan](
		s,
		http.MethodDelete,
		path.Join(s.lansURL(), lanID),
	)
}

func (s *Service) removeLANPendingRequestFromCluster() error {
	s.scope.ClusterScope.IonosCluster.DeleteCurrentRequest(s.datacenterID())
	if err := s.scope.ClusterScope.PatchObject(); err != nil {
		return fmt.Errorf("could not remove stale LAN pending request from cluster: %w", err)
	}
	return nil
}

// checkPrimaryNIC ensures the primary NIC of the server contains the endpoint IP address.
// This is needed for KubeVIP in order to set up control plane load balancing.
//
// If we want to support private clusters in the future, this will require some adjustments.
func (s *Service) checkPrimaryNIC(server *sdk.Server) (bool, error) {
	log := s.scope.Logger.WithName("checkPrimaryNIC")

	if !util.IsControlPlaneMachine(s.scope.Machine) {
		log.V(4).Info("Machine is a worker node and doesn't need a second IP address")
		return false, nil
	}

	serverNICs := ptr.Deref(server.GetEntities().GetNics().GetItems(), []sdk.Nic{})
	for _, nic := range serverNICs {
		// if the name doesn't match, we can continue
		if name := ptr.Deref(nic.GetProperties().GetName(), ""); name != s.serverName() {
			continue
		}

		log.V(4).Info("Found primary NIC", "name", s.serverName())
		ips := ptr.Deref(nic.GetProperties().GetIps(), []string{})
		for _, ip := range ips {
			if ip == s.scope.ClusterScope.GetControlPlaneEndpoint().Host {
				log.V(4).Info("Primary NIC contains endpoint IP address")
				return false, nil
			}
		}

		// Patch the NIC and include the complete set of IP addresses
		// The primary IP must be in the first position.
		patchSet := append(ips, s.scope.ClusterScope.GetControlPlaneEndpoint().Host)

		serverID := ptr.Deref(server.GetId(), "")
		nicProperties := sdk.NicProperties{Ips: &patchSet}

		err := s.patchNIC(serverID, nic, nicProperties)
		return true, err
	}

	return true, fmt.Errorf("could not find primary NIC with name %s", s.serverName())
}

func (s *Service) patchNIC(serverID string, nic sdk.Nic, props sdk.NicProperties) error {
	log := s.scope.Logger.WithName("patchNIC")

	nicID := ptr.Deref(nic.GetId(), "")
	log.V(4).Info("Patching NIC", "id", nicID)

	location, err := s.api().PatchNIC(s.ctx, s.datacenterID(), serverID, nicID, props)
	if err != nil {
		return fmt.Errorf("failed to patch NIC %s: %w", nicID, err)
	}

	s.scope.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodPatch, location))

	log.V(4).Info("Successfully patched NIC", "location", location)
	return nil
}
