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
	"strconv"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	lanTypePublic  string = "in"
	lanTypePrivate string = "out"
)

func (*Service) nlbLANName(lb *infrav1.IonosCloudLoadBalancer, lanType string) string {
	return fmt.Sprintf("lan-%s-%s-%s",
		lanType,
		lb.Namespace,
		lb.Name,
	)
}

func (*Service) nlbNICName(lb *infrav1.IonosCloudLoadBalancer) string {
	return fmt.Sprintf("nic-nlb-%s-%s",
		lb.Namespace,
		lb.Name,
	)
}

// ReconcileNLB ensures the creation and update of the NLB.
func (*Service) ReconcileNLB(_ context.Context, _ *scope.LoadBalancer) (requeue bool, err error) {
	panic("implement me")
}

// ReconcileLoadBalancerNetworks reconciles the networks for the corresponding NLB.
//
// The following networks need to be created for a basic NLB configuration:
// * Incoming public LAN. This will be used to expose the NLB to the internet.
// * Outgoing private LAN. This LAN will be connected with the NICs of control plane nodes.
func (s *Service) ReconcileLoadBalancerNetworks(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLoadBalancerNetworks")
	log.V(4).Info("Reconciling LoadBalancer Networks")

	if requeue, err := s.reconcileIncomingLAN(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	if requeue, err := s.reconcileOutgoingLAN(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	if requeue, err := s.reconcileControlPlaneLAN(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	log.V(4).Info("Successfully reconciled LoadBalancer Networks")
	return false, nil
}

// ReconcileLoadBalancerNetworksDeletion handles the deletion of the networks for the corresponding NLB.
func (s *Service) ReconcileLoadBalancerNetworksDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLoadBalancerNetworksDeletion")
	log.V(4).Info("Reconciling LoadBalancer Networks deletion")

	cpMachines, err := lb.ClusterScope.ListMachines(ctx, client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""})
	if err != nil {
		return true, err
	}

	if len(cpMachines) > 0 {
		log.Info("Control plane machines still exist. Waiting for machines to be deleted")
		return true, nil
	}

	if requeue, err := s.reconcileOutgoingLANDeletion(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	if requeue, err := s.reconcileIncomingLANDeletion(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	log.V(4).Info("Successfully reconciled LoadBalancer Networks deletion")
	return false, nil
}

func (s *Service) reconcileIncomingLAN(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	lan, requeue, err := s.reconcileLoadBalancerLAN(ctx, lb, lanTypePublic)
	if err != nil || requeue {
		return requeue, err
	}

	lb.LoadBalancer.SetPublicLANID(ptr.Deref(lan.GetId(), ""))

	return false, nil
}

func (s *Service) reconcileOutgoingLAN(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	lan, requeue, err := s.reconcileLoadBalancerLAN(ctx, lb, lanTypePrivate)
	if err != nil || requeue {
		return requeue, err
	}

	lb.LoadBalancer.SetPrivateLANID(ptr.Deref(lan.GetId(), ""))

	return false, nil
}

func (s *Service) reconcileLoadBalancerLAN(
	ctx context.Context,
	lb *scope.LoadBalancer,
	lanType string,
) (lan *sdk.Lan, requeue bool, err error) {
	log := s.logger.WithName("createLoadBalancerLAN")

	log.Info("Creating LAN for NLB", "type", lanType)

	var (
		datacenterID = lb.LoadBalancer.Spec.NLB.DatacenterID
		lanName      = s.nlbLANName(lb.LoadBalancer, lanType)
	)

	lan, requeue, err = s.findLoadBalancerLANByName(ctx, lb, lanType)
	if err != nil || requeue {
		return nil, requeue, err
	}

	if lan != nil {
		if state := getState(lan); !isAvailable(state) {
			log.Info("LAN is not yet available. Waiting for it to be available", "state", state)
			return nil, true, nil
		}

		return lan, false, nil
	}

	path, err := s.createLoadBalancerLAN(ctx, lanName, datacenterID, lanType == lanTypePublic)
	if err != nil {
		return nil, false, fmt.Errorf("could not create incoming LAN: %w", err)
	}

	// LAN creation is usually fast, so we can wait for it to be finished.
	if err := s.ionosClient.WaitForRequest(ctx, path); err != nil {
		lb.LoadBalancer.SetCurrentRequest(http.MethodPost, sdk.RequestStatusQueued, path)
		return nil, true, err
	}

	return lan, false, nil
}

func (s *Service) createLoadBalancerLAN(ctx context.Context, lanName, datacenterID string, public bool) (requestPath string, err error) {
	log := s.logger.WithName("createLoadBalancerLAN")
	log.Info("Creating LoadBalancer LAN", "name", lanName, "datacenterID", datacenterID, "public", public)

	return s.ionosClient.CreateLAN(ctx, datacenterID, sdk.LanProperties{
		Name:          ptr.To(lanName),
		Ipv6CidrBlock: ptr.To("AUTO"),
		Public:        ptr.To(public),
	})
}

func (s *Service) findLoadBalancerLANByName(ctx context.Context, lb *scope.LoadBalancer, lanType string) (lan *sdk.Lan, requeue bool, err error) {
	log := s.logger.WithName("findLoadBalancerLANByName")

	var (
		datacenterID = lb.LoadBalancer.Spec.NLB.DatacenterID
		lanName      = s.nlbLANName(lb.LoadBalancer, lanType)
	)

	lan, request, err := scopedFindResource(
		ctx, lb,
		s.getLANByNameFunc(datacenterID, lanName),
		s.getLatestLoadBalancerLANCreationRequest(lanType))
	if err != nil {
		return nil, true, fmt.Errorf("could not find or create incoming LAN: %w", err)
	}

	if request != nil && request.isPending() {
		log.Info("Request for incoming LAN is pending. Waiting for it to be finished")
		return nil, true, nil
	}

	return lan, false, nil
}

func (s *Service) reconcileIncomingLANDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	return s.reconcileLoadBalancerLANDeletion(ctx, lb, lb.LoadBalancer.Status.NLBStatus.PublicLANID)
}

func (s *Service) reconcileOutgoingLANDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	return s.reconcileLoadBalancerLANDeletion(ctx, lb, lanTypePrivate)
}

func (s *Service) reconcileLoadBalancerLANDeletion(ctx context.Context, lb *scope.LoadBalancer, lanID string) (requeue bool, err error) {
	log := s.logger.WithName("reconcileLoadBalancerLANDeletion")

	log.V(4).Info("Deleting LAN for NLB", "ID", lanID)

	// check if the LAN exists or if there is a pending creation request
	lan, requeue, err := s.findLoadBalancerLANByID(ctx, lb, lanID)
	if err != nil || requeue {
		return requeue, err
	}

	if lan == nil {
		// LAN is already deleted
		lb.LoadBalancer.DeleteCurrentRequest()
		return false, nil
	}

	// check if there is a pending deletion request for the LAN
	request, err := s.getLatestLoadBalancerLANDeletionRequest(ctx, lb, ptr.Deref(lan.GetId(), ""))
	if err != nil {
		return false, fmt.Errorf("could not check for pending LAN deletion request: %w", err)
	}

	if request != nil && request.isPending() {
		log.Info("Found pending LAN deletion request. Waiting for it to be finished")
		return true, nil
	}

	path, err := s.deleteLoadBalancerLAN(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID, ptr.Deref(lan.GetId(), ""))
	if err != nil {
		return true, err
	}

	if err := s.ionosClient.WaitForRequest(ctx, path); err != nil {
		lb.LoadBalancer.SetCurrentRequest(http.MethodDelete, sdk.RequestStatusQueued, path)
		return true, err
	}

	return false, nil
}

func (s *Service) findLoadBalancerLANByID(ctx context.Context, lb *scope.LoadBalancer, lanID string) (lan *sdk.Lan, requeue bool, err error) {
	log := s.logger.WithName("findLoadBalancerLANByID")

	datacenterID := lb.LoadBalancer.Spec.NLB.DatacenterID

	lan, request, err := scopedFindResource(
		ctx, lb,
		s.getLANByIDFunc(datacenterID, lanID),
		s.getLatestLoadBalancerLANCreationRequest(lanID))
	if err != nil {
		return nil, true, fmt.Errorf("could not find or create incoming LAN: %w", err)
	}

	if request != nil && request.isPending() {
		log.Info("Request for incoming LAN is pending. Waiting for it to be finished")
		return nil, true, nil
	}

	return lan, false, nil
}

func (s *Service) deleteLoadBalancerLAN(ctx context.Context, datacenterID, lanID string) (requestPath string, err error) {
	log := s.logger.WithName("createLoadBalancerLAN")
	log.Info("Deleting LoadBalancer LAN", "datacenterID", datacenterID, "lanID", lanID)

	return s.ionosClient.DeleteLAN(ctx, datacenterID, lanID)
}

func (s *Service) getLANByNameFunc(datacenterID, lanName string) func(context.Context, *scope.LoadBalancer) (*sdk.Lan, error) {
	return func(ctx context.Context, _ *scope.LoadBalancer) (*sdk.Lan, error) {
		// check if the LAN exists
		depth := int32(2) // for listing the LANs with their number of NICs
		lans, err := s.apiWithDepth(depth).ListLANs(ctx, datacenterID)
		if err != nil {
			return nil, fmt.Errorf("could not list LANs in data center %s: %w", datacenterID, err)
		}

		var (
			lanCount = 0
			foundLAN *sdk.Lan
		)

		for _, l := range *lans.Items {
			if l.Properties.HasName() && *l.Properties.Name == lanName {
				foundLAN = &l
				lanCount++
			}

			// If there are multiple LANs with the same name, we should return an error.
			// Our logic won't be able to proceed as we cannot select the correct LAN.
			if lanCount > 1 {
				return nil, fmt.Errorf("found multiple LANs with the name: %s", lanName)
			}
		}

		return foundLAN, nil
	}
}

func findNICByName(nics []sdk.Nic, expectedNICName string) *sdk.Nic {
	for _, nic := range nics {
		if ptr.Deref(nic.GetProperties().GetName(), "") == expectedNICName {
			return &nic
		}
	}

	return nil
}

func (s *Service) reconcileControlPlaneLAN(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	if lb.LoadBalancer.GetPrivateLANID() == "" {
		return true, errors.New("private LAN ID is not set")
	}

	cpMachines, err := lb.ClusterScope.ListMachines(ctx, client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""})
	if err != nil {
		return true, err
	}

	expectedNICName := s.nlbNICName(lb.LoadBalancer)

	for _, machine := range cpMachines {
		if !machine.Status.Ready {
			continue
		}

		server, err := s.getServerByServerID(ctx, machine.Spec.DatacenterID, machine.ExtractServerID())
		if err != nil {
			return true, err
		}

		nics := ptr.Deref(server.GetEntities().GetNics().GetItems(), []sdk.Nic{})
		nlbNIC := findNICByName(nics, expectedNICName)
		if nlbNIC != nil {
			// NIC already exists
			continue
		}

		lanID, err := strconv.ParseInt(lb.LoadBalancer.GetPrivateLANID(), 10, 32)
		if err != nil {
			return true, err
		}

		nics = append(nics, sdk.Nic{
			Properties: &sdk.NicProperties{
				Name: ptr.To(expectedNICName),
				Dhcp: ptr.To(true),
				Lan:  ptr.To(int32(lanID)),
			},
		})

		server.Entities.Nics.SetItems(nics)
	}

	return false, nil
}

func (s *Service) getLANByIDFunc(datacenterID, lanID string) func(context.Context, *scope.LoadBalancer) (*sdk.Lan, error) {
	return func(ctx context.Context, _ *scope.LoadBalancer) (*sdk.Lan, error) {
		return s.ionosClient.GetLAN(ctx, datacenterID, lanID)
	}
}

func (s *Service) getLatestLoadBalancerLANCreationRequest(lanType string) func(context.Context, *scope.LoadBalancer) (*requestInfo, error) {
	return func(ctx context.Context, lb *scope.LoadBalancer) (*requestInfo, error) {
		return s.getLatestLANRequestByMethod(ctx, http.MethodPost, s.lansURL(lb.LoadBalancer.Spec.NLB.DatacenterID),
			matchByName[*sdk.Lan, *sdk.LanProperties](s.nlbLANName(lb.LoadBalancer, lanType)))
	}
}

func (s *Service) getLatestLoadBalancerLANDeletionRequest(ctx context.Context, lb *scope.LoadBalancer, lanID string) (*requestInfo, error) {
	return s.getLatestLANRequestByMethod(ctx, http.MethodDelete, s.lanURL(lb.LoadBalancer.Spec.NLB.DatacenterID, lanID))
}
