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

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/utils/pointer"
)

// LANName returns the name of the cluster LAN.
func (s *MachineService) LANName() string {
	return fmt.Sprintf(
		"k8s-lan-%s-%s",
		s.scope.ClusterScope.Cluster.Namespace,
		s.scope.ClusterScope.Cluster.Name)
}

// GetLAN ensures a LAN is created and returns it if available.
func (s *MachineService) GetLAN() (*sdk.Lan, error) {
	var err error
	log := s.scope.Logger.WithName("GetLAN")

	// Check for LAN creation requests
	requestExists, err := s.lanRequestExists(http.MethodPost, "")
	if err != nil {
		return nil, fmt.Errorf("could not check if a LAN request exists")
	}
	if requestExists {
		return nil, nil
	}
	// Search for LAN
	lans, err := s.API().ListLANs(s.ctx, s.DataCenterID())
	if err != nil {
		return nil, fmt.Errorf("could not list LANs in data center")
	}
	for _, l := range *lans.Items {
		if name := l.Properties.Name; name != nil && *name == s.LANName() {
			return &l, nil
		}
	}
	// Create LAN
	log.Info("no LAN was found. requesting creation of a new one")
	requestPath, err := s.API().CreateLAN(s.ctx, s.DataCenterID(), sdk.LanPropertiesPost{
		Name:   pointer.String(s.LANName()),
		Public: pointer.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("could not request LAN creation: %w", err)
	}
	log.WithValues("requestPath", requestPath).Info("Successfully requested for LAN creation")
	return nil, nil
}

// DeleteLAN deletes the lan used by the cluster. A bool indicates if the LAN still exists.
func (s *MachineService) DeleteLAN(lanID string) (bool, error) {
	var err error
	log := s.scope.Logger.WithName("DestroyLAN")

	// Check for LAN deletion requests
	requestExists, err := s.lanRequestExists(http.MethodDelete, lanID)
	if err != nil {
		return false, fmt.Errorf("could not check if a LAN request exists: %w", err)
	}
	if requestExists {
		log.Info("the latest deletion request has not finished yet, so let's try again later.")
		return false, nil
	}
	// Search for LAN
	lan, err := s.API().GetLAN(s.ctx, s.DataCenterID(), lanID)
	if err != nil {
		return false, fmt.Errorf("could not check if LAN exists: %w", err)
	}
	if lan != nil && len(*lan.Entities.Nics.Items) > 0 {
		log.Info("the cluster still has more than node. skipping LAN deletion.")
		return false, nil
	}
	if lan == nil {
		log.Info("lan could not be found")
		return true, nil
	}
	// Destroy LAN
	log.Info("requesting deletion of LAN")
	requestPath, err := s.API().DestroyLAN(s.ctx, s.DataCenterID(), lanID)
	if err != nil {
		return false, fmt.Errorf("could not request deletion of LAN: %w", err)
	}
	log.WithValues("requestPath", requestPath).Info("successfully requested lan deletion.")
	return false, nil
}

// lanRequestExists checks if there is a request for the creation or deletion of a LAN in the data center.
// For deletion requests, it is also necessary to provide the LAN ID (value will be ignored for creation).
func (s *MachineService) lanRequestExists(method string, lanID string) (bool, error) {
	if method != http.MethodPost && method != http.MethodDelete {
		return false, fmt.Errorf("invalid method %s (only POST and DELETE are valid)", method)
	}
	if method == http.MethodDelete && lanID == "" {
		return false, fmt.Errorf("when method is DELETE, lanID cannot be empty")
	}

	lanPath, err := url.JoinPath("datacenter", s.scope.IonosCloudMachine.Spec.DatacenterID, "lan")
	if err != nil {
		return false, fmt.Errorf("could not generate datacenter/{dataCenterID}/lan path: %w", err)
	}
	requests, err := s.API().GetRequests(s.ctx, method, lanPath)
	if err != nil {
		return false, fmt.Errorf("could not get requests: %w", err)
	}
	for _, r := range *requests {
		if method == "POST" {
			var lan sdk.Lan
			err = json.Unmarshal([]byte(*r.Properties.Body), &lan)
			if err != nil {
				return false, fmt.Errorf("could not unmarshal request into LAN: %w", err)
			}
			if *lan.Properties.Name != s.LANName() {
				continue
			}
		} else if method == "DELETE" {
			u, err := url.Parse(*r.Properties.Url)
			if err != nil {
				return false, fmt.Errorf("could not format url: %w", err)
			}
			lanIDPath, err := url.JoinPath(lanPath, lanID)
			if err != nil {
				return false, fmt.Errorf("could not generate lanPath for lan resource: %w", err)
			}

			if !strings.HasSuffix(u.Path, lanIDPath) {
				continue
			}
		}
		status := *r.Metadata.RequestStatus.Metadata.Status
		if status == sdk.RequestStatusFailed {
			message := r.Metadata.RequestStatus.Metadata.Message
			s.scope.Logger.WithValues("requestID", r.Id, "requestStatus", status).
				Error(errors.New(*message), "last request for LAN has failed. logging it for debugging purposes")
			// We just log the error but not return it, so we can retry the request.
			return false, nil
		}
		if status == sdk.RequestStatusQueued || status == sdk.RequestStatusRunning {
			return true, nil
		}
	}
	return false, nil
}
