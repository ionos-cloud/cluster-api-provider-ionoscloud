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

	"github.com/google/go-cmp/cmp"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func defaultMachine() *IonosCloudMachine {
	return &IonosCloudMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: IonosCloudMachineSpec{
			ProviderID:       ptr.To("ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a"),
			DatacenterID:     "ee090ff2-1eef-48ec-a246-a51a33aa4f3a",
			NumCores:         1,
			AvailabilityZone: AvailabilityZoneTwo,
			MemoryMB:         2048,
			CPUFamily:        ptr.To("AMD_OPTERON"),
			Disk: &Volume{
				Name:             "disk",
				DiskType:         VolumeDiskTypeSSDStandard,
				SizeGB:           23,
				AvailabilityZone: AvailabilityZoneOne,
				Image: &ImageSpec{
					ID: "1eef-48ec-a246-a51a33aa4f3a",
				},
			},
			AdditionalNetworks: []Network{
				{
					NetworkID: 1,
				},
			},
		},
	}
}

func setInvalidPoolRef(m *IonosCloudMachine, poolType string, kind, apiGroup, name string) {
	ref := &corev1.TypedLocalObjectReference{
		APIGroup: ptr.To(apiGroup),
		Kind:     kind,
		Name:     name,
	}
	switch poolType {
	case "IPv6":
		m.Spec.AdditionalNetworks[0].IPv6PoolRef = ref
	case "IPv4":
		m.Spec.AdditionalNetworks[0].IPv4PoolRef = ref
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
				m.Spec.ProviderID = &want
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(*m.Spec.ProviderID).To(Equal(want))
			})
			DescribeTable("tests for extraction of provider IDs", func(providerID, want string) {
				m := defaultMachine()
				m.Spec.ProviderID = &providerID
				Expect(m.ExtractServerID()).To(Equal(want))
			},
				Entry("valid ID", "ionos://ee090ff2-1eef-48ec-a246-a51a33aa4f3a",
					"ee090ff2-1eef-48ec-a246-a51a33aa4f3a"),
				Entry("invalid provider name", "ionoscloud://ee090ff2-1eef-48ec-a246-a51a33aa4f3a", ""),
				Entry("typo in provider name", "ions://ee090ff2-1eef-48ec-a246-a51a33aa4f3a", ""),
				Entry("no provider name", "://ee090ff2-1eef-48ec-a246-a51a33aa4f3a", ""),
			)
		})

		Context("Data center ID", func() {
			It("should fail if not set", func() {
				m := defaultMachine()
				m.Spec.DatacenterID = ""
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})

			It("should fail if not a UUID", func() {
				m := defaultMachine()
				want := "not-a-UUID"
				m.Spec.DatacenterID = want
				Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
			})
			It("should be immutable", func() {
				m := defaultMachine()
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.DatacenterID).To(Equal(defaultMachine().Spec.DatacenterID))
				m.Spec.DatacenterID = "6ded8c5f-8df2-46ef-b4ce-61833daf0961"
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

		Context("CPU Family", func() {
			It("should not fail if not set", func() {
				m := defaultMachine()
				m.Spec.CPUFamily = nil
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
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
			Context("DiskType", func() {
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
			Context("Image", func() {
				It("should fail if not set", func() {
					m := defaultMachine()
					m.Spec.Disk.Image = nil
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				It("should fail if no fields are set", func() {
					m := defaultMachine()
					m.Spec.Disk.Image.ID = ""
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				It("should fail if no match labels are set", func() {
					m := defaultMachine()
					m.Spec.Disk.Image.Selector = &ImageSelector{}
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
				It("should not fail if ID is set", func() {
					m := defaultMachine()
					m.Spec.Disk.Image.ID = "1eef-48ec-a246-a51a33aa4f3a"
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				})
				It("should not fail if selector is set", func() {
					m := defaultMachine()
					m.Spec.Disk.Image.ID = ""
					m.Spec.Disk.Image.Selector = &ImageSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					}
					Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
					Expect(m.Spec.Disk.Image.Selector.UseMachineVersion).ToNot(BeNil())
					Expect(*m.Spec.Disk.Image.Selector.UseMachineVersion).To(BeTrue())
				})
				It("should fail if both ID and selector are set", func() {
					m := defaultMachine()
					m.Spec.Disk.Image.ID = "1eef-48ec-a246-a51a33aa4f3a"
					m.Spec.Disk.Image.Selector = &ImageSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					}
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				})
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
			DescribeTable("should allow IPv4PoolRef.Kind GlobalInClusterIPPool and InClusterIPPool", func(kind string) {
				m := defaultMachine()
				m.Spec.AdditionalNetworks[0].IPv4PoolRef = &corev1.TypedLocalObjectReference{
					APIGroup: ptr.To("ipam.cluster.x-k8s.io"),
					Kind:     kind,
					Name:     "ipv4-pool",
				}
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			},
				Entry("GlobalInClusterIPPool", "GlobalInClusterIPPool"),
				Entry("InClusterIPPool", "InClusterIPPool"),
			)
			DescribeTable("should allow IPv6PoolRef.Kind GlobalInClusterIPPool and InClusterIPPool", func(kind string) {
				m := defaultMachine()
				m.Spec.AdditionalNetworks[0].IPv6PoolRef = &corev1.TypedLocalObjectReference{
					APIGroup: ptr.To("ipam.cluster.x-k8s.io"),
					Kind:     kind,
					Name:     "ipv6-pool",
				}
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			},
				Entry("GlobalInClusterIPPool", "GlobalInClusterIPPool"),
				Entry("InClusterIPPool", "InClusterIPPool"),
			)
			DescribeTable("must not allow invalid pool references",
				func(poolType, kind, apiGroup, name string) {
					m := defaultMachine()
					setInvalidPoolRef(m, poolType, kind, apiGroup, name)
					Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
				},
				Entry("invalid IPv6PoolRef with invalid kind", "IPv6", "SomeOtherIPPoolKind", "ipam.cluster.x-k8s.io", "ipv6-pool"),
				Entry("invalid IPv6PoolRef with invalid apiGroup", "IPv6", "InClusterIPPool", "SomeWrongAPIGroup", "ipv6-pool"),
				Entry("invalid IPv6PoolRef with empty name", "IPv6", "InClusterIPPool", "ipam.cluster.x-k8s.io", ""),
				Entry("invalid IPv4PoolRef with invalid kind", "IPv4", "SomeOtherIPPoolKind", "ipam.cluster.x-k8s.io", "ipv4-pool"),
				Entry("invalid IPv4PoolRef with invalid apiGroup", "IPv4", "InClusterIPPool", "SomeWrongAPIGroup", "ipv4-pool"),
				Entry("invalid IPv4PoolRef with empty name", "IPv4", "InClusterIPPool", "ipam.cluster.x-k8s.io", ""),
			)
			It("DHCP should default to true", func() {
				m := defaultMachine()
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(*m.Spec.AdditionalNetworks[0].DHCP).To(BeTrue())
			})
		})
	})
	Context("FailoverIP", func() {
		It("should allow setting AUTO as the value", func() {
			m := defaultMachine()
			m.Spec.FailoverIP = ptr.To(CloudResourceConfigAuto)
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.FailoverIP).To(Equal(ptr.To(CloudResourceConfigAuto)))
		})
		It("should allow setting a valid IPv4 address", func() {
			m := defaultMachine()
			m.Spec.FailoverIP = ptr.To("203.0.113.1")
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.FailoverIP).To(Equal(ptr.To("203.0.113.1")))
		})
		It("should allow setting null", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.FailoverIP).To(BeNil())
		})
		DescribeTable("should not allow setting invalid IPv4 addresses", func(ip string) {
			m := defaultMachine()
			m.Spec.FailoverIP = &ip
			Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
		},
			Entry("IPv4 out of range", "203.0.113.256"),
			Entry("IPv4 missing a block", "203.0.113"),
			Entry("IPv4 ends on a dot", "203.0.113.255."),
			Entry("IPv4 two dots", "203..0.113.255"),
			Entry("IPv4 using commas", "203,0,113,255"),
		)
		It("should require AUTO to be in capital letters", func() {
			m := defaultMachine()
			m.Spec.FailoverIP = ptr.To("Auto")
			Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
		})
		It("should be immutable", func() {
			m := defaultMachine()
			m.Spec.FailoverIP = ptr.To(CloudResourceConfigAuto)
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.FailoverIP).To(Equal(ptr.To(CloudResourceConfigAuto)))
			m.Spec.FailoverIP = ptr.To("127.0.0.1")
			Expect(k8sClient.Update(context.Background(), m)).ToNot(Succeed())
			m.Spec.FailoverIP = ptr.To("")
			Expect(k8sClient.Update(context.Background(), m)).ToNot(Succeed())
		})
	})
	Context("NetworkID", func() {
		It("should allow setting an existing NetworkID in the spec", func() {
			m := defaultMachine()
			m.Spec.NetworkID = ptr.To("1")
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.NetworkID).To(Equal(ptr.To("1")))
		})
		It("should allow setting null", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.NetworkID).To(BeNil())
		})
		It("should not allow setting empty NetworkID", func() {
			m := defaultMachine()
			m.Spec.NetworkID = ptr.To("")
			Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
		})
		It("should be immutable", func() {
			m := defaultMachine()
			m.Spec.NetworkID = ptr.To("1")
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.NetworkID).To(Equal(ptr.To("1")))
			m.Spec.NetworkID = ptr.To("2")
			Expect(k8sClient.Update(context.Background(), m)).ToNot(Succeed())
			m.Spec.NetworkID = ptr.To("")
			Expect(k8sClient.Update(context.Background(), m)).ToNot(Succeed())
			m.Spec.NetworkID = nil
			Expect(k8sClient.Update(context.Background(), m)).ToNot(Succeed())
		})
	})
	Context("ServerType", func() {
		It("should default to ENTERPRISE", func() {
			m := defaultMachine()
			// because Type is a string, setting the value as "" is the same as not setting anything
			m.Spec.Type = ""
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(m.Spec.Type).To(Equal(ServerTypeEnterprise))
		})
		It("should fail if not part of the enum", func() {
			m := defaultMachine()
			m.Spec.Type = "this-should-fail"
			Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
		})
		It("should fail if cpuFamily is set and type is VCPU", func() {
			m := defaultMachine()
			m.Spec.CPUFamily = ptr.To("some-cpu-family")
			m.Spec.Type = ServerTypeVCPU
			Expect(k8sClient.Create(context.Background(), m)).ToNot(Succeed())
		})
		DescribeTable("should work for value",
			func(serverType ServerType) {
				m := defaultMachine()
				m.Spec.Type = serverType
				m.Spec.CPUFamily = nil
				Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
				Expect(m.Spec.Type).To(Equal(serverType))
			},
			Entry("ENTERPRISE", ServerTypeEnterprise),
			Entry("VCPU", ServerTypeVCPU),
		)
	})
	Context("Conditions", func() {
		It("should correctly set and get the conditions", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(
				context.Background(), client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

			// Calls SetConditions with required fields
			conditions.MarkTrue(m, MachineProvisionedCondition)

			Expect(k8sClient.Status().Update(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(context.Background(),
				client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

			machineConditions := m.GetConditions()
			Expect(machineConditions).To(HaveLen(1))
			Expect(machineConditions[0].Type).To(Equal(MachineProvisionedCondition))
			Expect(machineConditions[0].Status).To(Equal(corev1.ConditionTrue))
		})
	})
	Context("Status", func() {
		It("should correctly set and get the status", func() {
			m := defaultMachine()
			Expect(k8sClient.Create(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(context.Background(),
				client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

			m.Status.Ready = true
			conditions.MarkTrue(m, MachineProvisionedCondition)
			m.Status.CurrentRequest = &ProvisioningRequest{
				Method:      "GET",
				RequestPath: "path/to/resource",
				State:       sdk.RequestStatusRunning,
			}
			m.Status.FailureReason = ptr.To(errors.InvalidConfigurationMachineError)
			m.Status.FailureMessage = ptr.To("Failure message")
			m.Status.Location = "de/fra"

			m.Status.MachineNetworkInfo = &MachineNetworkInfo{
				NICInfo: []NICInfo{
					{
						IPv4Addresses: []string{"198.51.100.10"},
						IPv6Addresses: []string{"2001:db8:2c3:30a0::1", "2001:db8:2c3:30a0::2"},
						NetworkID:     10,
						Primary:       false,
					},
				},
			}

			want := *m.DeepCopy()

			Expect(k8sClient.Status().Update(context.Background(), m)).To(Succeed())
			Expect(k8sClient.Get(context.Background(),
				client.ObjectKey{Name: m.Name, Namespace: m.Namespace}, m)).To(Succeed())

			// Gomega matcher seems to have issues with comparing the dates.
			diff := cmp.Diff(want.Status, m.Status)
			Expect(diff).To(BeEmpty(), "m.Status differs from want.Status (-want +got):\n%s", diff)
		})
	})
})
