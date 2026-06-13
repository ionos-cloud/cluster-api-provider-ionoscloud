//go:build e2e

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

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	addonsv1 "sigs.k8s.io/cluster-api/api/addons/v1beta2"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

// IonosCloudInfraFinalizersAssertion maps IONOS Cloud infrastructure resource types to their expected finalizers.
var IonosCloudInfraFinalizersAssertion = map[string]func(types.NamespacedName) []string{
	"IonosCloudMachine":           func(types.NamespacedName) []string { return []string{infrav1.MachineFinalizer} },
	infrav1.IonosCloudClusterKind: func(types.NamespacedName) []string { return []string{infrav1.ClusterFinalizer} },
}

// ExpFinalizersAssertion maps experimental resource types to their expected finalizers.
var ExpFinalizersAssertion = map[string]func(types.NamespacedName) []string{
	kindClusterResourceSet: func(types.NamespacedName) []string { return []string{addonsv1.ClusterResourceSetFinalizer} },
}

// KubernetesFinalizersAssertion maps Kubernetes resource types to their expected finalizers.
func KubernetesFinalizersAssertion(clusters *infrav1.IonosCloudClusterList) map[string]func(types.NamespacedName) []string {
	// Add secret names here that are known to be used by the test suite.
	knownSecrets := sets.New(CloudAPISecretName)
	assertions := map[string]func(types.NamespacedName) []string{}

	if clusters != nil {
		secretAssertions := make([]string, 0, len(clusters.Items))
		for _, cluster := range clusters.Items {
			secretAssertions = append(secretAssertions, fmt.Sprintf("%s/%s", infrav1.ClusterFinalizer, cluster.GetUID()))
		}
		assertions["Secret"] = func(nn types.NamespacedName) []string {
			if knownSecrets.Has(nn.Name) {
				return secretAssertions
			}

			return []string{}
		}
	}
	return assertions
}
