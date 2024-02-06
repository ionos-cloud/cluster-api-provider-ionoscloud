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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// MachineScope defines a basic context for primary use in IonosCloudMachineReconciler.
type MachineScope struct {
	*logr.Logger

	client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine

	ClusterScope *ClusterScope
	IonosMachine *infrav1.IonosCloudMachine
}

// MachineScopeParams is a struct that contains the params used to create a new MachineScope through NewMachineScope.
type MachineScopeParams struct {
	Client       client.Client
	Logger       *logr.Logger
	Cluster      *clusterv1.Cluster
	Machine      *clusterv1.Machine
	ClusterScope *ClusterScope
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
		return nil, errors.New("machine scope params lack a IONOS Cloud machine")
	}
	if params.ClusterScope == nil {
		return nil, errors.New("machine scope params need a IONOS Cloud cluster scope")
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
		Logger:       params.Logger,
		client:       params.Client,
		patchHelper:  helper,
		Cluster:      params.Cluster,
		Machine:      params.Machine,
		ClusterScope: params.ClusterScope,
		IonosMachine: params.IonosMachine,
	}, nil
}

// GetBootstrapDataSecret returns the bootstrap data secret, which has been created by the
// Kubeadm provider.
func (m *MachineScope) GetBootstrapDataSecret(ctx context.Context) (*corev1.Secret, error) {
	name := ptr.Deref(m.Machine.Spec.Bootstrap.DataSecretName, "")
	if name == "" {
		return nil, errors.New("machine has no bootstrap data yet")
	}
	key := client.ObjectKey{
		Name:      name,
		Namespace: m.IonosMachine.Namespace,
	}

	m.WithName("GetBoostrapDataSecret").
		V(4).
		Info("searching for bootstrap data", "secret", key.String())

	var lookupSecret corev1.Secret
	if err := m.client.Get(ctx, key, &lookupSecret); err != nil {
		return nil, err
	}

	return &lookupSecret, nil
}

// DatacenterID returns the data center ID used by the IonosCloudMachine.
func (m *MachineScope) DatacenterID() string {
	return m.IonosMachine.Spec.DatacenterID
}

func (m *MachineScope) SetProviderID(id string) {
	m.IonosMachine.Spec.ProviderID = ptr.To(fmt.Sprintf("ionos://%s", id))
}

// HasFailed checks if the IonosCloudMachine is in a failed state.
func (m *MachineScope) HasFailed() bool {
	status := m.IonosMachine.Status
	return status.FailureReason != nil || status.FailureMessage != nil
}

// PatchObject will apply all changes from the IonosMachine.
// It will also make sure to patch the status subresource.
func (m *MachineScope) PatchObject() error {
	conditions.SetSummary(m.IonosMachine,
		conditions.WithConditions(
			infrav1.MachineProvisionedCondition))

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// We don't accept and forward a context here. This is on purpose: Even if a reconciliation is
	// aborted, we want to make sure that the final patch is applied. Reusing the context from the reconciliation
	// would cause the patch to be aborted as well.
	return m.patchHelper.Patch(
		timeoutCtx,
		m.IonosMachine,
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
