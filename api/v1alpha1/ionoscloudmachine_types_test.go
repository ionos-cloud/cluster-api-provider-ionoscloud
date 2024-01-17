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
			Disk: &Volume{
				Name:             "disk",
				DiskType:         VolumeDiskTypeSSDStandard,
				SizeGB:           23,
				AvailabilityZone: AvailabilityZoneOne,
				SSHKeys:          []string{"public-key"},
			},
			AdditionalNetworks: Networks{
				{
					NetworkID: 1,
				},
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
		It("should work if everything is set properly", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
		})

		Context("Provider ID", func() {
			It("should work if not set", func() {
				m := defaultMachine()
				want := ""
				m.Spec.ProviderID = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.ProviderID).To(Equal(want))
			})
		})

		Context("Data center ID", func() {
			It("should fail if not set", func() {
				m := defaultMachine()
				m.Spec.DataCenterID = ""
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})

			It("should fail if not a UUID", func() {
				m := defaultMachine()
				want := "not-a-UUID"
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

		Context("Number of cores", func() {
			It("should fail if less than 1", func() {
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
			It("should default to 1", func() {
				m := defaultMachine()
				// because NumCores is int32, setting the value as 0 is the same as not setting anything
				m.Spec.NumCores = 0
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.NumCores).To(Equal(int32(1)))
			})
		})

		Context("Availability zone", func() {
			It("should default to AUTO", func() {
				m := defaultMachine()
				// because AvailabilityZone is a string, setting the value as "" is the same as not setting anything
				m.Spec.AvailabilityZone = ""
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.AvailabilityZone).To(Equal(AvailabilityZoneAuto))
			})
			It("should fail if not part of the enum", func() {
				m := defaultMachine()
				m.Spec.AvailabilityZone = "this-should-not-work"
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			DescribeTable("should work for value",
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
			It("Should fail for ZONE_3", func() {
				m := defaultMachine()
				m.Spec.AvailabilityZone = AvailabilityZoneThree
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
		})

		Context("Memory size", func() {
			It("should default to 3072MB", func() {
				m := defaultMachine()
				// because MemoryMB is an int32, setting the value as 0 is the same as not setting anything
				m.Spec.MemoryMB = 0
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.MemoryMB).To(Equal(int32(3072)))
			})
			It("should be at least 2048, therefore less than it should fail", func() {
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
			It("should be a multiple of 1024", func() {
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

		Context("CPU family", func() {
			It("should fail if not set", func() {
				m := defaultMachine()
				m.Spec.CPUFamily = ""
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
		})

		Context("Disk", func() {
			It("should fail if not set", func() {
				m := defaultMachine()
				m.Spec.Disk = nil
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			It("can have an optional name", func() {
				m := defaultMachine()
				want := ""
				m.Spec.Disk.Name = want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Disk.Name).To(Equal(want))
			})
			Context("Availability zone", func() {
				It("should default to AUTO", func() {
					m := defaultMachine()
					// because AvailabilityZone is a string, setting the value as "" is the same as not setting anything
					m.Spec.Disk.AvailabilityZone = ""
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.AvailabilityZone).To(Equal(AvailabilityZoneAuto))
				})
				It("should fail if not part of the enum", func() {
					m := defaultMachine()
					m.Spec.Disk.AvailabilityZone = "this-should-not-work"
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				DescribeTable("should work for value",
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
			Context("SSH keys", func() {
				It("can be created without them", func() {
					m := defaultMachine()
					var want []string
					m.Spec.Disk.SSHKeys = want
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.SSHKeys).To(Equal(want))
				})
				It("should prevent duplicates", func() {
					m := defaultMachine()
					m.Spec.Disk.SSHKeys = []string{"Key1", "Key1", "Key2", "Key3"}
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
			})
			Context("Size (in GB)", func() {
				It("should fail if less than 10", func() {
					m := defaultMachine()
					m.Spec.Disk.SizeGB = 9
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				It("should default to 20", func() {
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
			Context("Type", func() {
				It("should default to HDD", func() {
					m := defaultMachine()
					// because DiskType is a string, setting the value as "" is the same as not setting anything
					m.Spec.Disk.DiskType = ""
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.DiskType).To(Equal(VolumeDiskTypeHDD))
				})
				It("should fail if not part of the enum", func() {
					m := defaultMachine()
					m.Spec.Disk.AvailabilityZone = "tape"
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				DescribeTable("should work for value",
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
		Context("Additional Networks", func() {
			It("network config should be optional", func() {
				m := defaultMachine()
				m.Spec.AdditionalNetworks = nil
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.AdditionalNetworks).To(BeNil())
			})
			It("network ID must be greater than 0", func() {
				m := defaultMachine()
				m.Spec.AdditionalNetworks[0].NetworkID = 0
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				m.Spec.AdditionalNetworks[0].NetworkID = -1
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
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
