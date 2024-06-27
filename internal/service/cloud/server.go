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
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/google/uuid"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// ReconcileServer ensures the cluster server exist, creating one if it doesn't.
func (s *Service) ReconcileServer(ctx context.Context, ms *scope.Machine) (requeue bool, retErr error) {
	log := s.logger.WithName("ReconcileServer")

	log.V(4).Info("Reconciling server")

	secret, err := ms.GetBootstrapDataSecret(ctx, s.logger)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Secret not available yet.
			// Just log the error and resume reconciliation.
			log.Info("Bootstrap secret not available yet", "error", err)
			return false, nil
		}
		return true, fmt.Errorf("unexpected error when trying to get bootstrap secret: %w", err)
	}

	server, request, err := scopedFindResource(ctx, ms, s.getServer, s.getLatestServerCreationRequest)
	if err != nil {
		return false, err
	}
	if request != nil && request.isPending() {
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	if server == nil {
		// Server does not exist yet, create it
		log.V(4).Info("No server was found. Creating new server")
		if err := s.createServer(ctx, secret, ms); err != nil {
			return false, err
		}
		log.V(4).Info("Successfully initiated server creation")
		// If we reach this point, we want to requeue as the request is not processed yet,
		// and we will check for the status again later.
		return true, nil
	}

	requeue, err = s.ensureServerAvailable(ctx, ms, server)
	if requeue || err != nil {
		return requeue, err
	}

	// Attach the IPs from all NICs of the server to the status
	netInfo := &infrav1.MachineNetworkInfo{NICInfo: make([]infrav1.NICInfo, 0)}

	for _, nic := range ptr.Deref(server.GetEntities().GetNics().GetItems(), []sdk.Nic{}) {
		netInfo.NICInfo = append(netInfo.NICInfo, infrav1.NICInfo{
			IPv4Addresses: ptr.Deref(nic.GetProperties().GetIps(), []string{}),
			IPv6Addresses: ptr.Deref(nic.GetProperties().GetIpv6Ips(), []string{}),
			NetworkID:     ptr.Deref(nic.GetProperties().GetLan(), 0),
			Primary:       s.isPrimaryNIC(ms.IonosMachine, &nic),
		})
	}

	ms.IonosMachine.Status.MachineNetworkInfo = netInfo

	log.Info("Server is available", "serverID", ptr.Deref(server.GetId(), ""))
	// server exists and is available.
	return false, nil
}

// ReconcileServerDeletion ensures the server is deleted.
func (s *Service) ReconcileServerDeletion(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLANDeletion")

	server, request, err := scopedFindResource(ctx, ms, s.getServer, s.getLatestServerCreationRequest)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		ms.IonosMachine.SetCurrentRequest(http.MethodPost, request.status, request.location)
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if server == nil {
		ms.IonosMachine.DeleteCurrentRequest()
		return false, nil
	}

	request, err = s.getLatestServerDeletionRequest(ctx, ms.DatacenterID(), *server.Id)
	if err != nil {
		return false, err
	}

	if request != nil {
		if request.isPending() {
			ms.IonosMachine.SetCurrentRequest(http.MethodDelete, request.status, request.location)

			// We want to requeue and check again after some time
			log.Info("Deletion request is pending", "location", request.location)
			return true, nil
		}

		if request.isDone() {
			ms.IonosMachine.DeleteCurrentRequest()
			return false, nil
		}
	}

	err = s.deleteServer(ctx, ms, server)
	return err == nil, err
}

// FinalizeMachineProvisioning marks the machine as provisioned.
func (*Service) FinalizeMachineProvisioning(_ context.Context, ms *scope.Machine) (bool, error) {
	ms.IonosMachine.Status.Ready = true
	conditions.MarkTrue(ms.IonosMachine, infrav1.MachineProvisionedCondition)
	return false, nil
}

// isServerAvailable checks if the server is in state AVAILABLE.
func (s *Service) isServerAvailable(server *sdk.Server) bool {
	log := s.logger.WithName("isServerAvailable")
	if state := getState(server); !isAvailable(state) {
		log.Info("Server is not available yet", "state", state)
		return false
	}
	return true
}

// ensureServerAvailable checks the availability of the specified server.
func (s *Service) ensureServerAvailable(ctx context.Context, ms *scope.Machine, server *sdk.Server) (bool, error) {
	log := s.logger.WithName("ensureServerAvailable")

	// Check if the server is available
	if !s.isServerAvailable(server) {
		// Server is still provisioning, checking again later
		return true, nil
	}

	// Check the VM state; if not running, try to start it
	if vmState := getVMState(server); !isRunning(vmState) {
		err := s.startServer(ctx, ms, *server.Id)
		if err != nil {
			log.Error(err, "Failed to start the server")
			return true, err
		}
		// If we reach this point, we want to requeue as the request is not processed yet,
		// and we will check for the status again later.
		return true, nil
	}

	// Default return path when no conditions are met (server is available and running)
	return false, nil
}

// getServerByServerID checks if the IonosCloudMachine has a provider ID set.
// If it does, it will attempt to extract the server ID from the provider ID and
// query for the server in the cloud.
func (s *Service) getServerByServerID(ctx context.Context, datacenterID, serverID string) (*sdk.Server, error) {
	if serverID == "" {
		return nil, nil
	}

	// we expect the server ID to be a valid UUID
	if err := uuid.Validate(serverID); err != nil {
		return nil, fmt.Errorf("invalid server ID %s: %w", serverID, err)
	}

	depth := int32(2) // for getting the server and its NICs' properties
	server, err := s.apiWithDepth(depth).GetServer(ctx, datacenterID, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server %s in data center %s: %w", serverID, datacenterID, err)
	}

	return server, nil
}

func (s *Service) getServer(ctx context.Context, ms *scope.Machine) (*sdk.Server, error) {
	server, err := s.getServerByServerID(ctx, ms.DatacenterID(), ms.IonosMachine.ExtractServerID())
	// if the server was not found, we try to find it by listing all servers.
	if server != nil || ignoreNotFound(err) != nil {
		return server, err
	}

	// listing requires one more level of depth to for instance
	// retrieving the NIC properties.
	const listDepth = 3
	// without provider ID, we need to list all servers and see if
	// there is one with the expected name.
	serverList, listErr := s.apiWithDepth(listDepth).ListServers(ctx, ms.DatacenterID())
	if listErr != nil {
		return nil, fmt.Errorf("failed to list servers in data center %s: %w", ms.DatacenterID(), listErr)
	}

	items := ptr.Deref(serverList.Items, []sdk.Server{})
	// find servers with the expected name
	for _, server := range items {
		if server.HasProperties() && *server.Properties.Name == ms.IonosMachine.Name {
			// if the server was found, we set the provider ID and return it
			ms.SetProviderID(ptr.Deref(server.Id, ""))
			return &server, nil
		}
	}

	// if we still can't find a server we return the potential
	// initial not found error.
	return nil, err
}

func (s *Service) deleteServer(ctx context.Context, ms *scope.Machine, server *sdk.Server) error {
	log := s.logger.WithName("deleteServer")

	serverID := ptr.Deref(server.GetId(), "")

	deleteVolumes := ms.ClusterScope.IsDeleted()
	bootVolumeID := server.GetProperties().GetBootVolume().GetId()
	if !deleteVolumes && bootVolumeID != nil {
		// We need to make sure to only delete volumes if the cluster is being deleted.
		// If a node is being replaced, we only delete the boot volume and keep all other volumes.
		// The CSI will take care of re-attaching the existing volumes to the new node.

		requestLocation, err := s.ionosClient.DeleteVolume(ctx, ms.DatacenterID(), *bootVolumeID)
		if err != nil {
			return fmt.Errorf("failed to request boot volume deletion: %w", err)
		}

		ms.IonosMachine.SetCurrentRequest(http.MethodDelete, sdk.RequestStatusQueued, requestLocation)
		log.V(4).Info("Successfully requested for boot volume deletion", "location", requestLocation)
		return nil
	}

	log.V(4).Info("Deleting server", "serverID", serverID, "deleteVolumes", deleteVolumes)
	requestLocation, err := s.ionosClient.DeleteServer(ctx, ms.DatacenterID(), serverID, deleteVolumes)
	if err != nil {
		return fmt.Errorf("failed to request server deletion: %w", err)
	}

	log.Info("Successfully requested for server deletion", "location", requestLocation)
	ms.IonosMachine.SetCurrentRequest(http.MethodDelete, sdk.RequestStatusQueued, requestLocation)

	log.V(4).Info("Done deleting server")
	return nil
}

func (s *Service) startServer(ctx context.Context, ms *scope.Machine, serverID string) error {
	log := s.logger.WithName("startServer")

	log.V(4).Info("Starting server", "serverID", serverID)
	requestLocation, err := s.ionosClient.StartServer(ctx, ms.DatacenterID(), serverID)
	if err != nil {
		return fmt.Errorf("failed to request server start: %w", err)
	}

	log.Info("Successfully requested for server start", "location", requestLocation)
	ms.IonosMachine.SetCurrentRequest(http.MethodPost, sdk.RequestStatusQueued, requestLocation)

	return nil
}

func (s *Service) getLatestServerCreationRequest(ctx context.Context, ms *scope.Machine) (*requestInfo, error) {
	return getMatchingRequest(
		ctx,
		s,
		http.MethodPost,
		path.Join("datacenters", ms.DatacenterID(), "servers"),
		matchByName[*sdk.Server, *sdk.ServerProperties](ms.IonosMachine.Name),
	)
}

func (s *Service) getLatestServerDeletionRequest(
	ctx context.Context, datacenterID, serverID string,
) (*requestInfo, error) {
	return getMatchingRequest[sdk.Server](
		ctx,
		s,
		http.MethodDelete,
		path.Join("datacenters", datacenterID, "servers", serverID),
	)
}

func (s *Service) createServer(ctx context.Context, secret *corev1.Secret, ms *scope.Machine) error {
	log := s.logger.WithName("createServer")

	bootstrapData, exists := secret.Data["value"]
	if !exists {
		return errors.New("unable to obtain bootstrap data from secret")
	}

	lan, err := s.getLAN(ctx, ms)
	if err != nil {
		return err
	}

	lanID, err := strconv.ParseInt(ptr.Deref(lan.GetId(), "invalid"), 10, 32)
	if err != nil {
		return fmt.Errorf("unable to parse LAN ID: %w", err)
	}

	renderedData := s.renderUserData(ms, string(bootstrapData))
	copySpec := ms.IonosMachine.Spec.DeepCopy()
	entityParams := serverEntityParams{
		boostrapData: renderedData,
		machineSpec:  *copySpec,
		lanID:        int32(lanID),
	}

	server, requestLocation, err := s.ionosClient.CreateServer(
		ctx,
		ms.DatacenterID(),
		s.buildServerProperties(ms, copySpec),
		s.buildServerEntities(ms, entityParams),
	)
	if err != nil {
		return fmt.Errorf("failed to create server in data center %s: %w", ms.DatacenterID(), err)
	}

	log.Info("Successfully requested for server creation", "location", requestLocation)
	ms.IonosMachine.SetCurrentRequest(http.MethodPost, sdk.RequestStatusQueued, requestLocation)

	serverID := ptr.Deref(server.GetId(), "")
	if serverID == "" {
		return errors.New("server ID is empty")
	}

	// make sure to set the provider ID
	ms.SetProviderID(serverID)

	log.V(4).Info("Done creating server")
	return nil
}

// buildServerProperties returns the server properties for the expected cloud server resource.
func (*Service) buildServerProperties(
	ms *scope.Machine, machineSpec *infrav1.IonosCloudMachineSpec,
) sdk.ServerProperties {
	props := sdk.ServerProperties{
		AvailabilityZone: ptr.To(machineSpec.AvailabilityZone.String()),
		Cores:            &machineSpec.NumCores,
		Name:             ptr.To(ms.IonosMachine.Name),
		Ram:              &machineSpec.MemoryMB,
		CpuFamily:        machineSpec.CPUFamily,
		Type:             ptr.To(machineSpec.Type.String()),
	}

	return props
}

type serverEntityParams struct {
	boostrapData string
	machineSpec  infrav1.IonosCloudMachineSpec
	lanID        int32
}

// buildServerEntities returns the server entities for the expected cloud server resource.
func (s *Service) buildServerEntities(ms *scope.Machine, params serverEntityParams) sdk.ServerEntities {
	machineSpec := params.machineSpec
	bootVolume := sdk.Volume{
		Properties: &sdk.VolumeProperties{
			AvailabilityZone: ptr.To(machineSpec.Disk.AvailabilityZone.String()),
			Name:             ptr.To(s.volumeName(ms.IonosMachine)),
			Size:             ptr.To(float32(machineSpec.Disk.SizeGB)),
			Type:             ptr.To(machineSpec.Disk.DiskType.String()),
			UserData:         &params.boostrapData,
		},
	}

	if machineSpec.Disk.Image.ID != "" {
		bootVolume.Properties.Image = &machineSpec.Disk.Image.ID
	}

	serverVolumes := sdk.AttachedVolumes{
		Items: &[]sdk.Volume{bootVolume},
	}

	primaryNIC := sdk.Nic{
		Properties: &sdk.NicProperties{
			Lan:  &params.lanID,
			Name: ptr.To(s.nicName(ms.IonosMachine)),
		},
	}

	if ms.IonosMachine.Status.MachineNetworkInfo != nil {
		nicInfo := ms.IonosMachine.Status.MachineNetworkInfo.NICInfo[0]
		primaryNIC.Properties.Ips = ptr.To(nicInfo.IPv4Addresses)
		primaryNIC.Properties.Ipv6Ips = ptr.To(nicInfo.IPv6Addresses)
	}

	primaryNIC.Properties.Dhcp = ptr.To(true)

	// In case we want to retrieve a public IP from the DHCP, we need to
	// create a NIC with empty IP addresses and patch the NIC afterward.
	// To simplify the code we also follow this approach when using IP pools.
	serverNICs := sdk.Nics{
		Items: &[]sdk.Nic{
			primaryNIC,
		},
	}

	// Attach server to additional LANs if any.
	items := *serverNICs.Items

	for i, nw := range ms.IonosMachine.Spec.AdditionalNetworks {
		nic := sdk.Nic{
			Properties: &sdk.NicProperties{
				Lan: &nw.NetworkID,
			},
		}

		if ms.IonosMachine.Status.MachineNetworkInfo != nil {
			nicInfo := ms.IonosMachine.Status.MachineNetworkInfo.NICInfo[i+1]
			nic.Properties.Ips = ptr.To(nicInfo.IPv4Addresses)
			nic.Properties.Ipv6Ips = ptr.To(nicInfo.IPv6Addresses)
		}

		nic.Properties.Dhcp = ptr.To(true)

		items = append(items, nic)
	}

	serverNICs.Items = &items

	return sdk.ServerEntities{
		Nics:    &serverNICs,
		Volumes: &serverVolumes,
	}
}

func (*Service) renderUserData(ms *scope.Machine, input string) string {
	const bootCmdFormat = `bootcmd:
  - echo %[1]s > /etc/hostname
  - hostname %[1]s
`
	bootCmdString := fmt.Sprintf(bootCmdFormat, ms.IonosMachine.Name)
	input = fmt.Sprintf("%s\n%s", input, bootCmdString)

	return base64.StdEncoding.EncodeToString([]byte(input))
}

func (*Service) serversURL(datacenterID string) string {
	return path.Join("datacenters", datacenterID, "servers")
}

func (*Service) volumeName(m *infrav1.IonosCloudMachine) string {
	return "vol-" + m.Name
}
