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

package scope

import (
	"context"
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/cluster-api/util/conditions"

	"github.com/go-logr/logr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

// MachineScope defines a basic context for primary use in IonosCloudMachineReconciler.
type MachineScope struct {
	*logr.Logger

	client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine

	ClusterScope      *ClusterScope
	IonosCloudMachine *infrav1.IonosCloudMachine
}

// MachineScopeParams is a struct that contains the params used to create a new MachineScope through NewMachineScope.
type MachineScopeParams struct {
	Client       client.Client
	Logger       *logr.Logger
	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	InfraCluster *ClusterScope
	IonosMachine *infrav1.IonosCloudMachine
}

// NewMachineScope creates a new MachineScope using the provided params.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("machine scope params lack a client")
	}
	if params.Cluster == nil {
		return nil, errors.New("machine scope params lack a cluster")
	}
	if params.Machine == nil {
		return nil, errors.New("machine scope params lack a Cluster API machine")
	}
	if params.IonosMachine == nil {
		return nil, errors.New("machine scope params lack a ionos cloud machine")
	}
	if params.InfraCluster == nil {
		return nil, errors.New("machine scope params need a ionos cloud cluster scope")
	}
	if params.Logger == nil {
		logger := ctrl.Log
		params.Logger = &logger
	}
	helper, err := patch.NewHelper(params.IonosMachine, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}
	return &MachineScope{
		Logger:            params.Logger,
		client:            params.Client,
		patchHelper:       helper,
		Cluster:           params.Cluster,
		Machine:           params.Machine,
		ClusterScope:      params.InfraCluster,
		IonosCloudMachine: params.IonosMachine,
	}, nil
}

// HasFailed Checks if the IonosCloudMachine is in a failed state.
func (m *MachineScope) HasFailed() bool {
	status := m.IonosCloudMachine.Status
	return status.FailureReason != nil || status.FailureMessage != nil
}

// PatchObject will apply all changes from the IonosCloudMachine.
// It will also make sure to patch the status subresource.
func (m *MachineScope) PatchObject() error {
	conditions.SetSummary(m.IonosCloudMachine,
		conditions.WithConditions(
			infrav1.MachineProvisionedCondition))

	return m.patchHelper.Patch(
		context.TODO(),
		m.IonosCloudMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.MachineProvisionedCondition,
		}})
}

// Finalize will make sure to apply a patch to the current IonosCloudMachine.
// It also implements a retry mechanism to increase the chance of success
// in case the patch operation was not successful.
func (m *MachineScope) Finalize() error {
	// NOTE(lubedacht) retry is only a way to reduce the failure chance,
	// but in general, the reconciliation logic must be resilient
	// to handle an outdated resource from that API server.
	shouldRetry := func(error) bool { return true }
	return retry.OnError(
		retry.DefaultBackoff,
		shouldRetry,
		m.PatchObject)
}
