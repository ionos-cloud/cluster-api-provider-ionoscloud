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

package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

func TestSyncFailureDomains(t *testing.T) {
	t.Run("populates status from spec", func(t *testing.T) {
		ionosCluster := &infrav1.IonosCloudCluster{
			Spec: infrav1.IonosCloudClusterSpec{
				FailureDomains: []infrav1.AvailabilityZone{
					infrav1.AvailabilityZoneOne, infrav1.AvailabilityZoneTwo,
				},
			},
		}

		syncFailureDomains(ionosCluster)

		require.Equal(t, clusterv1.FailureDomains{
			"ZONE_1": {ControlPlane: true},
			"ZONE_2": {ControlPlane: true},
		}, ionosCluster.Status.FailureDomains)
	})

	t.Run("clears status when spec is empty", func(t *testing.T) {
		ionosCluster := &infrav1.IonosCloudCluster{
			Status: infrav1.IonosCloudClusterStatus{
				FailureDomains: clusterv1.FailureDomains{"ZONE_1": {ControlPlane: true}},
			},
		}

		syncFailureDomains(ionosCluster)

		require.Nil(t, ionosCluster.Status.FailureDomains)
	})
}
