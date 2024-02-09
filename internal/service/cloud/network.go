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
	"errors"
	"fmt"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/util/sets"
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

func (s *Service) getLatestLANPatchRequest(lanID string) (*requestInfo, error) {
	return getMatchingRequest[sdk.Lan](
		s,
		http.MethodPatch,
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

// ReconcileIPFailover ensures, that the control plane nodes, will attach the endpoint IP to their primary
// NIC and add the NIC to the failover group of the public LAN.
// This is needed for KubeVIP in order to set up control plane load balancing.
//
// If we want to support private clusters in the future, this will require some adjustments.
func (s *Service) ReconcileIPFailover() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileIPFailover")

	if !util.IsControlPlaneMachine(s.scope.Machine) {
		log.V(4).Info("Failover is only applied to control-plane machines.")
		return false, nil
	}

	endpointIP := s.scope.ClusterScope.GetControlPlaneEndpoint().Host
	nic, err := s.reconcileNICConfig(endpointIP)
	if err != nil {
		return false, err
	}

	nicID := ptr.Deref(nic.GetId(), "")
	return s.reconcileIPFailoverGroup(nicID, endpointIP)
}

func (s *Service) ReconcileIPFailoverDeletion() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileIPFailoverDeletion")

	if !util.IsControlPlaneMachine(s.scope.Machine) {
		log.V(4).Info("Failover is only applied to control-plane machines.")
		return false, nil
	}

	server, err := s.getServer()
	if err != nil {
		if isNotFound(err) {
			log.Info("Server not found. Cannot determine NIC to remove from failover group.")
			return false, nil
		}
		return false, err
	}

	nic, err := s.findPrimaryNIC(server)
	if err != nil {
		log.Info("Unable to find primary NIC on server", "error", err)
		return false, nil
	}

	nicID := ptr.Deref(nic.GetId(), "")
	return s.removeNICFromFailoverGroup(nicID)
}

// reconcileNICConfig ensures, that the primary NIC contains the endpoint IP address.
func (s *Service) reconcileNICConfig(endpointIP string) (*sdk.Nic, error) {
	log := s.scope.Logger.WithName("reconcileNICConfig")

	log.V(4).Info("Reconciling NIC config")
	// Get current state of the server
	server, err := s.getServer()
	if err != nil {
		return nil, err
	}

	// Find the primary NIC and ensure, that the endpoint IP address is added to the NIC.
	nic, err := s.findPrimaryNIC(server)
	if err != nil {
		return nil, err
	}

	// if the NIC already contains the endpoint IP address, we can return
	if nicHasIP(nic, endpointIP) {
		log.V(4).Info("Primary NIC contains endpoint IP address. Reconcile successful.")
		return nic, nil
	}

	log.V(4).Info("Unable to find endpoint IP address in primary NIC. Patching NIC.")
	nicIPs := ptr.Deref(nic.GetProperties().GetIps(), []string{})
	nicIPs = append(nicIPs, endpointIP)

	if err := s.patchNIC(ptr.Deref(server.GetId(), ""), nic, sdk.NicProperties{Ips: &nicIPs}); err != nil {
		return nil, err
	}

	log.V(4).Info("Successfully Patched NIC. Finished reconciling NIC config.")
	// As we are waiting for the request to finish this time, we can assume, that the request was successful
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

// reconcileIPFailoverGroup ensures that the public LAN has a failover group with the NIC for this machine and
// the endpoint IP address. It further ensures, that NICs from additional control plane machines are also added.
func (s *Service) reconcileIPFailoverGroup(nicID, endpointIP string) (requeue bool, err error) {
	log := s.scope.Logger.WithName("reconcileIPFailoverGroup")
	if nicID == "" {
		return false, errors.New("nicID is empty")
	}

	// Add the NIC to the failover group of the LAN
	lan, err := s.getLAN()
	if err != nil {
		return false, err
	}

	// Check if the LAN is currently being updated
	lanID := ptr.Deref(lan.GetId(), "")
	if pending, err := s.isLANPatchPending(lanID); pending || err != nil {
		return pending, err
	}

	log.V(4).Info("Checking failover group of LAN", "id", lanID)
	ipFailoverConfig := ptr.Deref(lan.GetProperties().GetIpFailover(), []sdk.IPFailover{})

	for index, entry := range ipFailoverConfig {
		nicUUID := ptr.Deref(entry.GetNicUuid(), "undefined")
		// we need one nic to be in the failover? I think???
		ip := ptr.Deref(entry.GetIp(), "undefined")
		if ip == endpointIP && nicUUID != nicID {
			log.V(4).Info("Another NIC is already defined in this failover group. Skipping further actions")
			return false, nil
		}

		if nicUUID != nicID {
			continue
		}

		// Make sure the NIC is in the failover group with the correct IP address.
		log.V(4).Info("Found NIC in failover group", "nicUUID", nicUUID)
		if ip == endpointIP {
			log.V(4).Info("NIC is already in the failover group with the correct IP address")
			return false, nil
		}

		log.Info("NIC is already in the failover group but with a different IP address", "currentIP", ip, "expectedIP", endpointIP)
		// The IP address of the NIC is different. We need to update the failover group.
		entry.Ip = ptr.To(endpointIP)
		ipFailoverConfig[index] = entry

		err := s.patchLAN(lanID, sdk.LanProperties{IpFailover: &ipFailoverConfig})
		return true, err
	}

	// NIC was not found in failover group. We need to add it.
	ipFailoverConfig = append(ipFailoverConfig, sdk.IPFailover{
		Ip:      ptr.To(endpointIP),
		NicUuid: ptr.To(nicID),
	})

	props := sdk.LanProperties{IpFailover: &ipFailoverConfig}
	log.V(4).Info("Patching LAN failover group to add NIC", "nicID", nicID, "endpointIP", endpointIP)

	err = s.patchLAN(*lan.GetId(), props)
	return true, err
}

func (s *Service) removeNICFromFailoverGroup(nicID string) (requeue bool, err error) {
	log := s.scope.Logger.WithName("removeNICFromFailoverGroup")

	lan, err := s.getLAN()
	if err != nil {
		return false, err
	}

	// check if there is a pending patch request for the LAN
	lanID := ptr.Deref(lan.GetId(), "")
	if pending, err := s.isLANPatchPending(lanID); pending || err != nil {
		return pending, err
	}

	log.V(4).Info("Checking failover group of LAN", "id", lanID)
	ipFailoverConfig := ptr.Deref(lan.GetProperties().GetIpFailover(), []sdk.IPFailover{})

	for index, entry := range ipFailoverConfig {
		if ptr.Deref(entry.GetNicUuid(), "undefined") != nicID {
			continue
		}

		log.V(4).Info("Found NIC in failover group", "nicUUID", nicID)
		ipFailoverConfig = append(ipFailoverConfig[:index], ipFailoverConfig[index+1:]...)
		props := sdk.LanProperties{IpFailover: &ipFailoverConfig}

		log.V(4).Info("Patching LAN failover group to remove NIC", "nicID", nicID)
		err := s.patchLAN(lanID, props)
		return true, err
	}

	log.V(4).Info("NIC not found in failover group. No action required.")
	return false, nil
}

func (s *Service) isLANPatchPending(lanID string) (pending bool, err error) {
	ri, err := s.getLatestLANPatchRequest(lanID)
	if err != nil {
		return false, fmt.Errorf("unable to check for pending LAN patch request: %w", err)
	}

	return ri != nil && ri.isPending(), nil
}

func (s *Service) patchLAN(lanID string, properties sdk.LanProperties) error {
	log := s.scope.Logger.WithName("patchLAN")
	log.Info("Patching LAN", "id", lanID)

	location, err := s.api().PatchLAN(s.ctx, s.datacenterID(), lanID, properties)
	if err != nil {
		return fmt.Errorf("failed to patch LAN %s: %w", lanID, err)
	}

	s.scope.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodPatch, location))

	return nil
}

// nicHasIP returns true if the NIC contains the given IP address.
func nicHasIP(nic *sdk.Nic, expectedIP string) bool {
	ips := ptr.Deref(nic.GetProperties().GetIps(), []string{})
	ipSet := sets.New(ips...)
	return ipSet.Has(expectedIP)
}
