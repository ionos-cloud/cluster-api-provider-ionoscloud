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

func (n *nlbSuite) TestNLBLANName() {
	nlbLANName := n.service.nlbLANName(n.infraLoadBalancer, lanTypePublic)
	expected := "lan-in-" + n.infraLoadBalancer.Namespace + "-" + n.infraLoadBalancer.Name
	n.Equal(expected, nlbLANName)
}

func (n *nlbSuite) TestNLBNICName() {
	nlbNICName := n.service.nlbNICName(n.infraLoadBalancer)
	expected := "nic-nlb-" + n.infraLoadBalancer.Namespace + "-" + n.infraLoadBalancer.Name
	n.Equal(expected, nlbNICName)
}

func (n *nlbSuite) TestNLBName() {
	nlbName := n.service.nlbName(n.infraLoadBalancer)
	expected := n.infraLoadBalancer.Namespace + "-" + n.infraLoadBalancer.Name
	n.Equal(expected, nlbName)
}

func (n *nlbSuite) TestReconcileNLBCreateNLB() {
	status := n.loadBalancerScope.LoadBalancer.Status.NLBStatus

	status.PublicLANID = 2
	status.PrivateLANID = 3

	n.mockListNLBsCall().Return(nil, nil).Once()
	n.mockGetNLBCreationRequestCall().Return(nil, nil).Once()
	n.mockCreateNLBCall().Return("test/nlb/request/path", nil).Once()

	requeue, err := n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)
	n.NotNil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (n *nlbSuite) TestReconcileNLBRequestIsPending() {
	n.mockListNLBsCall().Return(nil, nil).Once()
	n.mockGetNLBCreationRequestCall().Return([]sdk.Request{n.examplePostRequest()}, nil)

	requeue, err := n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)
}

func (n *nlbSuite) TestReconcileNLBNoControlPlaneMachines() {
	n.setupExistingNLBScenario(n.defaultNLB())

	requeue, err := n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)

	n.NoError(err)
	n.False(requeue)
	n.Nil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (n *nlbSuite) TestReconcileNLBControlPlaneMachinesAvailableNoNICs() {
	n.setupExistingNLBScenario(n.defaultNLB())

	machines := n.createControlPlaneMachines(0, 3)
	for _, m := range machines {
		n.NoError(n.k8sClient.Create(n.ctx, &m))
	}

	n.mockListServersCall(exampleDatacenterID).Return(
		&sdk.Servers{Items: ptr.To(n.machinesToServers(machines))}, nil,
	).Once()

	requeue, err := n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)

	n.NoError(err)
	n.False(requeue)
	n.Nil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (n *nlbSuite) TestReconcileNLBControlPlaneMachinesAvailable() {
	n.setupExistingNLBScenario(n.defaultNLB())

	machines := n.createControlPlaneMachines(0, 3)
	for _, m := range machines {
		n.NoError(n.k8sClient.Create(n.ctx, &m))
	}

	n.mockListServersCall(exampleDatacenterID).Return(
		&sdk.Servers{Items: ptr.To(n.machinesToServers(
			machines,
			func(server *sdk.Server) {
				server.SetEntities(sdk.ServerEntities{
					Nics: &sdk.Nics{
						Items: &[]sdk.Nic{{
							Properties: &sdk.NicProperties{
								Name: ptr.To(n.service.nlbNICName(n.infraLoadBalancer)),
								Ips:  ptr.To([]string{"203.0.113.10"}),
								Lan:  ptr.To(int32(2)),
							},
						}},
					},
				})
			}))}, nil,
	).Once()

	n.mockCreateNLBForwardingRuleCall(exampleDatacenterID, exampleNLBID).Return("path/to/nlb/", nil).Once()
	requeue, err := n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)

	n.NoError(err)
	n.True(requeue)
	n.NotNil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (n *nlbSuite) TestReconcileNLBUpdateForwardingRules() {
	nlb := n.defaultNLB()
	n.setupExistingNLBScenario(nlb)

	machinesFirstCall := n.createControlPlaneMachines(0, 2)
	machinesSecondCall := n.createControlPlaneMachines(2, 2)

	for _, m := range machinesFirstCall {
		n.NoError(n.k8sClient.Create(n.ctx, &m))
	}

	ipAddress := new(string)
	*ipAddress = "203.0.113.10"

	applyLBNIC := func(server *sdk.Server) {
		oldIP, err := netip.ParseAddr(*ipAddress)
		n.NoError(err)
		nextIP := oldIP.Next()

		server.SetEntities(sdk.ServerEntities{
			Nics: &sdk.Nics{
				Items: &[]sdk.Nic{{
					Properties: &sdk.NicProperties{
						Name: ptr.To(n.service.nlbNICName(n.infraLoadBalancer)),
						Ips:  ptr.To([]string{oldIP.String()}),
						Lan:  ptr.To(int32(2)),
					},
				}},
			},
		})

		*ipAddress = nextIP.String()
	}

	initialServers := n.machinesToServers(machinesFirstCall, applyLBNIC)

	n.mockListServersCall(exampleDatacenterID).Return(&sdk.Servers{
		Items: ptr.To(initialServers),
	}, nil).Once()
	n.mockCreateNLBForwardingRuleCall(exampleDatacenterID, exampleNLBID).Return("path/to/nlb/", nil).Once()

	requeue, err := n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)

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
		n.NoError(n.k8sClient.Create(n.ctx, &m))
	}

	n.setupExistingNLBScenario(nlb)

	finalServers := n.machinesToServers(machinesSecondCall, applyLBNIC)
	allServers := append(initialServers, finalServers...)

	n.mockListServersCall(exampleDatacenterID).Return(&sdk.Servers{
		Items: ptr.To(allServers),
	}, nil).Once()

	n.mockUpdateNLBForwardingRuleCall(exampleDatacenterID, exampleNLBID, exampleForwardingRuleID, 4).
		Return("/path/to/update", nil).Once()

	requeue, err = n.service.ReconcileNLB(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)

	n.NotNil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (n *nlbSuite) TestEnsureNLBAvailable() {
	n.loadBalancerScope.LoadBalancer.Status.NLBStatus.ID = exampleNLBID
	n.loadBalancerScope.LoadBalancer.Status.CurrentRequest = &infrav1.ProvisioningRequest{}
	n.mockGetNLBCall(exampleNLBID).Return(n.defaultNLB(), nil).Once()

	nlb, requeue, err := n.service.ensureNLB(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.False(requeue)
	n.Nil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
	n.NotNil(nlb)
}

func (n *nlbSuite) TestReconcileNLBDeletionRequestPending() {
	n.mockGetNLBCreationRequestCall().Return([]sdk.Request{n.examplePostRequest()}, nil).Once()

	requeue, err := n.service.ReconcileNLBDeletion(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)
}

func (n *nlbSuite) TestReconcileNLBDeletionDeletionInProgress() {
	n.mockGetNLBCreationRequestCall().Return(nil, nil).Once()
	n.mockGetNLBDeletionRequestCall(exampleNLBID).Return([]sdk.Request{n.exampleDeleteRequest(exampleNLBID)}, nil)
	n.loadBalancerScope.LoadBalancer.SetNLBID(exampleNLBID)

	requeue, err := n.service.ReconcileNLBDeletion(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)
}

func (n *nlbSuite) TestReconcileNLBDeletion() {
	n.setupExistingNLBScenario(n.defaultNLB())
	n.mockGetNLBCreationRequestCall().Return(nil, nil).Once()
	n.mockGetNLBDeletionRequestCall(exampleNLBID).Return(nil, nil)

	n.mockDeleteNLBCall(exampleNLBID).Return("path/to/deletion", nil).Once()

	requeue, err := n.service.ReconcileNLBDeletion(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.True(requeue)
	n.NotNil(n.loadBalancerScope.LoadBalancer.Status.CurrentRequest)
}

func (n *nlbSuite) TestReconcileLoadBalancerNetworksCreateIncomingAndOutgoing() {
	n.mockListLANsCall(n.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(nil, nil).Once()
	n.mockGetLANCreationRequestsCall(n.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(nil, nil).Twice()

	n.mockCreateNLBLANCall(sdk.LanProperties{
		Name:          ptr.To(n.service.nlbLANName(n.infraLoadBalancer, lanTypePublic)),
		Ipv6CidrBlock: ptr.To("AUTO"),
		Public:        ptr.To(true),
	}).Return("path/to/lan", nil).Once()

	const (
		incomingLANID = "2"
		outgoingLANID = "3"
	)

	n.mockWaitForRequestCall("path/to/lan").Return(nil).Twice()

	firstLans := []sdk.Lan{{
		Id:       ptr.To(incomingLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(sdk.Available)},
		Properties: &sdk.LanProperties{
			Name:          ptr.To(n.service.nlbLANName(n.infraLoadBalancer, lanTypePublic)),
			Ipv6CidrBlock: ptr.To("AUTO"),
			Public:        ptr.To(true),
		},
	}}

	call := n.mockListLANsCall(n.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(&sdk.Lans{
		Items: &firstLans,
	}, nil).Twice()

	n.mockCreateNLBLANCall(sdk.LanProperties{
		Name:          ptr.To(n.service.nlbLANName(n.infraLoadBalancer, lanTypePrivate)),
		Ipv6CidrBlock: ptr.To("AUTO"),
		Public:        ptr.To(false),
	}).Return("path/to/lan", nil).Once().NotBefore(call)

	secondLANs := append(firstLans, sdk.Lan{
		Id:       ptr.To(outgoingLANID),
		Metadata: &sdk.DatacenterElementMetadata{State: ptr.To(sdk.Available)},
		Properties: &sdk.LanProperties{
			Name:          ptr.To(n.service.nlbLANName(n.infraLoadBalancer, lanTypePrivate)),
			Ipv6CidrBlock: ptr.To("AUTO"),
			Public:        ptr.To(false),
		},
	})

	n.mockListLANsCall(n.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID).Return(&sdk.Lans{
		Items: &secondLANs,
	}, nil).Once()

	requeue, err := n.service.ReconcileNLBNetworks(n.ctx, n.loadBalancerScope)
	n.NoError(err)
	n.False(requeue)
}

func (n *nlbSuite) setupExistingNLBScenario(nlb *sdk.NetworkLoadBalancer) {
	n.T().Helper()

	n.loadBalancerScope.LoadBalancer.Status.NLBStatus.ID = exampleNLBID
	n.loadBalancerScope.LoadBalancer.Status.CurrentRequest = &infrav1.ProvisioningRequest{}
	n.mockGetNLBCall(exampleNLBID).Return(nlb, nil).Once()
}

func (n *nlbSuite) mockGetNLBCall(nlbID string) *clienttest.MockClient_GetNLB_Call {
	return n.ionosClient.EXPECT().GetNLB(n.ctx, n.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID)
}

func (n *nlbSuite) mockListNLBsCall() *clienttest.MockClient_ListNLBs_Call {
	return n.ionosClient.EXPECT().ListNLBs(n.ctx, n.infraLoadBalancer.Spec.NLB.DatacenterID)
}

func (n *nlbSuite) mockCreateNLBCall() *clienttest.MockClient_CreateNLB_Call {
	return n.ionosClient.EXPECT().CreateNLB(n.ctx,
		n.infraLoadBalancer.Spec.NLB.DatacenterID,
		mock.Anything,
	)
}

func (n *nlbSuite) mockDeleteNLBCall(nlbID string) *clienttest.MockClient_DeleteNLB_Call {
	return n.ionosClient.EXPECT().DeleteNLB(n.ctx, n.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID)
}

func (n *nlbSuite) mockCreateNLBForwardingRuleCall(datacenterID, nlbID string) *clienttest.MockClient_CreateNLBForwardingRule_Call {
	return n.ionosClient.EXPECT().CreateNLBForwardingRule(n.ctx, datacenterID, nlbID, mock.Anything)
}

func (n *nlbSuite) mockUpdateNLBForwardingRuleCall(datacenterID, nlbID, ruleID string, targetLen int) *clienttest.MockClient_UpdateNLBForwardingRule_Call {
	return n.ionosClient.EXPECT().UpdateNLBForwardingRule(n.ctx, datacenterID, nlbID, ruleID, mock.MatchedBy(func(rule sdk.NetworkLoadBalancerForwardingRule) bool {
		targets := *rule.GetProperties().GetTargets()
		return len(targets) == targetLen
	}))
}

func (n *nlbSuite) mockGetNLBCreationRequestCall() *clienttest.MockClient_GetRequests_Call {
	return n.ionosClient.EXPECT().GetRequests(n.ctx, http.MethodPost, n.service.nlbsURL(n.infraLoadBalancer.Spec.NLB.DatacenterID))
}

func (n *nlbSuite) mockGetNLBDeletionRequestCall(nlbID string) *clienttest.MockClient_GetRequests_Call {
	return n.ionosClient.EXPECT().GetRequests(n.ctx, http.MethodDelete, n.service.nlbURL(n.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID))
}

func (n *nlbSuite) mockCreateNLBLANCall(properties sdk.LanProperties) *clienttest.MockClient_CreateLAN_Call {
	return n.ionosClient.EXPECT().CreateLAN(n.ctx, n.loadBalancerScope.LoadBalancer.Spec.NLB.DatacenterID, properties)
}

func (n *nlbSuite) defaultNLB() *sdk.NetworkLoadBalancer {
	return &sdk.NetworkLoadBalancer{
		Id: ptr.To(exampleNLBID),
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Properties: &sdk.NetworkLoadBalancerProperties{
			Name: ptr.To(n.service.nlbName(n.infraLoadBalancer)),
		},
		Entities: &sdk.NetworkLoadBalancerEntities{},
	}
}

func (n *nlbSuite) examplePostRequest() sdk.Request {
	return n.exampleRequest(requestBuildOptions{
		status:     sdk.RequestStatusQueued,
		method:     http.MethodPost,
		url:        n.service.nlbsURL(n.infraLoadBalancer.Spec.NLB.DatacenterID),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, n.service.nlbName(n.infraLoadBalancer)),
		href:       exampleRequestPath,
		targetID:   exampleNLBID,
		targetType: sdk.NETWORKLOADBALANCER,
	})
}

func (n *nlbSuite) exampleDeleteRequest(nlbID string) sdk.Request {
	return n.exampleRequest(requestBuildOptions{
		status:     sdk.RequestStatusQueued,
		method:     http.MethodDelete,
		url:        n.service.nlbURL(n.infraLoadBalancer.Spec.NLB.DatacenterID, nlbID),
		body:       fmt.Sprintf(`{"properties": {"name": "%s"}}`, n.service.nlbName(n.infraLoadBalancer)),
		href:       exampleRequestPath,
		targetID:   exampleNLBID,
		targetType: sdk.NETWORKLOADBALANCER,
	})
}

func (n *nlbSuite) createControlPlaneMachines(start, count int) []infrav1.IonosCloudMachine {
	cpMachines := make([]infrav1.IonosCloudMachine, count)
	index := 0
	for nameIndex := start; nameIndex < (start + count); nameIndex++ {
		cpMachines[index] = n.createMachineWithLabels(
			fmt.Sprintf("cp-test-%d", nameIndex),
			uuid.New().String(),
			map[string]string{
				clusterv1.ClusterNameLabel:         n.clusterScope.Cluster.Name,
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
