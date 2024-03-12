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

	sdk "github.com/ionos-cloud/sdk-go/v6"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// lanName returns the name of the cluster LAN.
func (s *Service) lanName(c *clusterv1.Cluster) string {
	return fmt.Sprintf(
		"k8s-%s-%s",
		c.Namespace,
		c.Name)
}

func (s *Service) lanURL(datacenterID, id string) string {
	return path.Join("datacenters", datacenterID, "lans", id)
}

func (s *Service) lansURL(datacenterID string) string {
	return path.Join("datacenters", datacenterID, "lans")
}

// ReconcileLAN ensures the cluster LAN exist, creating one if it doesn't.
func (s *Service) ReconcileLAN(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLAN")

	lan, request, err := scopedFindResource(ctx, ms, s.getLAN, s.getLatestLANCreationRequest)
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
	if err := s.createLAN(ctx, ms); err != nil {
		return false, err
	}

	// after creating the LAN, we want to requeue and let the request be finished
	return true, nil
}

// ReconcileLANDeletion ensures there's no cluster LAN available, requesting for deletion (if no other resource
// uses it) otherwise.
func (s *Service) ReconcileLANDeletion(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLANDeletion")

	// Try to retrieve the cluster LAN or even check if it's currently still being created.
	lan, request, err := scopedFindResource(ctx, ms, s.getLAN, s.getLatestLANCreationRequest)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if lan == nil {
		err = s.removeLANPendingRequestFromCluster(ms)
		return err != nil, err
	}

	// If we found a LAN, we check if there is a deletion already in progress.
	request, err = s.getLatestLANDeletionRequest(ctx, ms, *lan.Id)
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
	err = s.deleteLAN(ctx, ms, *lan.Id)
	return err == nil, err
}

// getLAN tries to retrieve the cluster-related LAN in the data center.
func (s *Service) getLAN(ctx context.Context, ms *scope.Machine) (*sdk.Lan, error) {
	// check if the LAN exists
	depth := int32(2) // for listing the LANs with their number of NICs
	lans, err := s.apiWithDepth(depth).ListLANs(ctx, ms.DatacenterID())
	if err != nil {
		return nil, fmt.Errorf("could not list LANs in data center %s: %w", ms.DatacenterID(), err)
	}

	var (
		expectedName = s.lanName(ms.ClusterScope.Cluster)
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

func (s *Service) createLAN(ctx context.Context, ms *scope.Machine) error {
	log := s.logger.WithName("createLAN")

	requestPath, err := s.ionosClient.CreateLAN(ctx, ms.DatacenterID(), sdk.LanPropertiesPost{
		Name:   ptr.To(s.lanName(ms.ClusterScope.Cluster)),
		Public: ptr.To(true),
	})
	if err != nil {
		return fmt.Errorf("unable to create LAN in data center %s: %w", ms.DatacenterID(), err)
	}

	ms.ClusterScope.IonosCluster.SetCurrentRequestByDatacenter(ms.DatacenterID(), infrav1.ProvisioningRequest{
		Method:      http.MethodPost,
		RequestPath: requestPath,
		State:       sdk.RequestStatusQueued,
	})

	err = ms.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch the cluster: %w", err)
	}

	log.Info("Successfully requested for LAN creation", "requestPath", requestPath)

	return nil
}

func (s *Service) deleteLAN(ctx context.Context, ms *scope.Machine, lanID string) error {
	log := s.logger.WithName("deleteLAN")

	requestPath, err := s.ionosClient.DeleteLAN(ctx, ms.DatacenterID(), lanID)
	if err != nil {
		return fmt.Errorf("unable to request LAN deletion in data center: %w", err)
	}

	if ms.ClusterScope.IonosCluster.Status.CurrentRequestByDatacenter == nil {
		ms.ClusterScope.IonosCluster.Status.CurrentRequestByDatacenter = make(map[string]infrav1.ProvisioningRequest)
	}

	ms.ClusterScope.IonosCluster.SetCurrentRequestByDatacenter(ms.DatacenterID(), infrav1.ProvisioningRequest{
		Method:      http.MethodDelete,
		RequestPath: requestPath,
		State:       sdk.RequestStatusQueued,
	})

	err = ms.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch cluster: %w", err)
	}
	log.WithValues("requestPath", requestPath).Info("Successfully requested for LAN deletion")
	return nil
}

func (s *Service) getLatestLANRequestByMethod(
	ctx context.Context, method, url string, matchers ...matcherFunc[*sdk.Lan],
) (*requestInfo, error) {
	return getMatchingRequest[sdk.Lan](
		ctx,
		s,
		method,
		url,
		matchers...,
	)
}

func (s *Service) getLatestLANCreationRequest(ctx context.Context, ms *scope.Machine) (*requestInfo, error) {
	return s.getLatestLANRequestByMethod(
		ctx,
		http.MethodPost,
		s.lansURL(ms.DatacenterID()),
		matchByName[*sdk.Lan, *sdk.LanProperties](s.lanName(ms.ClusterScope.Cluster)))
}

func (s *Service) getLatestLANDeletionRequest(ctx context.Context, ms *scope.Machine, lanID string) (*requestInfo, error) {
	return s.getLatestLANRequestByMethod(ctx, http.MethodDelete, s.lanURL(ms.DatacenterID(), lanID))
}

func (s *Service) getLatestLANPatchRequest(ctx context.Context, ms *scope.Machine, lanID string) (*requestInfo, error) {
	return s.getLatestLANRequestByMethod(ctx, http.MethodPatch, s.lanURL(ms.DatacenterID(), lanID))
}

func (s *Service) removeLANPendingRequestFromCluster(ms *scope.Machine) error {
	ms.ClusterScope.IonosCluster.DeleteCurrentRequestByDatacenter(ms.DatacenterID())
	if err := ms.ClusterScope.PatchObject(); err != nil {
		return fmt.Errorf("could not remove stale LAN pending request from cluster: %w", err)
	}
	return nil
}

// ReconcileIPFailover ensures that the control plane nodes will attach the endpoint IP to their primary
// NIC and add the NIC to the failover group of the public LAN.
// This is needed for kube-vip in order to set up HA control planes.
//
// If we want to support private clusters in the future, this will require some adjustments.
func (s *Service) ReconcileIPFailover(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileIPFailover")

	if !util.IsControlPlaneMachine(ms.Machine) {
		log.V(4).Info("Failover is only applied to control plane machines.")
		return false, nil
	}

	endpointIP := ms.ClusterScope.GetControlPlaneEndpoint().Host
	nic, err := s.reconcileNICConfig(ctx, ms, endpointIP)
	if err != nil {
		return false, err
	}

	nicID := ptr.Deref(nic.GetId(), "")
	return s.reconcileIPFailoverGroup(ctx, ms, nicID, endpointIP)
}

func (s *Service) ReconcileIPFailoverDeletion(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileIPFailoverDeletion")

	if !util.IsControlPlaneMachine(ms.Machine) {
		log.V(4).Info("Failover is only applied to control plane machines.")
		return false, nil
	}

	server, err := s.getServer(ctx, ms)
	if err != nil {
		if isNotFound(err) {
			log.Info("Server was not found or already deleted.")
			return false, nil
		}
		log.Error(err, "Unable to retrieve server")
		return false, err
	}

	nic, err := s.findPrimaryNIC(ms, server)
	if err != nil {
		log.Error(err, "Unable to find primary NIC on server")
		return false, nil
	}

	nicID := ptr.Deref(nic.GetId(), "")
	return s.removeNICFromFailoverGroup(ctx, ms, nicID)
}

// reconcileIPFailoverGroup ensures that the public LAN has a failover group with the NIC for this machine and
// the endpoint IP address. It further ensures that NICs from additional control plane machines are also added.
func (s *Service) reconcileIPFailoverGroup(
	ctx context.Context, ms *scope.Machine, nicID, endpointIP string,
) (requeue bool, err error) {
	log := s.logger.WithName("reconcileIPFailoverGroup")
	if nicID == "" {
		return false, errors.New("nicID is empty")
	}

	// Add the NIC to the failover group of the LAN

	lan, failoverConfig := &sdk.Lan{}, &[]sdk.IPFailover{}
	if requeue, err := s.retrieveLANFailoverConfig(ctx, ms, lan, failoverConfig); err != nil || requeue {
		return requeue, err
	}

	ipFailoverConfig := *failoverConfig
	lanID := *lan.GetId()

	for index, entry := range ipFailoverConfig {
		nicUUID := ptr.Deref(entry.GetNicUuid(), "undefined")
		ip := ptr.Deref(entry.GetIp(), "undefined")
		if ip == endpointIP && nicUUID != nicID {
			log.V(4).Info("Another NIC is already defined in this failover group. Skipping further actions")
			return false, nil
		}

		if nicUUID != nicID {
			continue
		}

		// Make sure the NIC is in the failover group with the correct IP address.
		log.V(4).Info("Found NIC in failover group", "nicID", nicID)
		if ip == endpointIP {
			log.V(4).Info("NIC is already in the failover group with the correct IP address")
			return false, nil
		}

		log.Info("NIC is already in the failover group but with a different IP address", "currentIP", ip, "expectedIP", endpointIP)
		// The IP address of the NIC is different. We need to update the failover group.
		entry.Ip = ptr.To(endpointIP)
		ipFailoverConfig[index] = entry

		err := s.patchLAN(ctx, ms, lanID, sdk.LanProperties{IpFailover: &ipFailoverConfig})
		return true, err
	}

	// NIC was not found in failover group. We need to add it.
	ipFailoverConfig = append(ipFailoverConfig, sdk.IPFailover{
		Ip:      ptr.To(endpointIP),
		NicUuid: ptr.To(nicID),
	})

	props := sdk.LanProperties{IpFailover: &ipFailoverConfig}
	log.V(4).Info("Patching LAN failover group to add NIC", "nicID", nicID, "endpointIP", endpointIP)

	err = s.patchLAN(ctx, ms, lanID, props)
	return true, err
}

func (s *Service) removeNICFromFailoverGroup(ctx context.Context, ms *scope.Machine, nicID string) (requeue bool, err error) {
	log := s.logger.WithName("removeNICFromFailoverGroup")

	lan, failoverConfig := &sdk.Lan{}, &[]sdk.IPFailover{}
	if requeue, err := s.retrieveLANFailoverConfig(ctx, ms, lan, failoverConfig); err != nil || requeue {
		return requeue, err
	}

	ipFailoverConfig := *failoverConfig
	lanID := *lan.GetId()

	index := slices.IndexFunc(ipFailoverConfig, func(failover sdk.IPFailover) bool {
		return ptr.Deref(failover.GetNicUuid(), "undefined") == nicID
	})

	if index < 0 {
		log.V(4).Info("NIC not found in failover group. No action required.")
		return false, nil
	}

	// found the NIC, remove it from the failover group
	log.V(4).Info("Found NIC in failover group", "nicID", nicID)
	ipFailoverConfig = append(ipFailoverConfig[:index], ipFailoverConfig[index+1:]...)
	props := sdk.LanProperties{IpFailover: &ipFailoverConfig}

	log.V(4).Info("Patching LAN failover group to remove NIC", "nicID", nicID)
	return true, s.patchLAN(ctx, ms, lanID, props)
}

func (s *Service) retrieveLANFailoverConfig(
	ctx context.Context, ms *scope.Machine, lan *sdk.Lan, failoverConfig *[]sdk.IPFailover,
) (requeue bool, err error) {
	log := s.logger.WithName("retrieveLANFailoverConfig")

	gotLAN, err := s.getLAN(ctx, ms)
	if err != nil {
		return true, err
	}
	*lan = ptr.Deref(gotLAN, sdk.Lan{})

	lanID := ptr.Deref(lan.GetId(), "")
	if pending, err := s.isLANPatchPending(ctx, lanID, ms); pending || err != nil {
		return pending, err
	}

	log.V(4).Info("Checking failover group of LAN", "id", lanID)
	*failoverConfig = ptr.Deref(lan.GetProperties().GetIpFailover(), []sdk.IPFailover{})
	return false, nil
}

func (s *Service) isLANPatchPending(ctx context.Context, lanID string, ms *scope.Machine) (pending bool, err error) {
	ri, err := s.getLatestLANPatchRequest(ctx, ms, lanID)
	if err != nil {
		return false, fmt.Errorf("unable to check for pending LAN patch request: %w", err)
	}

	return ri != nil && ri.isPending(), nil
}

func (s *Service) patchLAN(ctx context.Context, ms *scope.Machine, lanID string, properties sdk.LanProperties) error {
	log := s.logger.WithName("patchLAN")
	log.Info("Patching LAN", "id", lanID)

	location, err := s.ionosClient.PatchLAN(ctx, ms.DatacenterID(), lanID, properties)
	if err != nil {
		return fmt.Errorf("failed to patch LAN %s: %w", lanID, err)
	}
	ms.IonosMachine.SetCurrentRequest(http.MethodPatch, sdk.RequestStatusQueued, location)
	return nil
}
