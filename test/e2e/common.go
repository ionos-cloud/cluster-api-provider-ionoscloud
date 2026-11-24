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

// Package e2e offers end-to-end tests for the Cluster API IONOS Cloud provider.
package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
)

// Test suite constants for e2e config variables.
const (
	CNIPath      = "CNI"
	CNIResources = "CNI_RESOURCES"
	// KubernetesVersion is the workload cluster version.
	KubernetesVersion = "KUBERNETES_VERSION"
	// KubernetesVersionManagement is the kind bootstrap cluster version.
	KubernetesVersionManagement = "KUBERNETES_VERSION_MANAGEMENT"
	// KubernetesVersionFlatcar is the workload cluster version of the Flatcar spec.
	KubernetesVersionFlatcar = "KUBERNETES_VERSION_FLATCAR"
	// FlatcarImageID is the IONOS image ID of the Flatcar node image.
	FlatcarImageID = "IONOSCLOUD_FLATCAR_IMAGE_ID"
	// BootstrapVolumeImageID is the IONOS image ID of the Ignition config volume.
	BootstrapVolumeImageID = "IONOSCLOUD_BOOTSTRAP_VOLUME_IMAGE_ID"

	defaultCloudResourceNamePrefix = "capic-e2e-test-"
)

func Byf(format string, a ...any) {
	By(fmt.Sprintf(format, a...))
}
