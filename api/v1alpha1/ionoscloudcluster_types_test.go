package v1alpha1

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIonosCloudCluster_Conditions(t *testing.T) {
	conditions := clusterv1.Conditions{{Type: "type"}}
	cluster := &IonosCloudCluster{}

	cluster.SetConditions(conditions)
	require.Equal(t, conditions, cluster.GetConditions())
}

func defaultCluster() *IonosCloudCluster {
	return &IonosCloudCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: IonosCloudClusterSpec{
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
				Host: "1.2.3.4",
				Port: 5678,
			},
			ContractNumber: "12345678",
		},
	}
}

var _ = Describe("IonosCloudCluster", func() {
	AfterEach(func() {
		err := k8sClient.Delete(context.Background(), defaultCluster())
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})

	Context("Create", func() {
		It("should allow creating valid clusters", func() {
			Expect(k8sClient.Create(context.Background(), defaultCluster())).To(Succeed())
		})
	})

	Context("Update", func() {
		It("should not allow changing the contract number", func() {
			cluster := defaultCluster()
			Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())

			cluster.Spec.ContractNumber = "changed"
			Expect(k8sClient.Update(context.Background(), cluster)).Should(MatchError(ContainSubstring("contractNumber is immutable")))
		})
	})
})
