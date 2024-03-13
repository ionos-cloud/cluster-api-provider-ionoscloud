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
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

func TestNewCluster_MissingParams(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, infrav1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	tests := []struct {
		name    string
		params  ClusterParams
		wantErr bool
	}{
		{
			name: "all present",
			params: ClusterParams{
				Client:       cl,
				Cluster:      &clusterv1.Cluster{},
				IonosCluster: &infrav1.IonosCloudCluster{},
			},
			wantErr: false,
		},
		{
			name: "missing client",
			params: ClusterParams{
				Cluster:      &clusterv1.Cluster{},
				IonosCluster: &infrav1.IonosCloudCluster{},
			},
			wantErr: true,
		},
		{
			name: "missing cluster",
			params: ClusterParams{
				Client:       cl,
				IonosCluster: &infrav1.IonosCloudCluster{},
			},
			wantErr: true,
		},
		{
			name: "missing IONOS cluster",
			params: ClusterParams{
				Client:  cl,
				Cluster: &clusterv1.Cluster{},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.wantErr {
				_, err := NewCluster(test.params)
				require.Error(t, err)
			} else {
				params, err := NewCluster(test.params)
				require.NoError(t, err)
				require.NotNil(t, params)
			}
		})
	}
}
