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

package e2e

import (
	capie2e "sigs.k8s.io/cluster-api/test/e2e"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("When testing K8S conformance", Label("Conformance"), func() {
	// Note: This installs a cluster based on KUBERNETES_VERSION and runs conformance tests.
	capie2e.K8SConformanceSpec(ctx, func() capie2e.K8SConformanceSpecInput {
		return capie2e.K8SConformanceSpecInput{
			E2EConfig:             e2eConfig,
			ClusterctlConfigPath:  clusterctlConfigPath,
			BootstrapClusterProxy: bootstrapClusterProxy,
			ArtifactFolder:        artifactFolder,
			SkipCleanup:           skipCleanup,
			Flavor:                "image-selector",
			PostNamespaceCreated:  cloudEnv.createCredentialsSecretPNC,
		}
	})
})
