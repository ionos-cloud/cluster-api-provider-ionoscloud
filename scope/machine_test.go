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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

func exampleParams(t *testing.T) MachineParams {
	if err := infrav1.AddToScheme(scheme.Scheme); err != nil {
		require.NoError(t, err, "could not construct params")
	}
	return MachineParams{
		Client:  fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		Machine: &clusterv1.Machine{},
		ClusterScope: &Cluster{
			Cluster: &clusterv1.Cluster{},
		},
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
	scope.IonosMachine.Status.FailureReason = (*capierrors.MachineStatusError)(ptr.To("¯\\_(ツ)_/¯"))
	require.True(t, scope.HasFailed())
}

func TestCountMachinesWithDifferentLabels(t *testing.T) {
	scope, err := NewMachine(exampleParams(t))
	require.NoError(t, err)

	scope.ClusterScope.Cluster.SetName("test-cluster")

	count, err := scope.CountMachines(
		context.Background(),
		client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""},
	)

	require.NoError(t, err)
	require.Equal(t, 0, count)

	cpLabels := map[string]string{
		clusterv1.ClusterNameLabel:         scope.ClusterScope.Cluster.Name,
		clusterv1.MachineControlPlaneLabel: "",
	}

	cpMachines := []string{"cp-test-1", "cp-test-2", "cp-test-3"}
	for _, machine := range cpMachines {
		err = scope.client.Create(context.Background(), createMachineWithLabels(machine, cpLabels, 0))
		require.NoError(t, err)
	}

	workerLabels := map[string]string{
		clusterv1.ClusterNameLabel: scope.ClusterScope.Cluster.Name,
	}

	workerMachines := []string{"worker-test-1", "worker-test-2", "worker-test-3"}
	for _, machine := range workerMachines {
		err = scope.client.Create(context.Background(), createMachineWithLabels(machine, workerLabels, 0))
		require.NoError(t, err)
	}

	count, err = scope.CountMachines(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 6, count)

	count, err = scope.CountMachines(
		context.Background(),
		client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""},
	)

	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func TestMachineFindLatestMachineWithControlPlaneLabel(t *testing.T) {
	scope, err := NewMachine(exampleParams(t))
	require.NoError(t, err)

	scope.ClusterScope.Cluster.SetName("test-cluster")
	scope.IonosMachine.SetName("cp-test-1")

	controlPlaneLabel := client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""}
	latestMachine, err := scope.FindLatestMachine(context.Background(), controlPlaneLabel)
	require.NoError(t, err)
	require.Nil(t, latestMachine)

	cpLabels := map[string]string{
		clusterv1.ClusterNameLabel:         scope.ClusterScope.Cluster.Name,
		clusterv1.MachineControlPlaneLabel: "",
	}

	cpMachines := []string{"cp-test-1", "cp-test-2", "cp-test-3"}
	for _, machine := range cpMachines {
		err = scope.client.Create(context.Background(), createMachineWithLabels(machine, cpLabels, 0))
		require.NoError(t, err)
	}

	latestMachine, err = scope.FindLatestMachine(context.Background(), controlPlaneLabel)
	require.NoError(t, err)
	require.NotNil(t, latestMachine)
	require.Equal(t, "cp-test-3", latestMachine.Name)

	err = scope.client.Delete(context.Background(), latestMachine)
	require.NoError(t, err)

	latestMachine, err = scope.FindLatestMachine(context.Background(), controlPlaneLabel)
	require.NoError(t, err)
	require.NotNil(t, latestMachine)
	require.Equal(t, "cp-test-2", latestMachine.Name)
}

func TestFindLatestMachineMachineIsReceiver(t *testing.T) {
	scope, err := NewMachine(exampleParams(t))
	require.NoError(t, err)

	scope.ClusterScope.Cluster.SetName("test-cluster")
	scope.IonosMachine.SetName("cp-test-1")

	controlPlaneLabel := client.MatchingLabels{clusterv1.MachineControlPlaneLabel: ""}
	cpLabels := map[string]string{
		clusterv1.ClusterNameLabel:         scope.ClusterScope.Cluster.Name,
		clusterv1.MachineControlPlaneLabel: "",
	}

	cpMachines := []string{"cp-test-2", "cp-test-1"}
	for index, machine := range cpMachines {
		err = scope.client.Create(
			context.Background(),
			createMachineWithLabels(machine, cpLabels, time.Duration(index)*time.Minute),
		)
		require.NoError(t, err)
	}

	latestMachine, err := scope.FindLatestMachine(context.Background(), controlPlaneLabel)
	require.NoError(t, err)
	require.Nil(t, latestMachine)
}

func createMachineWithLabels(name string, labels map[string]string, offset time.Duration) *infrav1.IonosCloudMachine {
	return &infrav1.IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         metav1.NamespaceDefault,
			CreationTimestamp: metav1.NewTime(time.Now().Add(offset)),
			Labels:            labels,
		},
	}
}
