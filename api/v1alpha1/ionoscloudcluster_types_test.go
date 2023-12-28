package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestIonosCloudCluster_Conditions(t *testing.T) {
	conditions := clusterv1.Conditions{{Type: "type"}}
	cluster := &IonosCloudCluster{}

	cluster.SetConditions(conditions)
	require.Equal(t, conditions, cluster.GetConditions())
}
