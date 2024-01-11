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
	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
	"net/http"
	"path"
	"strings"

	sdk "github.com/ionos-cloud/sdk-go/v6"
)

// LANName returns the name of the cluster LAN.
func (s *Service) LANName() string {
	return fmt.Sprintf(
		"k8s-lan-%s-%s",
		s.scope.ClusterScope.Cluster.Namespace,
		s.scope.ClusterScope.Cluster.Name)
}

func (s *Service) ReconcileLAN() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileLAN")

	// try to retrieve the cluster lan
	clusterLan, err := s.GetLAN()
	if clusterLan != nil || err != nil {
		// If we found the LAN, we don't need to create one.
		// TODO(lubedacht) check if patching is required => future task.
		return false, err

	}

	// if we didn't find a lan, we check if a lan is already in creation
	requestStatus, err := s.checkForPendingLANRequest(http.MethodPost, "")
	if err != nil {
		return false, fmt.Errorf("unable to list pending lan requests: %w", err)
	}

	// We want to requeue and check again after some time
	if requestStatus == sdk.RequestStatusRunning || requestStatus == sdk.RequestStatusQueued {
		return true, nil
	}

	// check again as the request might be done right after we checked
	// to prevent duplicate creation
	if requestStatus == sdk.RequestStatusDone {
		clusterLan, err = s.GetLAN()
		if clusterLan != nil || err != nil {
			return false, err
		}

		// If we still don't get a lan here even though we found request, which was done
		// the lan was probably deleted before.
		// Therefore, we will attempt to create the lan again.
		//
		// TODO(lubedacht)
		//  Another solution would be to query for a deletion request and check if the created time
		//  is bigger than the created time of the lan POST request.
	}

	log.V(4).Info("No lan was found. Creating new lan")
	if err := s.CreateLAN(); err != nil {
		return false, err
	}

	// after creating the lan, we want to requeue and let the request be finished
	return true, nil
}

// GetLAN tries to retrieve the cluster related lan in the datacenter.
func (s *Service) GetLAN() (*sdk.Lan, error) {
	// check if the Lan exists
	lans, err := s.API().ListLANs(s.ctx, s.DataCenterID())
	if err != nil {
		return nil, fmt.Errorf("could not list lans in datacenter %s: %w", s.DataCenterID(), err)
	}

	var foundLan *sdk.Lan
	for _, l := range *lans.Items {
		if name := l.Properties.Name; name != nil && *l.Properties.Name == s.LANName() {
			foundLan = &l
			break
		}
	}

	return foundLan, nil
}

func (s *Service) ReconcileLANDeletion() (requeue bool, err error) {
	log := s.scope.Logger.WithName("ReconcileLANDeletion")

	// try to retrieve the cluster LAN
	clusterLAN, err := s.GetLAN()
	if clusterLAN == nil {
		err = s.removeLANPendingRequestFromCluster()
		return err != nil, err
	}
	if err != nil {
		return false, err
	}

	// if we found a LAN, we check if there is a deletion already in process
	requestStatus, err := s.checkForPendingLANRequest(http.MethodDelete, *clusterLAN.Id)
	if err != nil {
		return false, fmt.Errorf("unable to list pending LAN requests: %w", err)
	}
	if requestStatus != "" {
		// We want to requeue and check again after some time
		if requestStatus == sdk.RequestStatusRunning || requestStatus == sdk.RequestStatusQueued {
			return true, nil
		}

		if requestStatus == sdk.RequestStatusDone {
			// Here we can check if the LAN is indeed gone or there's some inconsistency in the last request or
			// this request points to an old, far gone LAN with the same ID.
			clusterLAN, err = s.GetLAN()
			if clusterLAN == nil {
				err = s.removeLANPendingRequestFromCluster()
				return err != nil, err
			}
			if err != nil {
				return false, err
			}
		}
	}

	if clusterLAN != nil && len(*clusterLAN.Entities.Nics.Items) > 0 {
		log.Info("the cluster LAN is still being used by another resource. skipping deletion")
		return false, nil
	}
	// Request for LAN destruction
	err = s.DeleteLAN(*clusterLAN.Id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) CreateLAN() error {
	log := s.scope.Logger.WithName("CreateLAN")

	requestPath, err := s.API().CreateLAN(s.ctx, s.DataCenterID(), sdk.LanPropertiesPost{
		Name:   ptr.To(s.LANName()),
		Public: ptr.To(true),
	})

	if err != nil {
		return fmt.Errorf("unable to create lan in data center %s: %w", s.DataCenterID(), err)
	}

	s.scope.ClusterScope.IonosCluster.Status.PendingRequests[s.DataCenterID()] = &infrav1.ProvisioningRequest{
		Method:      http.MethodPost,
		RequestPath: requestPath,
		State:       infrav1.RequestStatusQueued,
	}

	err = s.scope.ClusterScope.PatchObject()
	if err != nil {
		return fmt.Errorf("unable to patch the cluster: %w", err)
	}

	log.WithValues("requestPath", requestPath).Info("Successfully requested for LAN creation")

	return nil
}

func (s *Service) DeleteLAN(lanID string) error {
	log := s.scope.Logger.WithName("DeleteLAN")

	requestPath, err := s.API().DestroyLAN(s.ctx, s.DataCenterID(), lanID)
	if err != nil {
		return fmt.Errorf("unable to request lan deletion in data center: %w", err)
	}

	s.scope.ClusterScope.IonosCluster.Status.PendingRequests[s.DataCenterID()] = &infrav1.ProvisioningRequest{
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

// checkForPendingLANRequest checks if there is a request for the creation, update or deletion of a LAN in the data center.
// For update and deletion requests, it is also necessary to provide the LAN ID (value will be ignored for creation).
func (s *Service) checkForPendingLANRequest(method string, lanID string) (status string, err error) {
	switch method {
	default:
		return "", fmt.Errorf("unsupported method %s, allowed methods are %s", method, strings.Join(
			[]string{http.MethodPost, http.MethodDelete, http.MethodPatch},
			",",
		))
	case http.MethodDelete, http.MethodPatch:
		if lanID == "" {
			return "", errors.New("lanID cannot be empty for DELETE and PATCH requests")
		}
		break
	case http.MethodPost:
		break
	}

	lanPath := path.Join("datacenters", s.DataCenterID(), "lan")
	requests, err := s.getPendingRequests(method, lanPath)
	if err != nil {
		return "", err
	}

	for _, r := range requests {
		if method != http.MethodPost {
			id := *(*r.Metadata.RequestStatus.Metadata.Targets)[0].Target.Id
			if id != lanID {
				continue
			}
		} else {
			var lan sdk.Lan
			err = json.Unmarshal([]byte(*r.Properties.Body), &lan)
			if err != nil {
				return "", fmt.Errorf("could not unmarshal request into LAN: %w", err)
			}
			if *lan.Properties.Name != s.LANName() {
				continue
			}
		}

		status := *r.Metadata.RequestStatus.Metadata.Status

		if status == sdk.RequestStatusFailed {
			// We just log the error but not return it, so we can retry the request.
			message := r.Metadata.RequestStatus.Metadata.Message
			s.scope.Logger.WithValues("requestID", r.Id, "requestStatus", status).
				Error(errors.New(*message), "last request for LAN has failed. logging it for debugging purposes")
		}

		return status, nil
	}
	return "", nil
}

func (s *Service) removeLANPendingRequestFromCluster() error {
	s.scope.ClusterScope.IonosCluster.Status.PendingRequests[s.DataCenterID()] = nil
	if err := s.scope.ClusterScope.PatchObject(); err != nil {
		return fmt.Errorf("could not remove stale LAN pending request from cluster: %w", err)
	}
	return nil
}