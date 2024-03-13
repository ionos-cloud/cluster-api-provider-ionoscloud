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

func exampleParams(t *testing.T) MachineParams {
	if err := infrav1.AddToScheme(scheme.Scheme); err != nil {
		require.NoError(t, err, "could not construct params")
	}
	return MachineParams{
		Client:       fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		Machine:      &clusterv1.Machine{},
		ClusterScope: &Cluster{},
		IonosMachine: &infrav1.IonosCloudMachine{},
	}
}

func TestNewMachineOK(t *testing.T) {
	scope, err := NewMachine(exampleParams(t))
	require.NotNil(t, scope, "returned machine scope should not be nil")
	require.NoError(t, err)
	require.NotNil(t, scope.patchHelper, "returned scope should have a non-nil patchHelper")
}

func TestMachineParamsNilClientShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.Client = nil
	scope, err := NewMachine(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineParamsNilMachineShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.Machine = nil
	scope, err := NewMachine(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineParamsNilIonosMachineShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.IonosMachine = nil
	scope, err := NewMachine(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineParamsNilClusterScopeShouldFail(t *testing.T) {
	params := exampleParams(t)
	params.ClusterScope = nil
	scope, err := NewMachine(params)
	require.Nil(t, scope, "returned machine scope should be nil")
	require.Error(t, err)
}

func TestMachineHasFailedFailureMessage(t *testing.T) {
	scope, err := NewMachine(exampleParams(t))
	require.NoError(t, err)
	scope.IonosMachine.Status.FailureMessage = ptr.To("¯\\_(ツ)_/¯")
	require.True(t, scope.HasFailed())
}

func TestMachineHasFailedFailureReason(t *testing.T) {
	scope, err := NewMachine(exampleParams(t))
	require.NoError(t, err)
	scope.IonosMachine.Status.FailureReason = capierrors.MachineStatusErrorPtr("¯\\_(ツ)_/¯")
	require.True(t, scope.HasFailed())
}
