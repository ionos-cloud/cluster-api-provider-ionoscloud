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
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
				require.Equal(t, net.DefaultResolver, params.resolver)
			}
		})
	}
}

type mockResolver struct {
	addrs map[string][]netip.Addr
}

func (m *mockResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	return m.addrs[host], nil
}

func resolvesTo(ips ...string) []netip.Addr {
	res := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		res = append(res, netip.MustParseAddr(ip))
	}
	return res
}

func TestCluster_GetControlPlaneEndpointIP(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		resolver resolver
		want     string
	}{
		{
			name: "host empty",
			host: "",
			want: "",
		},
		{
			name: "host is IP",
			host: "127.0.0.1",
			want: "127.0.0.1",
		},
		{
			name: "host is FQDN with single IP",
			host: "localhost",
			resolver: &mockResolver{
				addrs: map[string][]netip.Addr{
					"localhost": resolvesTo("127.0.0.1"),
				},
			},
			want: "127.0.0.1",
		},
		{
			name: "host is FQDN with multiple IPs",
			host: "example.org",
			resolver: &mockResolver{
				addrs: map[string][]netip.Addr{
					"example.org": resolvesTo("2.3.4.5", "1.2.3.4"),
				},
			},
			want: "1.2.3.4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cluster{
				resolver: tt.resolver,
				IonosCluster: &infrav1.IonosCloudCluster{
					Spec: infrav1.IonosCloudClusterSpec{
						ControlPlaneEndpoint: clusterv1.APIEndpoint{
							Host: tt.host,
						},
					},
				},
			}
			got, err := c.GetControlPlaneEndpointIP(context.Background())
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
