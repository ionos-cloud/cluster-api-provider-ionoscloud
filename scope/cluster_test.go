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

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

func TestNewClusterMissingParams(t *testing.T) {
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

func TestClusterListMachinesForCluster(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, infrav1.AddToScheme(scheme))

	const clusterName = "test-cluster"

	makeLabels := func(clusterName string, additionalLabels map[string]string) map[string]string {
		if additionalLabels == nil {
			return map[string]string{clusterv1.ClusterNameLabel: clusterName}
		}

		additionalLabels[clusterv1.ClusterNameLabel] = clusterName
		return additionalLabels
	}

	tests := []struct {
		name           string
		initialObjects []client.Object
		searchLabels   client.MatchingLabels
		expectedNames  map[string]struct{}
	}{{
		name: "List all machines for a cluster",
		initialObjects: []client.Object{
			buildMachineWithLabel("machine-1", makeLabels(clusterName, nil)),
			buildMachineWithLabel("machine-2", makeLabels(clusterName, nil)),
			buildMachineWithLabel("machine-3", makeLabels(clusterName, nil)),
		},
		searchLabels: client.MatchingLabels{},
		expectedNames: map[string]struct{}{
			"machine-1": {},
			"machine-2": {},
			"machine-3": {},
		},
	}, {
		name: "List only machines with specific labels",
		initialObjects: []client.Object{
			buildMachineWithLabel("machine-1", makeLabels(clusterName, map[string]string{"foo": "bar"})),
			buildMachineWithLabel("machine-2", makeLabels(clusterName, map[string]string{"foo": "bar"})),
			buildMachineWithLabel("machine-3", makeLabels(clusterName, nil)),
		},
		searchLabels: client.MatchingLabels{
			"foo": "bar",
		},
		expectedNames: map[string]struct{}{
			"machine-1": {},
			"machine-2": {},
		},
	}, {
		name: "List no machines",
		initialObjects: []client.Object{
			buildMachineWithLabel("machine-1", makeLabels(clusterName, map[string]string{"foo": "notbar"})),
			buildMachineWithLabel("machine-2", makeLabels(clusterName, map[string]string{"foo": "notbar"})),
			buildMachineWithLabel("machine-3", makeLabels(clusterName, map[string]string{"foo": "notbar"})),
		},
		searchLabels:  makeLabels(clusterName, map[string]string{"foo": "bar"}),
		expectedNames: map[string]struct{}{},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := ClusterParams{
				Cluster: &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterName,
						Namespace: metav1.NamespaceDefault,
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: clusterName,
						},
					},
				},
				IonosCluster: &infrav1.IonosCloudCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-ionos-cluster",
						Namespace: metav1.NamespaceDefault,
						Labels: map[string]string{
							clusterv1.ClusterNameLabel: clusterName,
						},
					},
					Status: infrav1.IonosCloudClusterStatus{},
				},
			}

			cl := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(test.initialObjects...).Build()

			params.Client = cl
			cs, err := NewCluster(params)
			require.NoError(t, err)
			require.NotNil(t, cs)

			machines, err := cs.ListMachinesForCluster(context.Background(), test.searchLabels)
			require.NoError(t, err)
			require.Len(t, machines, len(test.expectedNames))

			for _, m := range machines {
				require.Contains(t, test.expectedNames, m.Name)
			}
		})
	}
}

func buildMachineWithLabel(name string, labels map[string]string) *infrav1.IonosCloudMachine {
	return &infrav1.IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
			Labels:    labels,
		},
	}
}
