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
	"context"

	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // only contract CAPI v1.10.x serves
	"sigs.k8s.io/cluster-api/test/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/gomega"
)

// AssertV1Beta1ClusterAndMachinesHealthy is a PostUpgrade hook checking Cluster/Machine
// Status.Phase only — safe to use when the upgrade target doesn't serve v1beta2 yet.
func AssertV1Beta1ClusterAndMachinesHealthy(ctx context.Context) func(framework.ClusterProxy, string, string) {
	return func(proxy framework.ClusterProxy, namespace, clusterName string) {
		c := proxy.GetClient()

		cluster := &clusterv1beta1.Cluster{}
		Expect(c.Get(ctx, runtimeclient.ObjectKey{Namespace: namespace, Name: clusterName}, cluster)).To(Succeed())
		Expect(cluster.Status.Phase).To(Equal(string(clusterv1beta1.ClusterPhaseProvisioned)),
			"Cluster %s should be in the Provisioned phase", clusterName)

		machines := &clusterv1beta1.MachineList{}
		Expect(c.List(ctx, machines, runtimeclient.InNamespace(namespace),
			runtimeclient.MatchingLabels{clusterv1beta1.ClusterNameLabel: clusterName})).To(Succeed())
		Expect(machines.Items).NotTo(BeEmpty(), "expected at least one Machine for cluster %s", clusterName)

		for _, m := range machines.Items {
			Expect(m.Status.Phase).To(Equal(string(clusterv1beta1.MachinePhaseRunning)),
				"Machine %s should be in the Running phase", m.Name)
		}
	}
}
