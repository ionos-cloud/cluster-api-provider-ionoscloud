//go:build e2e
// +build e2e

package helpers

import infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"

// IonosCloudInfraFinalizersAssertion maps IONOS Cloud infrastructure resource types to their expected finalizers.
var IonosCloudInfraFinalizersAssertion = map[string][]string{
	"IonosCloudMachine": {infrav1.MachineFinalizer},
	"IonosCloudCluster": {infrav1.ClusterFinalizer},
}
