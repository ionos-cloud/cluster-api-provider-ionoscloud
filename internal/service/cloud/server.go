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

	"github.com/google/uuid"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api/util"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// ReconcileServer ensures the cluster server exist, creating one if it doesn't.
func (s *Service) ReconcileServer() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileServer")

	log.V(4).Info("Reconciling server")

	secret, err := s.scope.GetBootstrapDataSecret(s.ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// secret not available yet
			// just log the error and resume reconciliation.
			log.Info("Bootstrap secret not available yet", "error", err)
			return requeue, nil
		}
		return true, fmt.Errorf("unexpected error when trying to get bootstrap secret: %w", err)
	}

	server, request, err := findResource(
		s.getServer,
		s.getLatestServerCreationRequest,
	)
	if err != nil {
		return false, err
	}
	if request != nil && request.isPending() {
		log.Info("Request is pending", "location", request.location)
		return true, nil
	}

	// server is still provisioning
	if server != nil {
		if state := getState(server); !isAvailable(state) {
			log.Info("LAN is not available yet", "state", state)
			return true, nil
		}
		// server exists and is available.
		return false, nil
	}

	// server does not exist yet, create it
	log.V(4).Info("No server was found. Creating new server")
	if err := s.createServer(secret); err != nil {
		return false, err
	}

	log.V(4).Info("successfully finished reconciling server")
	return false, nil
}

// getServer looks for the server in the data center.
func (s *Service) getServer() (*sdk.Server, error) {
	// first we check if the provider ID is set
	if !ptr.IsNullOrDefault(s.scope.IonosMachine.Spec.ProviderID) {
		serverID := s.scope.IonosMachine.ExtractServerID()
		// we expect the server ID to be a valid UUID
		if _, err := uuid.Parse(serverID); err != nil {
			return nil, fmt.Errorf("invalid server ID %s: %w", serverID, err)
		}

		server, err := s.api().GetServer(s.ctx, s.datacenterID(), serverID)
		// if the server was not found, we will continue, as the request might not
		// have been completed yet
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("failed to get server %s in data center %s: %w", serverID, s.datacenterID(), err)
		}

		// if the server was found, we return it
		if server != nil {
			return server, nil
		}
	}

	serverList, err := s.api().ListServers(s.ctx, s.datacenterID())
	if err != nil {
		return nil, fmt.Errorf("failed to list servers in data center %s: %w", s.datacenterID(), err)
	}

	if items := ptr.Deref(serverList.Items, []sdk.Server{}); len(items) > 0 {
		// find servers with the expected name
		for _, server := range items {
			if server.HasProperties() && *server.Properties.Name == s.serverName() {
				// if the server was found, we set the provider ID and return it
				s.scope.SetProviderID(ptr.Deref(server.Id, ""))
				return &server, nil
			}
		}
	}

	return nil, nil
}

func (s *Service) getLatestServerCreationRequest() (*requestInfo, error) {
	return getMatchingRequest(
		s,
		http.MethodPost,
		path.Join("datacenters", s.datacenterID(), "servers"),
		matchByName[*sdk.Server, *sdk.ServerProperties](s.serverName()),
	)
}

func (s *Service) createServer(secret *corev1.Secret) error {
	log := s.scope.WithName("createServer")

	serverProps := s.buildServerProperties()
	entities := s.buildServerEntities(secret)

	server, requestLocation, err := s.api().CreateServer(s.ctx, s.datacenterID(), serverProps, entities)
	if err != nil {
		return fmt.Errorf("failed to create server in data center %s: %w", s.datacenterID(), err)
	}

	log.Info("Successfully requested for server creation", "location", requestLocation)
	s.scope.IonosMachine.Status.CurrentRequest = ptr.To(infrav1.NewQueuedRequest(http.MethodPost, requestLocation))

	serverID := ptr.Deref(server.GetId(), "")
	if serverID == "" {
		return errors.New("server ID is empty")
	}

	// make sure to set the provider ID
	s.scope.SetProviderID(serverID)

	log.V(4).Info("Done creating server")
	return nil
}

// buildServerProperties returns the server properties for the expected cloud server resource.
func (s *Service) buildServerProperties() sdk.ServerProperties {
	copySpec := s.scope.IonosMachine.Spec.DeepCopy()
	props := sdk.ServerProperties{
		AvailabilityZone: ptr.To(copySpec.AvailabilityZone.String()),
		BootVolume:       nil,
		Cores:            &copySpec.NumCores,
		CpuFamily:        &copySpec.CPUFamily,
		Name:             ptr.To(s.serverName()),
		Ram:              &copySpec.MemoryMB,
	}

	return props
}

func (s *Service) buildServerEntities(_ *corev1.Secret) sdk.ServerEntities {
	return sdk.ServerEntities{}
}

// serverName returns a formatted name for the expected cloud server resource.
func (s *Service) serverName() string {
	role := "worker"
	if util.IsControlPlaneMachine(s.scope.Machine) {
		role = "control-plane"
	}

	return fmt.Sprintf(
		"k8s-%s-%s-%s",
		role,
		s.scope.IonosMachine.Namespace,
		s.scope.IonosMachine.Name)
}
