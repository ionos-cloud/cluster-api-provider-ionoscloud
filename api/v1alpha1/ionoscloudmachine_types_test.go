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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func defaultMachine() *IonosCloudMachine {
	return &IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: IonosCloudMachineSpec{
			ProviderID:       "ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a",
			DataCenterID:     "ee090ff2-1eef-48ec-a246-a51a33aa4f3a",
			NumCores:         1,
			AvailabilityZone: AvailabilityZoneTwo,
			MemoryMB:         2048,
			CPUFamily:        "AMD_OPTERON",
			Disk: Volume{
				Name:             "disk",
				DiskType:         VolumeDiskTypeSSDStandard,
				SizeGB:           23,
				AvailabilityZone: AvailabilityZoneOne,
				SSHKeys:          []string{"public-key"},
			},
			Network: &Network{
				IPs:     []string{"1.2.3.4"},
				UseDHCP: ptr.To(true),
			},
		},
	}
}

var _ = Describe("IonosCloudMachine Tests", func() {
	AfterEach(func() {
		m := &IonosCloudMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-machine",
				Namespace: metav1.NamespaceDefault,
			},
		}
		err := k8sClient.Delete(context.Background(), m)
		Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
	})

	Context("Validation", func() {
		It("shouldn't fail if everything is seems to be properly set", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
		})

		It("should not fail if providerID is empty", func() {
			m := defaultMachine()
			want := ""
			m.Spec.ProviderID = want
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.ProviderID).To(Equal(want))
		})

		When("data center id", func() {
			It("it should fail if data center ID is not a UUID", func() {
				m := defaultMachine()
				want := ""
				m.Spec.DataCenterID = want
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			It("should be immutable", func() {
				m := defaultMachine()
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.DataCenterID).To(Equal(defaultMachine().Spec.DataCenterID))
				m.Spec.DataCenterID = "6ded8c5f-8df2-46ef-b4ce-61833daf0961"
				Expect(k8sClient.Update(context.Background(), m)).ToNot(Succeed())
			})
		})

		When("the number of cores, ", func() {
			It("is less than 1, it should fail", func() {
				m := defaultMachine()
				m.Spec.NumCores = -1
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			It("should have a minimum value of 1", func() {
				m := defaultMachine()
				want := int32(1)
				m.Spec.NumCores = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.NumCores).To(Equal(want))
			})
			It("isn't set, it should work and default to 1", func() {
				m := defaultMachine()
				// because NumCores is int32, setting the value as 0 is the same as not setting anything
				m.Spec.NumCores = 0
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.NumCores).To(Equal(int32(1)))
			})
		})

		When("the machine availability zone", func() {
			It("isn't set, should default to AUTO", func() {
				m := defaultMachine()
				// because AvailabilityZone is a string, setting the value as "" is the same as not setting anything
				m.Spec.AvailabilityZone = ""
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.AvailabilityZone).To(Equal(AvailabilityZoneAuto))
			})
			It("it not part of the enum it should not work", func() {
				m := defaultMachine()
				m.Spec.AvailabilityZone = "this-should-not-work"
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			DescribeTable("should work for these values",
				func(zone AvailabilityZone) {
					m := defaultMachine()
					m.Spec.AvailabilityZone = zone
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.AvailabilityZone).To(Equal(zone))
				},
				Entry("AUTO", AvailabilityZoneAuto),
				Entry("ZONE_1", AvailabilityZoneOne),
				Entry("ZONE_2", AvailabilityZoneTwo),
			)
			It("Should not work for ZONE_3", func() {
				m := defaultMachine()
				m.Spec.AvailabilityZone = AvailabilityZoneThree
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
		})

		When("the machine memory size", func() {
			It("isn't set, should default to 3072MB", func() {
				m := defaultMachine()
				// because MemoryMB is an int32, setting the value as 0 is the same as not setting anything
				m.Spec.MemoryMB = 0
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.MemoryMB).To(Equal(int32(3072)))
			})
			It("should be at least 2048, therefore less than it should not work", func() {
				m := defaultMachine()
				m.Spec.MemoryMB = 1024
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			It("should be at least 2048, therefore 2048 should work", func() {
				m := defaultMachine()
				want := int32(2048)
				m.Spec.MemoryMB = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.MemoryMB).To(Equal(want))
			})
			It("it should be a multiple of 1024", func() {
				m := defaultMachine()
				m.Spec.MemoryMB = 2100
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			It("should be at least 2048 and a multiple of 1024, therefore 4096 should work", func() {
				m := defaultMachine()
				want := int32(4096)
				m.Spec.MemoryMB = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.MemoryMB).To(Equal(want))
			})
		})
		When("the machine CPU family", func() {
			It("isn't set, it should fail", func() {
				m := defaultMachine()
				m.Spec.CPUFamily = ""
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
		})

		Context("Volume", func() {
			It("can have an optional name", func() {
				m := defaultMachine()
				want := ""
				m.Spec.Disk.Name = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Disk.Name).To(Equal(want))
			})
			When("the disk availability zone", func() {
				It("isn't set, should default to AUTO", func() {
					m := defaultMachine()
					// because AvailabilityZone is a string, setting the value as "" is the same as not setting anything
					m.Spec.Disk.AvailabilityZone = ""
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.AvailabilityZone).To(Equal(AvailabilityZoneAuto))
				})
				It("is not part of the enum it should not work", func() {
					m := defaultMachine()
					m.Spec.Disk.AvailabilityZone = "this-should-not-work"
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				DescribeTable("should work for these values",
					func(zone AvailabilityZone) {
						m := defaultMachine()
						m.Spec.Disk.AvailabilityZone = zone
						Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
						Expect(m.Spec.Disk.AvailabilityZone).To(Equal(zone))
					},
					Entry("AUTO", AvailabilityZoneAuto),
					Entry("ZONE_1", AvailabilityZoneOne),
					Entry("ZONE_2", AvailabilityZoneTwo),
					Entry("ZONE_3", AvailabilityZoneThree),
				)
			})
			It("can be created without SSH keys", func() {
				m := defaultMachine()
				var want []string
				m.Spec.Disk.SSHKeys = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Disk.SSHKeys).To(Equal(want))
			})
			It("should prevent setting identical SSH keys", func() {
				m := defaultMachine()
				m.Spec.Disk.SSHKeys = []string{"Key1", "Key1", "Key2", "Key3"}
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			When("the disk size (in GB)", func() {
				It("is less than 10, it should fail", func() {
					m := defaultMachine()
					m.Spec.Disk.SizeGB = 9
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				It("it is not set, it should default to 20", func() {
					m := defaultMachine()
					// Because disk size is an int, setting it as 0 is the same as not setting anything
					m.Spec.Disk.SizeGB = 0
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.SizeGB).To(Equal(20))
				})
				It("should be at least 10; therefore 10 should work", func() {
					m := defaultMachine()
					want := 10
					m.Spec.Disk.SizeGB = want
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.SizeGB).To(Equal(want))
				})
			})
			When("the disk type", func() {
				It("isn't set, should default to HDD", func() {
					m := defaultMachine()
					// because DiskType is a string, setting the value as "" is the same as not setting anything
					m.Spec.Disk.DiskType = ""
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.DiskType).To(Equal(VolumeDiskTypeHDD))
				})
				It("is not part of the enum it should not work", func() {
					m := defaultMachine()
					m.Spec.Disk.AvailabilityZone = "tape"
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				DescribeTable("should work for these values",
					func(diskType VolumeDiskType) {
						m := defaultMachine()
						m.Spec.Disk.DiskType = diskType
						Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
						Expect(m.Spec.Disk.DiskType).To(Equal(diskType))
					},
					Entry("HDD", VolumeDiskTypeHDD),
					Entry("SSD Standard", VolumeDiskTypeSSDStandard),
					Entry("SSD Premium", VolumeDiskTypeSSDPremium),
				)
			})
		})
		Context("Network", func() {
			It("network config should be optional", func() {
				m := defaultMachine()
				m.Spec.Network = nil
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Network).To(BeNil())
			})
			It("if UseDHCP is not set, it should default to true", func() {
				m := defaultMachine()
				m.Spec.Network.UseDHCP = nil
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Network.UseDHCP).ToNot(BeNil())
				Expect(*m.Spec.Network.UseDHCP).To(BeTrue())
			})
			DescribeTable("if set UseDHCP can be",
				func(useDHCP *bool) {
					m := defaultMachine()
					m.Spec.Network.UseDHCP = useDHCP
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Network.UseDHCP).To(Equal(useDHCP))
				},
				Entry("true", ptr.To(true)),
				Entry("false", ptr.To(false)),
			)
			It("reserved IPs should be optional", func() {
				m := defaultMachine()
				m.Spec.Network.IPs = nil
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Network.IPs).To(BeNil())
			})
		})
		Context("Conditions", func() {
			It("should correctly set and get the conditions", func() {
				m := defaultMachine()
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

				// Calls SetConditions with required fields
				conditions.MarkTrue(m, MachineProvisionedCondition)

				Expect(k8sClient.Status().Update(context.Background(), m)).To(Succeed())
				Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

				machineConditions := m.GetConditions()
				Expect(machineConditions).To(HaveLen(1))
				Expect(machineConditions[0].Type).To(Equal(MachineProvisionedCondition))
				Expect(machineConditions[0].Status).To(Equal(corev1.ConditionTrue))
			})
		})
	})
})
