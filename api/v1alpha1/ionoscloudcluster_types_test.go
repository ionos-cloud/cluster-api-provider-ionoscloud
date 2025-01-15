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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	// newValueStr is a string that represents a changed value.
	newValueStr = "changed"
	exampleIP   = "198.51.100.1"
)

func TestIonosCloudCluster_Conditions(t *testing.T) {
	conds := clusterv1.Conditions{{Type: "type"}}
	cluster := &IonosCloudCluster{}

	cluster.SetConditions(conds)
	require.Equal(t, conds, cluster.GetConditions())
}

func defaultCluster() *IonosCloudCluster {
	return &IonosCloudCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: IonosCloudClusterSpec{
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
				Host: exampleIP,
				Port: 5678,
			},
			Location:       "de/txl",
			CredentialsRef: corev1.LocalObjectReference{Name: "secret-name"},
			LoadBalancerProviderRef: &corev1.LocalObjectReference{
				Name: "fake-lb-provider",
			},
		},
	}
}

var _ = Describe("IonosCloudCluster", func() {
	AfterEach(func() {
		err := k8sClient.Delete(context.Background(), defaultCluster())
		Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
	})

	Context("Create", func() {
		It("should allow creating valid clusters", func() {
			Expect(k8sClient.Create(context.Background(), defaultCluster())).To(Succeed())
		})
		It("should work with a FQDN controlplane endpoint", func() {
			cluster := defaultCluster()
			cluster.Spec.ControlPlaneEndpoint.Host = "example.org"
			Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())
		})
		It("should not allow creating clusters with empty credential secret", func() {
			cluster := defaultCluster()
			cluster.Spec.CredentialsRef.Name = ""
			Expect(k8sClient.Create(context.Background(), cluster)).
				Should(MatchError(ContainSubstring("credentialsRef.name must be provided")))
		})
		It("should not allow creating clusters with empty location when loadBalancerProviderRef is set", func() {
			cluster := defaultCluster()
			cluster.Spec.Location = ""
			Expect(k8sClient.Create(context.Background(), cluster)).
				Should(MatchError(ContainSubstring("location is required when loadBalancerProviderRef is set")))
		})
		It("should allow creating clusters with empty location when ControlPlaneEndpoint host is not set", func() {
			cluster := defaultCluster()
			cluster.Spec.LoadBalancerProviderRef = nil
			cluster.Spec.Location = ""
			cluster.Spec.ControlPlaneEndpoint.Host = ""
			Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())
		})
	})

	Context("Update", func() {
		It("should not allow changing the location", func() {
			cluster := defaultCluster()
			Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())

			cluster.Spec.Location = newValueStr
			Expect(k8sClient.Update(context.Background(), cluster)).ToNot(Succeed())
		})

		When("trying to update the control plane endpoint", func() {
			It("should fail when attempting to set an invalid port number", func() {
				cluster := defaultCluster()
				Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())

				cluster.Spec.ControlPlaneEndpoint.Port = 0
				Expect(k8sClient.Update(context.Background(), cluster)).
					Should(MatchError(ContainSubstring("port must be within 1-65535")))
			})
			It("should not fail when updating the endpoint correctly", func() {
				cluster := defaultCluster()
				Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())

				cluster.Spec.ControlPlaneEndpoint.Port = 1234
				cluster.Spec.ControlPlaneEndpoint.Host = "example.org"
				Expect(k8sClient.Update(context.Background(), cluster)).To(Succeed())
			})
		})
	})
	Context("Status", func() {
		It("should correctly get and set the status", func() {
			By("initially having an empty status")

			cluster := defaultCluster()
			Expect(k8sClient.Create(context.Background(), cluster)).To(Succeed())

			key := client.ObjectKey{Namespace: cluster.Namespace, Name: cluster.Name}
			fetched := &IonosCloudCluster{}
			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Status.Ready).To(BeFalse())
			Expect(fetched.Status.CurrentRequestByDatacenter).To(BeEmpty())
			Expect(fetched.Status.Conditions).To(BeEmpty())

			By("retrieving the cluster and setting the status")
			fetched.Status.Ready = true
			wantProvisionRequest := ProvisioningRequest{
				Method:      "POST",
				RequestPath: "/path/to/resource",
				State:       "QUEUED",
			}
			fetched.Status.CurrentRequestByDatacenter = map[string]ProvisioningRequest{
				"123": wantProvisionRequest,
			}
			conditions.MarkTrue(fetched, clusterv1.ReadyCondition)

			By("updating the cluster status")
			Expect(k8sClient.Status().Update(context.Background(), fetched)).To(Succeed())

			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Status.Ready).To(BeTrue())
			Expect(fetched.Status.CurrentRequestByDatacenter).To(HaveLen(1))
			Expect(fetched.Status.CurrentRequestByDatacenter["123"]).To(Equal(wantProvisionRequest))
			Expect(fetched.Status.Conditions).To(HaveLen(1))
			Expect(conditions.IsTrue(fetched, clusterv1.ReadyCondition)).To(BeTrue())

			By("Removing the entry from the status again")
			delete(fetched.Status.CurrentRequestByDatacenter, "123")
			Expect(k8sClient.Status().Update(context.Background(), fetched)).To(Succeed())

			Expect(k8sClient.Get(context.Background(), key, fetched)).To(Succeed())
			Expect(fetched.Status.CurrentRequestByDatacenter).To(BeEmpty())
		})
	})
})
