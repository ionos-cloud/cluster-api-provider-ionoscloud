/*
Copyright 2023 IONOS Cloud.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func defaultMachine() *IonosCloudMachine {
	return &IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: IonosCloudMachineSpec{
			ProviderID: "ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a",
			ServerID:   "11432411-cf83-4f62-b50f-798060493ea9",
			Network:    &Network{},
		},
		Status: IonosCloudMachineStatus{},
	}
}

var _ = Describe("IonosCloudMachine Tests", func() {
	AfterEach(func() {
		err := k8sClient.Delete(context.Background(), defaultMachine())
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})

	Context("Create", func() {
		It("Should allow creation of valid machines", func() {
			Expect(k8sClient.Create(context.Background(), defaultMachine())).To(Succeed())
		})
	})

	Context("Validation", func() {
		It("Should set defaults for Volumes", func() {
			m := defaultMachine()
			m.Spec.Disk = &Volume{
				Name:   "test-volume",
				SizeGB: 5,
			}

			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(context.Background(), client.ObjectKeyFromObject(m), m)).To(Succeed())

			spec := m.Spec
			Expect(spec.NumCores).To(Equal(int32(1)))
			Expect(spec.MemoryMiB).To(Equal(int32(1024)))

			Expect(spec.Disk.DiskType).To(Equal(VolumeDiskTypeHDD))
			Expect(spec.Disk.AvailabilityZone).To(Equal("AUTO"))
			Expect(spec.Disk.SizeGB).To(Equal(5))

			Expect(spec.Network.UseDHCP).ToNot(BeNil())
			Expect(deref(spec.Network.UseDHCP, false)).To(BeTrue())
		})

		It("Should fail if datacenterID would be updated", func() {
			m := defaultMachine()
			m.Spec.DatacenterID = "test"

			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(context.Background(), client.ObjectKeyFromObject(m), m)).To(Succeed())

			m.Spec.DatacenterID = "changed"
			Expect(k8sClient.Update(context.Background(), m)).To(HaveOccurred())
		})

		It("Should fail if size is less than 5", func() {
			m := defaultMachine()
			m.Spec.Disk = &Volume{
				Name:   "test-volume",
				SizeGB: 4,
			}

			Expect(k8sClient.Create(context.Background(), m)).To(MatchError(ContainSubstring("should be greater than or equal to 5")))
		})

		It("Should fail when providing duplicate ip address", func() {
			m := defaultMachine()

			m.Spec.Network.IPs = append(m.Spec.Network.IPs, "192.0.2.0", "192.0.2.1", "192.0.2.1")
			Expect(k8sClient.Create(context.Background(), m)).To(MatchError(ContainSubstring("Duplicate value: \"192.0.2.1\"")))
		})
	})
})

func deref[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}

	return def
}
