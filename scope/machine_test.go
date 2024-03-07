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
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

func exampleParams(t *testing.T) MachineScopeParams {
	if err := infrav1.AddToScheme(scheme.Scheme); err != nil {
		require.NoError(t, err, "could not construct params")
	}
	return MachineScopeParams{
		Client:       fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		Cluster:      &clusterv1.Cluster{},
		Machine:      &clusterv1.Machine{},
		ClusterScope: &ClusterScope{},
		IonosMachine: &infrav1.IonosCloudMachine{},
	}
}

func TestNewMachineScope_OK(t *testing.T) {
	scope, err := NewMachineScope(exampleParams(t))
	require.NotNil(t, scope, "returned machine scope should not be nil")
	require.NoError(t, err)
	require.NotNil(t, scope.patchHelper, "returned scope should have a non-nil patchHelper")
}

func TestMachineScopeParams_NilClientShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.Client = nil
	scope, err := NewMachineScope(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineScopeParams_NilClusterShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.Cluster = nil
	scope, err := NewMachineScope(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineScopeParams_NilMachineShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.Machine = nil
	scope, err := NewMachineScope(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineScopeParams_NilIonosMachineShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.IonosMachine = nil
	scope, err := NewMachineScope(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineScopeParams_NilClusterScopeShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.ClusterScope = nil
	scope, err := NewMachineScope(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineScope_HasFailed_FailureMessage(t *testing.T) {
	scope, err := NewMachineScope(exampleParams(t))
	require.NoError(t, err)
	scope.IonosMachine.Status.FailureMessage = ptr.To("¯\\_(ツ)_/¯")
	require.True(t, scope.HasFailed())
}

func TestMachineScope_HasFailed_FailureReason(t *testing.T) {
	scope, err := NewMachineScope(exampleParams(t))
	require.NoError(t, err)
	scope.IonosMachine.Status.FailureReason = capierrors.MachineStatusErrorPtr("¯\\_(ツ)_/¯")
	require.True(t, scope.HasFailed())
}
