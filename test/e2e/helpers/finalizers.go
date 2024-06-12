//go:build e2e
// +build e2e

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

package helpers

import (
	"fmt"

	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

// IonosCloudInfraFinalizersAssertion maps IONOS Cloud infrastructure resource types to their expected finalizers.
var IonosCloudInfraFinalizersAssertion = map[string][]string{
	"IonosCloudMachine": {infrav1.MachineFinalizer},
	"IonosCloudCluster": {infrav1.ClusterFinalizer},
}

// ExpFinalizersAssertion maps experimental resource types to their expected finalizers.
var ExpFinalizersAssertion = map[string][]string{
	"ClusterResourceSet": {addonsv1.ClusterResourceSetFinalizer},
}

// KubernetesFinalizersAssertion maps Kubernetes resource types to their expected finalizers.
func KubernetesFinalizersAssertion(clusters *infrav1.IonosCloudClusterList) map[string][]string {
	assertions := map[string][]string{}
	if clusters != nil {
		secretAssertions := make([]string, 0)
		for _, cluster := range clusters.Items {
			secretAssertions = append(secretAssertions, fmt.Sprintf("%s/%s", infrav1.ClusterFinalizer, cluster.GetUID()))
		}
		assertions["Secret"] = secretAssertions
	}
	return assertions
}
