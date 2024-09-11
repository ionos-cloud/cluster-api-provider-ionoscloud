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
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"
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

func (*Service) nlbName(lb *infrav1.IonosCloudLoadBalancer) string {
	return lb.Namespace + "-" + lb.Name
}

// ReconcileNLB ensures the creation and update of the NLB.
func (s *Service) ReconcileNLB(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	logger := s.logger.WithName("ReconcileNLB")
	logger.V(4).Info("Reconciling NLB")

	// Check if NLB needs to be created

	nlb, requeue, err := s.ensureNLB(ctx, lb)
	if err != nil || requeue {
		return requeue, err
	}

	return s.ensureForwardingRules(ctx, lb, nlb)
}

func (s *Service) ensureNLB(
	ctx context.Context,
	lb *scope.LoadBalancer,
) (nlb *sdk.NetworkLoadBalancer, requeue bool, err error) {
	nlb, request, err := scopedFindResource(ctx, lb, s.getNLB, s.getLatestNLBCreationRequest)
	if err != nil {
		return nil, true, err
	}

	if request != nil && request.isPending() {
		// Creation is in progress, we need to wait
		return nil, true, nil
	}

	if nlb != nil {
		if ptr.Deref(nlb.GetId(), "") != "" {
			lb.LoadBalancer.SetNLBID(*nlb.GetId())
		}

		return nlb, false, nil
	}

	location, err := s.ionosClient.CreateNLB(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID, sdk.NetworkLoadBalancerProperties{
		Name:        ptr.To(s.nlbName(lb.LoadBalancer)),
		ListenerLan: ptr.To(lb.LoadBalancer.Status.NLBStatus.PublicLANID),
		TargetLan:   ptr.To(lb.LoadBalancer.Status.NLBStatus.PrivateLANID),
	})
	if err != nil {
		return nil, true, err
	}

	lb.LoadBalancer.SetCurrentRequest(http.MethodPost, sdk.RequestStatusQueued, location)
	return nil, true, nil
}

func (s *Service) getControlPlaneMachines(ctx context.Context, lb *scope.LoadBalancer) (sets.Set[sdk.Server], error) {
	// Get CP nodes
	cpMachines, err := lb.ClusterScope.ListMachines(ctx, client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""})
	if err != nil {
		return nil, err
	}

	machineSet := sets.New[string]()
	for _, machine := range cpMachines {
		machineSet.Insert(machine.ExtractServerID())
	}

	servers, err := s.ionosClient.ListServers(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID)
	if err != nil {
		return nil, err
	}

	cpServers := sets.New[sdk.Server]()
	for _, server := range *servers.GetItems() {
		if machineSet.Has(*server.GetId()) {
			cpServers.Insert(server)
		}
	}

	return cpServers, nil
}

func (s *Service) buildExpectedTargets(cpServers []sdk.Server, lb *scope.LoadBalancer) ([]sdk.NetworkLoadBalancerForwardingRuleTarget, error) {
	targets := []sdk.NetworkLoadBalancerForwardingRuleTarget{}

	for _, server := range cpServers {
		nics := ptr.Deref(server.GetEntities().GetNics().GetItems(), []sdk.Nic{})
		nic := findNICByName(nics, s.nlbNICName(lb.LoadBalancer))
		if nic == nil {
			return nil, fmt.Errorf("could not find NIC for server %s", *server.GetId())
		}

		ips := ptr.Deref(nic.GetProperties().GetIps(), []string{})
		if len(ips) == 0 {
			return nil, fmt.Errorf("could not find IP for NIC %s", *nic.GetId())
		}

		targets = append(targets, sdk.NetworkLoadBalancerForwardingRuleTarget{
			Ip:     ptr.To(ips[0]),
			Port:   ptr.To(lb.Endpoint().Port),
			Weight: ptr.To(int32(1)),
		})
	}
	return targets, nil
}

func (s *Service) ensureForwardingRules(ctx context.Context, lb *scope.LoadBalancer, nlb *sdk.NetworkLoadBalancer) (requeue bool, err error) {
	cpServers, err := s.getControlPlaneMachines(ctx, lb)
	if err != nil {
		return true, err
	}

	targets, err := s.buildExpectedTargets(cpServers.UnsortedList(), lb)
	if err != nil {
		return true, err
	}

	ruleName := "control-plane-rule" // TODO make this configurable?

	rules := ptr.Deref(nlb.GetEntities().GetForwardingrules().GetItems(), []sdk.NetworkLoadBalancerForwardingRule{})
	var existingRule sdk.NetworkLoadBalancerForwardingRule
	for _, rule := range rules {
		if *rule.GetProperties().GetName() == ruleName {
			existingRule = rule
			break
		}
	}

	// no rule found, we need to create it
	if !existingRule.HasId() {
		expectedRule := sdk.NetworkLoadBalancerForwardingRule{
			Properties: &sdk.NetworkLoadBalancerForwardingRuleProperties{
				Name:         ptr.To(ruleName),
				Algorithm:    ptr.To("ROUND_ROBIN"),
				ListenerIp:   ptr.To(lb.Endpoint().Host),
				ListenerPort: ptr.To(lb.Endpoint().Port),
				Protocol:     ptr.To("TCP"),
				Targets:      ptr.To(targets),
			},
		}

		location, err := s.ionosClient.CreateNLBForwardingRule(
			ctx,
			lb.LoadBalancer.Spec.NLB.DatacenterID,
			*nlb.Id,
			expectedRule)
		if err != nil {
			return true, err
		}

		lb.LoadBalancer.SetCurrentRequest(http.MethodPost, sdk.RequestStatusQueued, location)
		return true, nil
	}

	// check if rule needs to be updated
	if !s.targetsValid(*existingRule.GetProperties().GetTargets(), targets) {
		existingRule.GetProperties().SetTargets(targets)
		location, err := s.ionosClient.UpdateNLBForwardingRule(
			ctx, lb.LoadBalancer.Spec.NLB.DatacenterID,
			*nlb.GetId(),
			*existingRule.GetId(),
			existingRule,
		)
		if err != nil {
			return true, err
		}

		lb.LoadBalancer.SetCurrentRequest(http.MethodPut, sdk.RequestStatusQueued, location)
		// Update the rule with the new targets
		return true, nil
	}

	return false, nil
}

func (*Service) targetsValid(existing, expected []sdk.NetworkLoadBalancerForwardingRuleTarget) bool {
	if len(existing) != len(expected) {
		return false
	}

	existingSet := sets.New[sdk.NetworkLoadBalancerForwardingRuleTarget](existing...)

	for _, target := range expected {
		if !existingSet.Has(target) {
			return false
		}
	}

	return true
}

func (s *Service) getNLB(ctx context.Context, lb *scope.LoadBalancer) (nlb *sdk.NetworkLoadBalancer, err error) {
	if nlbID := lb.LoadBalancer.GetNLBID(); nlbID != "" {
		nlb, err = s.ionosClient.GetNLB(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID, nlbID)
		if ignoreNotFound(err) != nil {
			return nil, err
		}

		if nlb != nil {
			return nlb, nil
		}
	}

	// We don't have an ID, we need to find the NLB by name
	const listDepth = 3
	nlbs, err := s.apiWithDepth(listDepth).ListNLBs(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID)
	if err != nil {
		return nil, err
	}

	for _, nlb := range ptr.Deref(nlbs.GetItems(), []sdk.NetworkLoadBalancer{}) {
		if nlb.Properties.HasName() && *nlb.Properties.Name == s.nlbName(lb.LoadBalancer) {
			return &nlb, nil
		}
	}

	// return not found error
	return nil, err
}

func (*Service) nlbsURL(datacenterID string) string {
	return fmt.Sprintf("/datacenters/%s/networkloadbalancers", datacenterID)
}

func (*Service) nlbURL(datacenterID, nlbID string) string {
	return fmt.Sprintf("/datacenters/%s/networkloadbalancers/%s", datacenterID, nlbID)
}

// ReconcileNLBDeletion ensures the deletion of the NLB.
func (s *Service) ReconcileNLBDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileNLBDeletion")
	log.V(4).Info("Reconciling NLB deletion")

	// Check if there is an ongoing creation request

	request, err := s.getLatestNLBCreationRequest(ctx, lb)
	if err != nil {
		return true, fmt.Errorf("could not get latest NLB creation request: %w", err)
	}

	if request != nil && request.isPending() {
		log.Info("Found pending NLB creation request. Waiting for it to be finished")
		return true, nil
	}

	// Check if there is an ongoing deletion request
	nlb, request, err := scopedFindResource(ctx, lb, s.getNLB, s.getLatestNLBDeletionRequest)
	if err != nil {
		return true, err
	}

	if request != nil && request.isPending() {
		log.Info("Found pending NLB deletion request. Waiting for it to be finished")
		return true, nil
	}

	if nlb == nil {
		// NLB was already deleted
		lb.LoadBalancer.DeleteCurrentRequest()
		return false, nil
	}

	// Delete the NLB
	path, err := s.ionosClient.DeleteNLB(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID, *nlb.GetId())
	if err != nil {
		return true, err
	}

	lb.LoadBalancer.SetCurrentRequest(http.MethodDelete, sdk.RequestStatusQueued, path)
	return true, nil
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

	if lan != nil {
		if err := lb.LoadBalancer.SetPublicLANID(ptr.Deref(lan.GetId(), "")); err != nil {
			return true, err
		}
	}

	return false, nil
}

func (s *Service) reconcileOutgoingLAN(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	lan, requeue, err := s.reconcileLoadBalancerLAN(ctx, lb, lanTypePrivate)
	if err != nil || requeue {
		return requeue, err
	}

	if lan != nil {
		if err := lb.LoadBalancer.SetPrivateLANID(ptr.Deref(lan.GetId(), "")); err != nil {
			return true, err
		}
	}

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
	return s.reconcileLoadBalancerLANDeletion(ctx, lb, lanTypePublic)
}

func (s *Service) reconcileOutgoingLANDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	return s.reconcileLoadBalancerLANDeletion(ctx, lb, lanTypePrivate)
}

func (s *Service) reconcileLoadBalancerLANDeletion(ctx context.Context, lb *scope.LoadBalancer, lanType string) (requeue bool, err error) {
	log := s.logger.WithName("reconcileLoadBalancerLANDeletion")

	lanID := lb.LoadBalancer.GetPublicLANID()
	if lanType == lanTypePrivate {
		lanID = lb.LoadBalancer.GetPrivateLANID()
	}

	if lanID == "" {
		lan, requeue, err := s.findLoadBalancerLANByName(ctx, lb, lanType)
		if err != nil || requeue {
			return requeue, err
		}

		if lan == nil {
			// LAN was already deleted
			lb.LoadBalancer.DeleteCurrentRequest()
			return false, nil
		}

		return s.deleteAndWaitForLAN(ctx, lb, ptr.Deref(lan.GetId(), ""))
	}

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
	request, err := s.getLatestLoadBalancerLANDeletionRequest(ctx, lb, lanID)
	if err != nil {
		return false, fmt.Errorf("could not check for pending LAN deletion request: %w", err)
	}

	if request != nil && request.isPending() {
		log.Info("Found pending LAN deletion request. Waiting for it to be finished")
		return true, nil
	}

	return s.deleteAndWaitForLAN(ctx, lb, lanID)
}

func (s *Service) deleteAndWaitForLAN(ctx context.Context, lb *scope.LoadBalancer, lanID string) (requeue bool, err error) {
	path, err := s.deleteLoadBalancerLAN(ctx, lb.LoadBalancer.Spec.NLB.DatacenterID, lanID)
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

	errGrp, ctx := errgroup.WithContext(ctx)
	errGrp.SetLimit(len(cpMachines))

	for _, machine := range cpMachines {
		if !machine.Status.Ready {
			continue
		}

		var (
			datacenterID = machine.Spec.DatacenterID
			serverID     = machine.ExtractServerID()
		)

		if pending, err := s.isNICCreationPending(ctx, datacenterID, serverID, expectedNICName); err != nil || pending {
			return pending, err
		}

		server, err := s.getServerByServerID(ctx, datacenterID, serverID)
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

		newNICProps := sdk.NicProperties{
			Name: ptr.To(expectedNICName),
			Dhcp: ptr.To(true),
			Lan:  ptr.To(int32(lanID)),
		}

		errGrp.Go(func() error {
			return s.createAndAttachNIC(ctx, datacenterID, serverID, newNICProps)
		})
	}

	if err := errGrp.Wait(); err != nil {
		return true, err
	}

	return false, nil
}

func (s *Service) isNICCreationPending(ctx context.Context, datacenterID, serverID, expectedName string) (bool, error) {
	// check if there is an ongoing request for the server
	request, err := s.getLatestNICCreateRequest(ctx, datacenterID, serverID, expectedName)
	if err != nil {
		return true, err
	}

	if request != nil && request.isPending() {
		return true, nil
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

func (s *Service) getLatestNLBCreationRequest(ctx context.Context, lb *scope.LoadBalancer) (*requestInfo, error) {
	return s.getLatestNLBRequestByMethod(ctx, http.MethodPost, s.nlbsURL(lb.LoadBalancer.Spec.NLB.DatacenterID),
		matchByName[*sdk.NetworkLoadBalancer, *sdk.NetworkLoadBalancerProperties](s.nlbName(lb.LoadBalancer)))
}

func (s *Service) getLatestNLBDeletionRequest(ctx context.Context, lb *scope.LoadBalancer) (*requestInfo, error) {
	return s.getLatestNLBRequestByMethod(ctx, http.MethodDelete, s.nlbURL(lb.LoadBalancer.Spec.NLB.DatacenterID, lb.LoadBalancer.Status.NLBStatus.ID))
}

func (s *Service) getLatestNLBRequestByMethod(
	ctx context.Context,
	method, path string,
	matcher ...matcherFunc[*sdk.NetworkLoadBalancer],
) (*requestInfo, error) {
	return getMatchingRequest(
		ctx, s,
		method,
		path,
		matcher...)
}
