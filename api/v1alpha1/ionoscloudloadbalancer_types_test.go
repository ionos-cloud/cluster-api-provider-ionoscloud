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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	exampleDatacenterID = "fe3b4e3d-3b0e-4e6c-9e3e-4f3c9e3e4f3c"
)

var exampleEndpoint = clusterv1.APIEndpoint{
	Host: "example.com",
	Port: 6443,
}

func defaultLoadBalancer(source LoadBalancerSource) *IonosCloudLoadBalancer {
	return &IonosCloudLoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-loadbalancer",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: IonosCloudLoadBalancerSpec{
			LoadBalancerSource: source,
		},
	}
}

var _ = Describe("IonosCloudLoadBalancer", func() {
	AfterEach(func() {
		err := k8sClient.Delete(context.Background(), defaultLoadBalancer(LoadBalancerSource{}))
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})

	Context("Create", func() {
		When("Using a KubeVIP load balancer", func() {
			It("Should succeed when providing a datacenter ID", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{KubeVIP: &KubeVIPSpec{}})
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())
			})
			It("Should succeed with an endpoint and a port", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{KubeVIP: &KubeVIPSpec{}})
				dlb.Spec.LoadBalancerEndpoint = exampleEndpoint
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())
			})
		})
		When("Using an NLB", func() {
			It("Should fail when not providing a datacenter ID", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{NLB: &NLBSpec{}})
				Expect(k8sClient.Create(context.Background(), dlb)).NotTo(Succeed())
			})
			It("Should succeed when providing a datacenter ID", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{NLB: &NLBSpec{DatacenterID: exampleDatacenterID}})
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())
			})
			It("Should succeed providing an endpoint and a port", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{NLB: &NLBSpec{DatacenterID: exampleDatacenterID}})
				dlb.Spec.LoadBalancerEndpoint = exampleEndpoint
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())
			})
			It("Should fail when providing a host and a port without a datacenter ID", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{NLB: &NLBSpec{}})
				dlb.Spec.LoadBalancerEndpoint = exampleEndpoint
				Expect(k8sClient.Create(context.Background(), dlb)).NotTo(Succeed())
			})
		})
		When("Using an external load balancer", func() {
			It("Should fail when not providing an endpoint", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{External: &ExternalLoadBalancerSpec{}})
				Expect(k8sClient.Create(context.Background(), dlb)).NotTo(Succeed())
			})
			It("Should fail when providing an empty endpoint", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{External: &ExternalLoadBalancerSpec{}})
				dlb.Spec.LoadBalancerEndpoint = clusterv1.APIEndpoint{}
				Expect(k8sClient.Create(context.Background(), dlb)).NotTo(Succeed())
			})
			It("Should fail when providing an endpoint without a port", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{External: &ExternalLoadBalancerSpec{}})
				dlb.Spec.LoadBalancerEndpoint = clusterv1.APIEndpoint{
					Host: "example.com",
				}
				Expect(k8sClient.Create(context.Background(), dlb)).NotTo(Succeed())
			})
			It("Should fail when providing an endpoint without a host", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{External: &ExternalLoadBalancerSpec{}})
				dlb.Spec.LoadBalancerEndpoint = clusterv1.APIEndpoint{
					Port: 6443,
				}
				Expect(k8sClient.Create(context.Background(), dlb)).NotTo(Succeed())
			})
			It("Should succeed when providing an endpoint and a port", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{External: &ExternalLoadBalancerSpec{}})
				dlb.Spec.LoadBalancerEndpoint = exampleEndpoint
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())
			})
		})
		Context("Update", func() {
			It("Should fail when updating the datacenter ID", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{NLB: &NLBSpec{DatacenterID: exampleDatacenterID}})
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())

				dlb.Spec.NLB.DatacenterID = "new-datacenter-id"
				Expect(k8sClient.Update(context.Background(), dlb)).NotTo(Succeed())
			})
			It("Should succeed creating a KubeVIP load balancer with an empty endpoint and updating it", func() {
				dlb := defaultLoadBalancer(LoadBalancerSource{KubeVIP: &KubeVIPSpec{}})
				Expect(k8sClient.Create(context.Background(), dlb)).To(Succeed())

				dlb.Spec.LoadBalancerEndpoint = exampleEndpoint
				Expect(k8sClient.Update(context.Background(), dlb)).To(Succeed())
			})
		})
	})
})
