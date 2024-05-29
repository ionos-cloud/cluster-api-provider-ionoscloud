//go:build e2e
// +build e2e

package helpers

import (
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
