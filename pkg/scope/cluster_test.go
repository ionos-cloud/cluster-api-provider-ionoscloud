package scope

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/client"
)

func TestNewClusterScope_MissingParams(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, infrav1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name    string
		params  ClusterScopeParams
		wantErr bool
	}{
		{
			name: "all present",
			params: ClusterScopeParams{
				Client:       cl,
				Cluster:      &clusterv1.Cluster{},
				IonosCluster: &infrav1.IonosCloudCluster{},
				IonosClient:  &client.IonosCloudClient{},
			},
			wantErr: false,
		},
		{
			name: "missing client",
			params: ClusterScopeParams{
				Cluster:      &clusterv1.Cluster{},
				IonosCluster: &infrav1.IonosCloudCluster{},
				IonosClient:  &client.IonosCloudClient{},
			},
			wantErr: true,
		},
		{
			name: "missing cluster",
			params: ClusterScopeParams{
				Client:       cl,
				IonosCluster: &infrav1.IonosCloudCluster{},
				IonosClient:  &client.IonosCloudClient{},
			},
			wantErr: true,
		},
		{
			name: "missing IONOS cluster",
			params: ClusterScopeParams{
				Client:      cl,
				Cluster:     &clusterv1.Cluster{},
				IonosClient: &client.IonosCloudClient{},
			},
			wantErr: true,
		},
		{
			name: "missing IONOS client",
			params: ClusterScopeParams{
				Client:       cl,
				Cluster:      &clusterv1.Cluster{},
				IonosCluster: &infrav1.IonosCloudCluster{},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.wantErr {
				_, err := NewClusterScope(test.params)
				require.Error(t, err)
			} else {
				params, err := NewClusterScope(test.params)
				require.NoError(t, err)
				require.NotNil(t, params)
			}
		})
	}
}
