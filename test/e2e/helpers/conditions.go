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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterctlcluster "sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/cluster-api/test/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"

	. "github.com/onsi/gomega"
)

// AssertConditionsMigration returns a PostUpgrade hook that verifies CAPIC resources
// carry the v1beta2 status fields after upgrading from v0.6.x to v0.8+.
func AssertConditionsMigration(ctx context.Context) func(framework.ClusterProxy, string, string) {
	return func(proxy framework.ClusterProxy, namespace, _ string) {
		c := proxy.GetClient()

		clusters := &infrav1.IonosCloudClusterList{}
		Expect(c.List(ctx, clusters, runtimeclient.InNamespace(namespace))).To(Succeed())
		Expect(clusters.Items).NotTo(BeEmpty(),
			"expected at least one IonosCloudCluster in namespace %s", namespace)

		for i := range clusters.Items {
			assertClusterConditionsMigrated(&clusters.Items[i])
		}

		machines := &infrav1.IonosCloudMachineList{}
		Expect(c.List(ctx, machines, runtimeclient.InNamespace(namespace))).To(Succeed())
		Expect(machines.Items).NotTo(BeEmpty(),
			"expected at least one IonosCloudMachine in namespace %s", namespace)

		for i := range machines.Items {
			assertMachineConditionsMigrated(&machines.Items[i])
		}
	}
}

// AssertCAPIV1Beta2Migration returns a PostUpgrade hook that verifies CAPI core
// v1beta2 types are active and the Cluster is available after upgrading to v1.11+.
func AssertCAPIV1Beta2Migration(ctx context.Context) func(framework.ClusterProxy, string, string) {
	return func(proxy framework.ClusterProxy, namespace, clusterName string) {
		c := proxy.GetClient()

		// Assert the clusters.cluster.x-k8s.io CRD storage migration to v1beta2 has completed
		// (status.storedVersions == [v1beta2], not just spec.versions[].storage).
		framework.ValidateCRDMigration(ctx, proxy, namespace, clusterName,
			func(crd apiextensionsv1.CustomResourceDefinition) bool {
				return crd.Name == "clusters.cluster.x-k8s.io"
			},
			clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName))

		// Assert Cluster has Available=True.
		framework.VerifyClusterAvailable(ctx, framework.VerifyClusterAvailableInput{
			Getter:    proxy.GetClient(),
			Name:      clusterName,
			Namespace: namespace,
		})

		// Assert no continuous reconcile loop (resource versions stable).
		framework.ValidateResourceVersionStable(ctx, framework.ValidateResourceVersionStableInput{
			ClusterProxy:             proxy,
			Namespace:                namespace,
			OwnerGraphFilterFunction: clusterctlcluster.FilterClusterObjectsWithNameFilter(clusterName),
		})

		// Assert CAPIC resources also carry the v1beta2 fields (upgrade included CAPIC too).
		clusters := &infrav1.IonosCloudClusterList{}
		Expect(c.List(ctx, clusters, runtimeclient.InNamespace(namespace))).To(Succeed())
		for i := range clusters.Items {
			assertClusterConditionsMigrated(&clusters.Items[i])
		}

		machines := &infrav1.IonosCloudMachineList{}
		Expect(c.List(ctx, machines, runtimeclient.InNamespace(namespace))).To(Succeed())
		for i := range machines.Items {
			assertMachineConditionsMigrated(&machines.Items[i])
		}
	}
}

func assertClusterConditionsMigrated(cluster *infrav1.IonosCloudCluster) {
	assertConditionsMigrated("IonosCloudCluster", cluster.Name,
		cluster.Status.Initialization.Provisioned, cluster.Status.Conditions)
}

func assertMachineConditionsMigrated(machine *infrav1.IonosCloudMachine) {
	assertConditionsMigrated("IonosCloudMachine", machine.Name,
		machine.Status.Initialization.Provisioned, machine.Status.Conditions)
}

// assertConditionsMigrated verifies the v1beta2 status shape shared by IonosCloudCluster
// and IonosCloudMachine after upgrading from v0.6.x/v0.7.x to v0.8+, including that any
// condition carried over from the old v1beta1 conditions API (empty Reason) was backfilled
// by api/v1alpha1.BackfillLegacyConditionReasons rather than rejected by the new schema.
func assertConditionsMigrated(kind, name string, provisioned *bool, conditions []metav1.Condition) {
	// status.initialization.provisioned must be set and true.
	Expect(provisioned).NotTo(BeNil(),
		"%s %s: status.initialization.provisioned must be set after upgrade", kind, name)
	Expect(*provisioned).To(BeTrue(),
		"%s %s: status.initialization.provisioned must be true after upgrade", kind, name)

	// status.conditions (metav1.Condition) must be non-empty.
	Expect(conditions).NotTo(BeEmpty(),
		"%s %s: status.conditions must be non-empty after upgrade", kind, name)
	hasTrue := false
	for _, cond := range conditions {
		if cond.Status == metav1.ConditionTrue {
			hasTrue = true
		}
		Expect(cond.Reason).NotTo(BeEmpty(),
			"%s %s: condition %q must carry a non-empty Reason after upgrade "+
				"(legacy v1beta1 conditions must be backfilled, see api/v1alpha1.BackfillLegacyConditionReasons)",
			kind, name, cond.Type)
	}
	Expect(hasTrue).To(BeTrue(),
		"%s %s: at least one metav1.Condition must have Status=True", kind, name)
}
