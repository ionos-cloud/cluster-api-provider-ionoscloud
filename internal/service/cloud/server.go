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
	"strings"

	"github.com/google/uuid"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

// ReconcileServer ensures the cluster server exist, creating one if it doesn't.
func (s *Service) ReconcileServer(ctx context.Context, cs *scope.ClusterScope, ms *scope.MachineScope) (requeue bool, retErr error) {
	log := s.logger.WithName("ReconcileServer")

	log.V(4).Info("Reconciling server")

	secret, err := s.GetBootstrapDataSecret(ctx, ms)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// secret not available yet
			// just log the error and resume reconciliation.
			log.Info("Bootstrap secret not available yet", "error", err)
			return false, nil
		}
		return true, fmt.Errorf("unexpected error when trying to get bootstrap secret: %w", err)
	}

	server, request, err := findResource(ctx, s.getServer(ms), s.getLatestServerCreationRequest(ms))
	if err != nil {
		return false, err
	}
	if request != nil && request.isPending() {
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	if server != nil {
		// we have to make sure that after the NIC was created, the endpoint IP must be added
		// as a secondary IP address
		if shouldRequeue, err := s.checkPrimaryNIC(ctx, cs, ms, server); shouldRequeue || err != nil {
			return shouldRequeue, err
		}

		if !s.isServerAvailable(server) {
			// server is still provisioning, checking again later
			return true, nil
		}

		log.Info("Server is available", "serverID", ptr.Deref(server.GetId(), ""))
		// server exists and is available.
		return s.finalizeServerProvisioning(ms)
	}

	// server does not exist yet, create it
	log.V(4).Info("No server was found. Creating new server")
	if err := s.createServer(ctx, cs, ms, secret); err != nil {
		return false, err
	}

	log.V(4).Info("successfully finished reconciling server")
	// If we reach this point, we want to requeue as the request is not processed yet,
	// and we will check for the status again later.
	return true, nil
}

// ReconcileServerDeletion ensures the server is deleted.
func (s *Service) ReconcileServerDeletion(ctx context.Context, _ *scope.ClusterScope, ms *scope.MachineScope) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLANDeletion")

	server, request, err := findResource(ctx, s.getServer(ms), s.getLatestServerCreationRequest(ms))
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		log.Info("Creation request is pending", "location", request.location)
		return true, nil
	}

	if server == nil {
		ms.IonosMachine.Status.CurrentRequest = nil
		return false, nil
	}

	// TODO(lubedacht) add handling for volume deletion
	// if requeue, err := s.ensureVolumeDeleted(server); requeue || err != nil {
	//	return requeue, err
	// }

	request, err = s.getLatestServerDeletionRequest(ctx, ms, *server.Id)
	if err != nil {
		return false, err
	}

	if request != nil && request.isPending() {
		// We want to requeue and check again after some time
		log.Info("Deletion request is pending", "location", request.location)
		return true, nil
	}

	// TODO(lubedacht) Delete Volume before server deletion
	// 	if the bool flag in the API doesn't work
	// err = s.ensureVolumeDeleted(*server.Id)
	err = s.deleteServer(ctx, ms, *server.Id)
	return err == nil, err
}

// TODO(lubedacht)check if we need to manually delete volumes
// func (s *Service) ensureVolumeDeleted(server *sdk.Server) (bool, error) {
//	log := s.scope.logger.WithName("ReconcileLANDeletion")
//	log.Info("Ensuring that server volumes will be deleted", "serverID", ptr.Deref(server.GetId(), ""))
//
//	volumes := ptr.Deref(server.GetEntities().GetVolumes().GetItems(), []sdk.Volume{})
//	if len(volumes) == 0 {
//		return false, nil
//	}
//
//	for _, vol := range volumes {
//		location, err := s.api().DeleteVolume(s.ctx, s.datacenterID(), ptr.Deref(vol.GetId(), ""))
//		if err != nil {
//			log.Error(err, "Unexpected error occurred during Volume deletion", "volumeID", ptr.Deref(vol.GetId(), ""))
//			return false, fmt.Errorf("failed to request volume deletion: %w", err)
//		}
//
//		if location == "" {
//			continue
//		}
//	}
//
//	return false, nil
// }

func (s *Service) finalizeServerProvisioning(ms *scope.MachineScope) (bool, error) {
	conditions.MarkTrue(ms.IonosMachine, clusterv1.ReadyCondition)
	conditions.MarkTrue(ms.IonosMachine, infrav1.MachineProvisionedCondition)
	return false, nil
}

func (s *Service) isServerAvailable(server *sdk.Server) bool {
	log := s.logger.WithName("isServerAvailable")
	if state := getState(server); !isAvailable(state) {
		log.Info("Server is not available yet", "state", state)
		return false
	}

	// TODO ensure server is started
	if vmState := getVMState(server); !isRunning(vmState) {
		log.Info("Server is not running yet", "state", vmState)
		// TODO start server
		return false
	}

	return true
}

// getServerByProviderID checks if the IonosCloudMachine has a provider ID set.
// If it does, it will attempt to extract the server ID from the provider ID and
// query for the server in the cloud.
func (s *Service) getServerByProviderID(ctx context.Context, ms *scope.MachineScope) (*sdk.Server, error) {
	// first we check if the provider ID is set
	if !ptr.IsNullOrDefault(ms.IonosMachine.Spec.ProviderID) {
		serverID := ms.IonosMachine.ExtractServerID()
		// we expect the server ID to be a valid UUID
		if err := uuid.Validate(serverID); err != nil {
			return nil, fmt.Errorf("invalid server ID %s: %w", serverID, err)
		}

		server, err := s.cloud.GetServer(ctx, s.datacenterID(nil), serverID)
		// if the server was not found, we will continue, as the request might not
		// have been completed yet
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("failed to get server %s in data center %s: %w", serverID, s.datacenterID(nil), err)
		}

		// if the server was found, we return it
		if server != nil {
			return server, nil
		}
	}

	// couldn't find a server
	return nil, nil
}

// getServer looks for the server in the data center.
func (s *Service) getServer(ms *scope.MachineScope) listAndFilterFunc[sdk.Server] {
	return func(ctx context.Context) (*sdk.Server, error) {
		server, err := s.getServerByProviderID(ctx, ms)
		if server != nil || err != nil {
			return server, err
		}

		// without provider ID, we need to list all servers and see if
		// there is one with the expected name.
		serverList, err := s.cloud.ListServers(ctx, s.datacenterID(nil))
		if err != nil {
			return nil, fmt.Errorf("failed to list servers in data center %s: %w", s.datacenterID(nil), err)
		}

		items := ptr.Deref(serverList.Items, []sdk.Server{})
		// find servers with the expected name
		for _, server := range items {
			if server.HasProperties() && *server.Properties.Name == s.serverName(ms) {
				// if the server was found, we set the provider ID and return it
				ms.SetProviderID(ptr.Deref(server.Id, ""))
				return &server, nil
			}
		}

		return nil, nil
	}
}

func (s *Service) deleteServer(ctx context.Context, ms *scope.MachineScope, serverID string) error {
	log := s.logger.WithName("deleteServer")

	log.V(4).Info("Deleting server", "serverID", serverID)
	requestLocation, err := s.cloud.DeleteServer(ctx, s.datacenterID(nil), serverID)
	if err != nil {
		return fmt.Errorf("failed to request server deletion: %w", err)
	}

	log.Info("Successfully requested for server deletion", "location", requestLocation)
	ms.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodDelete, requestLocation))

	log.V(4).Info("Done deleting server")
	return nil
}

func (s *Service) getLatestServerCreationRequest(ms *scope.MachineScope) checkQueueFunc {
	return func(ctx context.Context) (*requestInfo, error) {
		return getMatchingRequest(
			ctx,
			s,
			http.MethodPost,
			path.Join("datacenters", s.datacenterID(nil), "servers"),
			matchByName[*sdk.Server, *sdk.ServerProperties](s.serverName(ms)),
		)
	}
}

func (s *Service) getLatestServerDeletionRequest(ctx context.Context, ms *scope.MachineScope, serverID string) (*requestInfo, error) {
	return getMatchingRequest(
		ctx,
		s,
		http.MethodDelete,
		path.Join("datacenters", s.datacenterID(nil), "servers", serverID),
		matchByName[*sdk.Server, *sdk.ServerProperties](s.serverName(ms)),
	)
}

func (s *Service) createServer(ctx context.Context, cs *scope.ClusterScope, ms *scope.MachineScope, secret *corev1.Secret) error {
	log := s.logger.WithName("createServer")

	bootstrapData, exists := secret.Data["value"]
	if !exists {
		return errors.New("unable to obtain bootstrap data from secret")
	}

	lan, err := s.getLAN(cs, ms)(ctx)
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

	server, requestLocation, err := s.cloud.CreateServer(
		ctx,
		s.datacenterID(nil),
		s.buildServerProperties(ms, copySpec),
		s.buildServerEntities(ms, entityParams),
	)
	if err != nil {
		return fmt.Errorf("failed to create server in data center %s: %w", s.datacenterID(nil), err)
	}

	log.Info("Successfully requested for server creation", "location", requestLocation)
	ms.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodPost, requestLocation))

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
func (s *Service) buildServerProperties(ms *scope.MachineScope, machineSpec *infrav1.IonosCloudMachineSpec) sdk.ServerProperties {
	props := sdk.ServerProperties{
		AvailabilityZone: ptr.To(machineSpec.AvailabilityZone.String()),
		Cores:            &machineSpec.NumCores,
		CpuFamily:        &machineSpec.CPUFamily,
		Name:             ptr.To(s.serverName(ms)),
		Ram:              &machineSpec.MemoryMB,
	}

	return props
}

type serverEntityParams struct {
	boostrapData string
	machineSpec  infrav1.IonosCloudMachineSpec
	lanID        int32
}

// buildServerEntities returns the server entities for the expected cloud server resource.
func (s *Service) buildServerEntities(ms *scope.MachineScope, params serverEntityParams) sdk.ServerEntities {
	machineSpec := params.machineSpec
	bootVolume := sdk.Volume{
		Properties: &sdk.VolumeProperties{
			AvailabilityZone: ptr.To(machineSpec.Disk.AvailabilityZone.String()),
			Name:             ptr.To(s.serverName(ms)),
			Size:             ptr.To(float32(machineSpec.Disk.SizeGB)),
			SshKeys:          ptr.To(machineSpec.Disk.SSHKeys),
			Type:             ptr.To(machineSpec.Disk.DiskType.String()),
			UserData:         ptr.To(params.boostrapData),
		},
	}

	bootVolume.Properties.ImageAlias = ptr.To(strings.Join(machineSpec.Disk.Image.Aliases, ","))
	if machineSpec.Disk.Image.ID != nil {
		bootVolume.Properties.Image = machineSpec.Disk.Image.ID
		bootVolume.Properties.ImageAlias = nil // we don't want to use the aliases if we have an ID provided
	}

	serverVolumes := sdk.AttachedVolumes{
		Items: ptr.To([]sdk.Volume{bootVolume}),
	}

	// As we want to retrieve a public IP from the DHCP, we need to
	// create a NIC with empty IP addresses and patch the NIC afterward.
	serverNICs := sdk.Nics{
		Items: &[]sdk.Nic{
			{
				Properties: &sdk.NicProperties{
					Dhcp: ptr.To(true),
					Lan:  &params.lanID,
					Name: ptr.To(s.serverName(ms)),
				},
			},
		},
	}

	// Attach server to additional LANs
	items := *serverNICs.Items
	for _, nic := range ms.IonosMachine.Spec.AdditionalNetworks {
		items = append(items, sdk.Nic{Properties: &sdk.NicProperties{
			Lan: ptr.To(nic.NetworkID),
		}})
	}

	return sdk.ServerEntities{
		Nics:    &serverNICs,
		Volumes: &serverVolumes,
	}
}

func (s *Service) renderUserData(ms *scope.MachineScope, input string) string {
	// TODO(lubedacht) update user data to include needed information
	// 	VNC and hostname

	const bootCmdFormat = `bootcmd:
  - echo %[1]s > /etc/hostname
  - hostname %[1]s
`
	bootCmdString := fmt.Sprintf(bootCmdFormat, s.serverName(ms))
	input = fmt.Sprintf("%s\n%s", input, bootCmdString)

	return base64.StdEncoding.EncodeToString([]byte(input))
}

// GetBootstrapDataSecret returns the bootstrap data secret, which has been created by the
// Kubeadm provider.
func (s *Service) GetBootstrapDataSecret(ctx context.Context, ms *scope.MachineScope) (*corev1.Secret, error) {
	name := ptr.Deref(ms.Machine.Spec.Bootstrap.DataSecretName, "")
	if name == "" {
		return nil, errors.New("machine has no bootstrap data yet")
	}
	key := client.ObjectKey{
		Name:      name,
		Namespace: ms.IonosMachine.Namespace,
	}

	s.logger.WithName("GetBoostrapDataSecret").
		V(4).
		Info("searching for bootstrap data", "secret", key.String())

	var lookupSecret corev1.Secret
	if err := ms.Client().Get(ctx, key, &lookupSecret); err != nil {
		return nil, err
	}

	return &lookupSecret, nil
}

func (s *Service) serversURL() string {
	return path.Join("datacenters", s.datacenterID(nil), "servers")
}

// serverName returns a formatted name for the expected cloud server resource.
func (s *Service) serverName(ms *scope.MachineScope) string {
	return fmt.Sprintf(
		"k8s-%s-%s",
		ms.IonosMachine.Namespace,
		ms.IonosMachine.Name)
}
