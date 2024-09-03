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
	"fmt"
	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	"net/http"
)

func (s *Service) nlbLANName(lb *infrav1.IonosCloudLoadBalancer, public bool) string {
	lanType := "outgoing"
	if public {
		lanType = "incomming"
	}

	return fmt.Sprintf("lan-%s-%s-%s",
		lb.Namespace,
		lb.Name,
		lanType,
	)
}

func (s *Service) ReconcileNLB(ctx context.Context, ls *scope.LoadBalancer) (requeue bool, err error) {
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

	log.V(4).Info("Successfully reconciled LoadBalancer Networks")
	return false, nil
}

func (s *Service) reconcileIncomingLAN(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	lan, requeue, err := s.reconcileLoadBalancerLAN(ctx, lb, true)
	if err != nil || requeue {
		return requeue, err
	}

	lb.LoadBalancer.SetPublicLANID(ptr.Deref(lan.GetId(), ""))

	return false, nil
}

func (s *Service) reconcileOutgoingLAN(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	lan, requeue, err := s.reconcileLoadBalancerLAN(ctx, lb, false)
	if err != nil || requeue {
		return requeue, err
	}

	lb.LoadBalancer.SetPrivateLANID(ptr.Deref(lan.GetId(), ""))

	return false, nil
}

func (s *Service) reconcileLoadBalancerLAN(ctx context.Context, lb *scope.LoadBalancer, incoming bool) (lan *sdk.Lan, requeue bool, err error) {
	log := s.logger.WithName("createLoadBalancerLAN")

	log.Info("Creating LAN for NLB", "incoming", incoming)

	var (
		datacenterID = lb.LoadBalancer.Spec.NLB.DatacenterID
		lanName      = s.nlbLANName(lb.LoadBalancer, true)
	)

	lan, requeue, err = s.findLoadBalancerLANByName(ctx, lb, true)
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

	path, err := s.createLoadBalancerLAN(ctx, lanName, datacenterID, true)
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

	requestPath, err = s.ionosClient.CreateLAN(ctx, datacenterID, sdk.LanProperties{
		Name:          ptr.To(lanName),
		Ipv6CidrBlock: ptr.To("AUTO"),
		Public:        ptr.To(public),
	})

	if err != nil {
		return "", err
	}

	return requestPath, nil
}

func (s *Service) findLoadBalancerLANByName(ctx context.Context, lb *scope.LoadBalancer, public bool) (lan *sdk.Lan, requeue bool, err error) {
	log := s.logger.WithName("findLoadBalancerLANByName")

	var (
		datacenterID = lb.LoadBalancer.Spec.NLB.DatacenterID
		lanName      = s.nlbLANName(lb.LoadBalancer, public)
	)

	lan, request, err := scopedFindResource(
		ctx, lb,
		s.getLANByNameFunc(datacenterID, lanName),
		s.getLatestLoadBalancerLANCreationRequest(true))

	if err != nil {
		return nil, true, fmt.Errorf("could not find or create incoming LAN: %w", err)
	}

	if request != nil && request.isPending() {
		log.Info("Request for incoming LAN is pending. Waiting for it to be finished")
		return nil, true, nil
	}

	return lan, false, nil
}

func (s *Service) ReconcileLoadBalancerNetworksDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	log := s.logger.WithName("ReconcileLoadBalancerNetworksDeletion")
	log.V(4).Info("Reconciling LoadBalancer Networks deletion")

	if requeue, err := s.reconcileIncomingLANDeletion(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	if requeue, err := s.reconcileOutgoingLANDeletion(ctx, lb); err != nil || requeue {
		return requeue, err
	}

	log.V(4).Info("Successfully reconciled LoadBalancer Networks deletion")
	return false, nil
}

func (s *Service) reconcileIncomingLANDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	return s.reconcileLoadBalancerLANDeletion(ctx, lb, true)
}

func (s *Service) reconcileOutgoingLANDeletion(ctx context.Context, lb *scope.LoadBalancer) (requeue bool, err error) {
	return s.reconcileLoadBalancerLANDeletion(ctx, lb, false)
}

func (s *Service) reconcileLoadBalancerLANDeletion(ctx context.Context, lb *scope.LoadBalancer, incoming bool) (requeue bool, err error) {
	log := s.logger.WithName("reconcileLoadBalancerLANDeletion")

	log.V(4).Info("Deleting LAN for NLB", "incoming", incoming)

	// check if the LAN exists or if there is a pending creation request
	lan, requeue, err := s.findLoadBalancerLANByName(ctx, lb, incoming)
	if err != nil || requeue {
		return requeue, err
	}

	if lan == nil {

	}

	return false, nil
}

func (s *Service) getLANByNameFunc(datacenterID, lanName string) func(context.Context, *scope.LoadBalancer) (*sdk.Lan, error) {
	return func(ctx context.Context, lb *scope.LoadBalancer) (*sdk.Lan, error) {
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

func (s *Service) getLatestLoadBalancerLANCreationRequest(public bool) func(context.Context, *scope.LoadBalancer) (*requestInfo, error) {
	return func(ctx context.Context, lb *scope.LoadBalancer) (*requestInfo, error) {
		return getMatchingRequest[sdk.Lan](
			ctx, s,
			http.MethodPost,
			s.lansURL(lb.LoadBalancer.Spec.NLB.DatacenterID),
			matchByName[*sdk.Lan, *sdk.LanProperties](s.nlbLANName(lb.LoadBalancer, public)))
	}
}
