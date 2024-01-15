package scope

import (
	"github.com/go-logr/logr"
	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func exampleParams(t *testing.T) MachineScopeParams {
	if err := infrav1.AddToScheme(scheme.Scheme); err != nil {
		require.NoError(t, err, "could not construct params")
	}
	logger := logr.Discard()
	return MachineScopeParams{
		Client:       fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		Logger:       &logger,
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

func TestMachineScopeParams_NilLoggerShouldWork(t *testing.T) {
	params := exampleParams(t)
	params.Logger = nil
	scope, err := NewMachineScope(params)
	require.NotNil(t, scope, "returned machine scope shouldn't be nil")
	require.NoError(t, err)
	require.NotNil(t, scope.Logger, "logger should not be nil")
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
