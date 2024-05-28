//go:build e2e
// +build e2e

package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
)

// Test suite constants for e2e config variables.
const (
	CNIPath                     = "CNI"
	CNIResources                = "CNI_RESOURCES"
	KubernetesVersionManagement = "KUBERNETES_VERSION_MANAGEMENT"
)

const (
	infrastructureProvider = "ionoscloud"
)

func Byf(format string, a ...interface{}) {
	By(fmt.Sprintf(format, a...))
}
