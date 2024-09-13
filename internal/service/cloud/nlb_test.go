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
	"fmt"
	"net/http"
	"net/netip"
	"testing"

	"github.com/google/uuid"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type nlbSuite struct {
	ServiceTestSuite
}

func TestNLBSuite(t *testing.T) {
	suite.Run(t, new(nlbSuite))
}

func (s *nlbSuite) TestNLBLANName() {
	nlbLANName := s.service.nlbLANName(s.infraLoadBalancer, lanTypePublic)
	expected := "lan-in-" + s.infraLoadBalancer.Namespace + "-" + s.infraLoadBalancer.Name
	s.Equal(expected, nlbLANName)
}

func (s *nlbSuite) TestNLBNICName() {
	nlbNICName := s.service.nlbNICName(s.infraLoadBalancer)
	expected := "nic-nlb-" + s.infraLoadBalancer.Namespace + "-" + s.infraLoadBalancer.Name
	s.Equal(expected, nlbNICName)
}

func (s *nlbSuite) TestNLBName() {
	nlbName := s.service.nlbName(s.infraLoadBalancer)
	expected := s.infraLoadBalancer.Namespace + "-" + s.infraLoadBalancer.Name
	s.Equal(expected, nlbName)
}

func (s *nlbSuite) TestReconcileNLBCreateNLB() {
	status := s.loadBalancerScope.LoadBalancer.Status.NLBStatus

	status.PublicLANID = 2
	status.PrivateLANID = 3

	s.mockListNLBsCall().Return(nil, nil).Once()
	s.mockGetNLBCreationRequestCall().Return(nil, nil).Once()
	s.mockCreateNLBCall().Return("test/nlb/request/path", nil).Once()

	requeue, err := s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)
	s.NotNil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (s *nlbSuite) TestReconcileNLBRequestIsPending() {
	s.mockListNLBsCall().Return(nil, nil).Once()
	s.mockGetNLBCreationRequestCall().Return([]sdk.Request{s.examplePostRequest()}, nil)

	requeue, err := s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *nlbSuite) TestReconcileNLBNoControlPlaneMachines() {
	s.setupExistingNLBScenario(s.defaultNLB())

	requeue, err := s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)

	s.NoError(err)
	s.False(requeue)
	s.Nil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (s *nlbSuite) TestReconcileNLBControlPlaneMachinesAvailableNoNICs() {
	s.setupExistingNLBScenario(s.defaultNLB())

	machines := s.createControlPlaneMachines(0, 3)
	for _, m := range machines {
		s.NoError(s.k8sClient.Create(s.ctx, &m))
	}

	s.mockListServersCall(exampleDatacenterID).Return(
		&sdk.Servers{Items: ptr.To(s.machinesToServers(machines))}, nil,
	).Once()

	requeue, err := s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)

	s.NoError(err)
	s.False(requeue)
	s.Nil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (s *nlbSuite) TestReconcileNLBControlPlaneMachinesAvailable() {
	s.setupExistingNLBScenario(s.defaultNLB())

	machines := s.createControlPlaneMachines(0, 3)
	for _, m := range machines {
		s.NoError(s.k8sClient.Create(s.ctx, &m))
	}

	s.mockListServersCall(exampleDatacenterID).Return(
		&sdk.Servers{Items: ptr.To(s.machinesToServers(
			machines,
			func(server *sdk.Server) {
				server.SetEntities(sdk.ServerEntities{
					Nics: &sdk.Nics{
						Items: &[]sdk.Nic{{
							Properties: &sdk.NicProperties{
								Name: ptr.To(s.service.nlbNICName(s.infraLoadBalancer)),
								Ips:  ptr.To([]string{"203.0.113.10"}),
								Lan:  ptr.To(int32(2)),
							},
						}},
					},
				})
			}))}, nil,
	).Once()

	s.mockCreateNLBForwardingRuleCall(exampleDatacenterID, exampleNLBID).Return("path/to/nlb/", nil).Once()
	requeue, err := s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)

	s.NoError(err)
	s.True(requeue)
	s.NotNil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (s *nlbSuite) TestReconcileNLBUpdateForwardingRules() {
	nlb := s.defaultNLB()
	s.setupExistingNLBScenario(nlb)

	machinesFirstCall := s.createControlPlaneMachines(0, 2)
	machinesSecondCall := s.createControlPlaneMachines(2, 2)

	for _, m := range machinesFirstCall {
		s.NoError(s.k8sClient.Create(s.ctx, &m))
	}

	ipAddress := new(string)
	*ipAddress = "203.0.113.10"

	applyLBNIC := func(server *sdk.Server) {
		oldIP, err := netip.ParseAddr(*ipAddress)
		s.NoError(err)
		nextIP := oldIP.Next()

		server.SetEntities(sdk.ServerEntities{
			Nics: &sdk.Nics{
				Items: &[]sdk.Nic{{
					Properties: &sdk.NicProperties{
						Name: ptr.To(s.service.nlbNICName(s.infraLoadBalancer)),
						Ips:  ptr.To([]string{oldIP.String()}),
						Lan:  ptr.To(int32(2)),
					},
				}},
			},
		})

		*ipAddress = nextIP.String()
	}

	initialServers := s.machinesToServers(machinesFirstCall, applyLBNIC)

	s.mockListServersCall(exampleDatacenterID).Return(&sdk.Servers{
		Items: ptr.To(initialServers),
	}, nil).Once()
	s.mockCreateNLBForwardingRuleCall(exampleDatacenterID, exampleNLBID).Return("path/to/nlb/", nil).Once()

	requeue, err := s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)

	// We need to pretend we have a rule now
	rule := sdk.NetworkLoadBalancerForwardingRule{
		Id: ptr.To(exampleForwardingRuleID),
		Properties: &sdk.NetworkLoadBalancerForwardingRuleProperties{
			Name:      ptr.To("control-plane-rule"),
			Algorithm: ptr.To("ROUND_ROBIN"),
			Protocol:  ptr.To("TCP"),
		},
	}

	rules := sdk.NetworkLoadBalancerForwardingRules{
		Items: &[]sdk.NetworkLoadBalancerForwardingRule{rule},
	}

	targets := make([]sdk.NetworkLoadBalancerForwardingRuleTarget, 0, len(initialServers))

	for _, s := range initialServers {
		nicIP := (*(*s.Entities.GetNics().GetItems())[0].GetProperties().GetIps())[0]

		targets = append(targets, sdk.NetworkLoadBalancerForwardingRuleTarget{
			Ip: ptr.To(nicIP),
		})
	}

	rule.GetProperties().SetTargets(targets)
	nlb.Entities.SetForwardingrules(rules)

	for _, m := range machinesSecondCall {
		s.NoError(s.k8sClient.Create(s.ctx, &m))
	}

	s.setupExistingNLBScenario(nlb)

	finalServers := s.machinesToServers(machinesSecondCall, applyLBNIC)
	allServers := append(initialServers, finalServers...)

	s.mockListServersCall(exampleDatacenterID).Return(&sdk.Servers{
		Items: ptr.To(allServers),
	}, nil).Once()

	s.mockUpdateNLBForwardingRuleCall(exampleDatacenterID, exampleNLBID, exampleForwardingRuleID, 4).
		Return("/path/to/update", nil).Once()

	requeue, err = s.service.ReconcileNLB(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)

	s.NotNil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (s *nlbSuite) TestEnsureNLBAvailable() {
	s.loadBalancerScope.LoadBalancer.Status.NLBStatus.ID = exampleNLBID
	s.loadBalancerScope.LoadBalancer.Status.CurrentRequest = &infrav1.ProvisioningRequest{}
	s.mockGetNLBCall(exampleNLBID).Return(s.defaultNLB(), nil).Once()

	nlb, requeue, err := s.service.ensureNLB(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
	s.NotNil(nlb)
}

func (s *nlbSuite) TestReconcileNLBDeletionRequestPending() {
	s.mockGetNLBCreationRequestCall().Return([]sdk.Request{s.examplePostRequest()}, nil).Once()

	requeue, err := s.service.ReconcileNLBDeletion(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *nlbSuite) TestReconcileNLBDeletionDeletionInProgress() {
	s.mockGetNLBCreationRequestCall().Return(nil, nil).Once()
	s.mockGetNLBDeletionRequestCall(exampleNLBID).Return([]sdk.Request{s.exampleDeleteRequest(exampleNLBID)}, nil)
	s.loadBalancerScope.LoadBalancer.SetNLBID(exampleNLBID)

	requeue, err := s.service.ReconcileNLBDeletion(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *nlbSuite) TestReconcileNLBDeletion() {
	s.setupExistingNLBScenario(s.defaultNLB())
	s.mockGetNLBCreationRequestCall().Return(nil, nil).Once()
	s.mockGetNLBDeletionRequestCall(exampleNLBID).Return(nil, nil)

	s.mockDeleteNLBCall(exampleNLBID).Return("path/to/deletion", nil).Once()

	requeue, err := s.service.ReconcileNLBDeletion(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)
	s.NotNil(s.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (s *nlbSuite) TestReconcileLoadBalancerNetworksCreateIncomingAndOutgoing() {
	s.mockListLANsCall(s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(nil, nil).Once()
	s.mockGetLANCreationRequestsCall(s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(nil, nil).Twice()

	s.mockCreateNLBLANCall(sdk.LanProperties{
		Name:          ptr.To(s.service.nlbLANName(s.infraLoadBalancer, lanTypePublic)),
		Ipv6CidrBlock: ptr.To("AUTO"),
		Public:        ptr.To(true),
	}).Return("path/to/lan", nil).Once()

	const (
		incomingLANID = "2"
		outgoingLANID = "3"
	)

	s.mockWaitForRequestCall("path/to/lan").Return(nil).Twice()

	firstLans := []sdk.Lan{{
		Id:       ptr.To(incomingLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(sdk.Available)},
		Properties: &sdk.LanProperties{
			Name:          ptr.To(s.service.nlbLANName(s.infraLoadBalancer, lanTypePublic)),
			Ipv6CidrBlock: ptr.To("AUTO"),
			Public:        ptr.To(true),
		},
	}}

	call := s.mockListLANsCall(s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(&sdk.Lans{
		Items: &firstLans,
	}, nil).Twice()

	s.mockCreateNLBLANCall(sdk.LanProperties{
		Name:          ptr.To(s.service.nlbLANName(s.infraLoadBalancer, lanTypePrivate)),
		Ipv6CidrBlock: ptr.To("AUTO"),
		Public:        ptr.To(false),
	}).Return("path/to/lan", nil).Once().NotBefore(call)

	secondLANs := append(firstLans, sdk.Lan{
		Id:       ptr.To(outgoingLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(sdk.Available)},
		Properties: &sdk.LanProperties{
			Name:          ptr.To(s.service.nlbLANName(s.infraLoadBalancer, lanTypePrivate)),
			Ipv6CidrBlock: ptr.To("AUTO"),
			Public:        ptr.To(false),
		},
	})

	s.mockListLANsCall(s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(&sdk.Lans{
		Items: &secondLANs,
	}, nil).Once()

	requeue, err := s.service.ReconcileNLBNetworks(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.False(requeue)
}

func (s *nlbSuite) TestReconcileLoadBalancerNetworksLANsExist() {
	const (
		incomingLANID = "2"
		outgoingLANID = "3"
	)

	lans := []sdk.Lan{{
		Id:       ptr.To(incomingLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(sdk.Available)},
		Properties: &sdk.LanProperties{
			Name:          ptr.To(s.service.nlbLANName(s.infraLoadBalancer, lanTypePublic)),
			Ipv6CidrBlock: ptr.To("AUTO"),
			Public:        ptr.To(true),
		},
	}, {
		Id:       ptr.To(outgoingLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(sdk.Busy)},
		Properties: &sdk.LanProperties{
			Name:          ptr.To(s.service.nlbLANName(s.infraLoadBalancer, lanTypePrivate)),
			Ipv6CidrBlock: ptr.To("AUTO"),
			Public:        ptr.To(false),
		},
	}}
	s.mockListLANsCall(s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(&sdk.Lans{Items: &lans}, nil).Twice()

	requeue, err := s.service.ReconcileNLBNetworks(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *nlbSuite) TestReconcileControlPlaneLAN() {
	machines := s.createControlPlaneMachines(0, 2)
	servers := s.machinesToServers(machines)
	s.NoError(s.loadBalancerScope.LoadBalancer.SetPrivateLANID("2"))

	for _, m := range machines {
		s.NoError(s.k8sClient.Create(s.ctx, &m))
	}

	for _, srv := range servers {
		s.mockGetNICCreationRequestCall(*srv.GetId()).Return(nil, nil).Once()
		s.mockGetServerCall(s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID, *srv.GetId()).
			Return(&srv, nil).
			Once()

		s.mockCreateNICCall(*srv.GetId()).Return("path/to/nic", nil).Once()
		s.mockWaitForRequestCall("path/to/nic").Return(nil).Once()
	}

	requeue, err := s.service.reconcileControlPlaneLAN(s.ctx, s.loadBalancerScope)
	s.NoError(err)
	s.False(requeue)
}

func (s *nlbSuite) setupExistingNLBScenario(nlb *sdk.NetworkLoadBalancer) {
	s.T().Helper()

	s.loadBalancerScope.LoadBalancer.Status.NLBStatus.ID = exampleNLBID
	s.loadBalancerScope.LoadBalancer.Status.CurrentRequest = &infrav1.ProvisioningRequest{}
	s.mockGetNLBCall(exampleNLBID).Return(nlb, nil).Once()
}

func (s *nlbSuite) mockGetNLBCall(nlbID string) *clienttest.MockClient_GetNLB_Call {
	return s.ionosClient.EXPECT().GetNLB(mock.MatchedBy(nonNilCtx), s.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID)
}

func (s *nlbSuite) mockListNLBsCall() *clienttest.MockClient_ListNLBs_Call {
	return s.ionosClient.EXPECT().ListNLBs(mock.MatchedBy(nonNilCtx), s.infraLoadBalancer.Spec.NLB.DatacenterID)
}

func (s *nlbSuite) mockCreateNLBCall() *clienttest.MockClient_CreateNLB_Call {
	return s.ionosClient.EXPECT().CreateNLB(mock.MatchedBy(nonNilCtx),
		s.infraLoadBalancer.Spec.NLB.DatacenterID,
		mock.Anything,
	)
}

func (s *nlbSuite) mockDeleteNLBCall(nlbID string) *clienttest.MockClient_DeleteNLB_Call {
	return s.ionosClient.EXPECT().DeleteNLB(mock.MatchedBy(nonNilCtx), s.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID)
}

func (s *nlbSuite) mockCreateNLBForwardingRuleCall(datacenterID, nlbID string) *clienttest.MockClient_CreateNLBForwardingRule_Call {
	return s.ionosClient.EXPECT().CreateNLBForwardingRule(mock.MatchedBy(nonNilCtx), datacenterID, nlbID, mock.Anything)
}

func (s *nlbSuite) mockUpdateNLBForwardingRuleCall(datacenterID, nlbID, ruleID string, targetLen int) *clienttest.MockClient_UpdateNLBForwardingRule_Call {
	return s.ionosClient.EXPECT().UpdateNLBForwardingRule(mock.MatchedBy(nonNilCtx), datacenterID, nlbID, ruleID, mock.MatchedBy(func(rule sdk.NetworkLoadBalancerForwardingRule) bool {
		targets := *rule.GetProperties().GetTargets()
		return len(targets) == targetLen
	}))
}

func (s *nlbSuite) mockGetNLBCreationRequestCall() *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(mock.MatchedBy(nonNilCtx), http.MethodPost, s.service.nlbsURL(s.infraLoadBalancer.Spec.NLB.DatacenterID))
}

func (s *nlbSuite) mockGetNLBDeletionRequestCall(nlbID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(mock.MatchedBy(nonNilCtx), http.MethodDelete, s.service.nlbURL(s.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID))
}

func (s *nlbSuite) mockCreateNLBLANCall(properties sdk.LanProperties) *clienttest.MockClient_CreateLAN_Call {
	return s.ionosClient.EXPECT().CreateLAN(mock.MatchedBy(nonNilCtx), s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID, properties)
}

func (s *nlbSuite) mockGetNICCreationRequestCall(serverID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(mock.MatchedBy(nonNilCtx), http.MethodPost, s.service.nicsURL(s.infraLoadBalancer.Spec.NLB.DatacenterID, serverID))
}

func (s *nlbSuite) mockCreateNICCall(serverID string) *clienttest.MockClient_CreateNIC_Call {
	return s.ionosClient.EXPECT().CreateNIC(mock.MatchedBy(nonNilCtx), s.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID, serverID, mock.Anything)
}

func (s *nlbSuite) defaultNLB() *sdk.NetworkLoadBalancer {
	return &sdk.NetworkLoadBalancer{
		Id: ptr.To(exampleNLBID),
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Properties: &sdk.NetworkLoadBalancerProperties{
			Name: ptr.To(s.service.nlbName(s.infraLoadBalancer)),
		},
		Entities: &sdk.NetworkLoadBalancerEntities{},
	}
}

func (s *nlbSuite) examplePostRequest() sdk.Request {
	return s.exampleRequest(requestBuildOptions{
		status:     sdk.RequestStatusQueued,
		method:     http.MethodPost,
		url:        s.service.nlbsURL(s.infraLoadBalancer.Spec.NLB.DatacenterID),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.nlbName(s.infraLoadBalancer)),
		href:       exampleRequestPath,
		targetID:   exampleNLBID,
		targetType: sdk.NETWORKLOADBALANCER,
	})
}

func (s *nlbSuite) exampleDeleteRequest(nlbID string) sdk.Request {
	return s.exampleRequest(requestBuildOptions{
		status:     sdk.RequestStatusQueued,
		method:     http.MethodDelete,
		url:        s.service.nlbURL(s.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, s.service.nlbName(s.infraLoadBalancer)),
		href:       exampleRequestPath,
		targetID:   exampleNLBID,
		targetType: sdk.NETWORKLOADBALANCER,
	})
}

func (s *nlbSuite) createControlPlaneMachines(start, count int) []infrav1.IonosCloudMachine {
	cpMachines := make([]infrav1.IonosCloudMachine, count)
	index := 0
	for nameIndex := start; nameIndex < (start + count); nameIndex++ {
		cpMachines[index] = s.createMachineWithLabels(
			fmt.Sprintf("cp-test-%d", nameIndex),
			uuid.New().String(),
			map[string]string{
				clusterv1.ClusterNameLabel:         s.clusterScope.Cluster.Name,
				clusterv1.MachineControlPlaneLabel: "",
			},
		)
		index++
	}
	return cpMachines
}

func (*nlbSuite) createMachineWithLabels(name, id string, labels map[string]string) infrav1.IonosCloudMachine {
	return infrav1.IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         metav1.NamespaceDefault,
			Labels:            labels,
			CreationTimestamp: metav1.Now(),
		},
		Spec: infrav1.IonosCloudMachineSpec{
			ProviderID:   ptr.To("ionos://" + id),
			DatacenterID: exampleDatacenterID,
		},
	}
}

func (*nlbSuite) machinesToServers(
	machines []infrav1.IonosCloudMachine,
	applyFuncs ...func(*sdk.Server),
) []sdk.Server {
	servers := make([]sdk.Server, len(machines))
	for index, m := range machines {
		servers[index] = sdk.Server{
			Id: ptr.To(m.ExtractServerID()),
			Metadata: &sdk.DatacenterElementMetadata{
				State: ptr.To(sdk.Available),
			},
			Properties: &sdk.ServerProperties{
				Name: ptr.To(m.Name),
			},
		}

		for _, apply := range applyFuncs {
			apply(&servers[index])
		}
	}

	return servers
}
