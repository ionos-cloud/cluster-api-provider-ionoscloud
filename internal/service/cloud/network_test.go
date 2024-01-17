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

package cloud

import (
	"fmt"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/utils/ptr"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
)

const (
	reqPath = "this/is/a/path"
	lanID   = "1"
)

func createLANCall() *clienttest.MockClient_CreateLAN_Call {
	return ionosClient.EXPECT().CreateLAN(ctx, service.dataCenterID(), sdk.LanPropertiesPost{
		Name:   ptr.To(service.lanName()),
		Public: ptr.To(true),
	})
}

func deleteLANCall(id string) *clienttest.MockClient_DeleteLAN_Call {
	return ionosClient.EXPECT().DeleteLAN(ctx, service.dataCenterID(), id)
}

func listLANsCall() *clienttest.MockClient_ListLANs_Call {
	return ionosClient.EXPECT().ListLANs(ctx, service.dataCenterID())
}

func getRequestsCall() *clienttest.MockClient_GetRequests_Call {
	return ionosClient.EXPECT().GetRequests(ctx, mock.Anything,
		path.Join("datacenters", service.dataCenterID(), "lans"))
}

func examplePostRequest(status string) []sdk.Request {
	body := fmt.Sprintf(`{"properties": {"name": "%s"}}`, service.lanName())
	return []sdk.Request{
		{
			Id: ptr.To("1"),
			Metadata: &sdk.RequestMetadata{
				RequestStatus: &sdk.RequestStatus{
					Metadata: &sdk.RequestStatusMetadata{
						Status:  ptr.To(status),
						Message: ptr.To("test"),
					},
				},
			},
			Properties: &sdk.RequestProperties{
				Method: ptr.To(http.MethodPost),
				Body:   ptr.To(body),
			},
		},
	}
}

var _ = Describe("Network tests", func() {

	Context("Helper functions", func() {
		It("can return the LAN name", func() {
			Expect(service.lanName()).To(Equal("k8s-default-test-cluster"))
		})
	})

	Context("Chatting with the API", func() {
		When("creating a LAN", func() {
			It("should update the infra cluster current request, when successful", func() {
				createLANCall().Return(reqPath, nil).Once()
				Expect(service.createLAN()).To(Succeed())
				Expect(service.dataCenterID()).To(BeKeyOf(infraCluster.Status.CurrentRequest))
				req := infraCluster.Status.CurrentRequest[service.dataCenterID()]
				Expect(req.RequestPath).To(Equal(reqPath), "Request path is different than expected")
				Expect(req.Method).To(Equal(http.MethodPost), "Request method is different than expected")
				Expect(req.State).To(Equal(infrav1.RequestStatusQueued), "Request state is different than expected")
			})

			It("should return an error, when the API call fails", func() {
				createLANCall().Return("", errMock).Once()
				err := service.createLAN()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
			})
		})

		When("deleting a LAN", func() {
			It("should update the infra cluster current request, when successful", func() {
				deleteLANCall(lanID).Return(reqPath, nil).Once()
				Expect(service.deleteLAN(lanID)).To(Succeed())
				Expect(service.dataCenterID()).To(BeKeyOf(infraCluster.Status.CurrentRequest))
				req := infraCluster.Status.CurrentRequest[service.dataCenterID()]
				Expect(req.RequestPath).To(Equal(reqPath), "Request path is different than expected")
				Expect(req.Method).To(Equal(http.MethodDelete), "Request method is different than expected")
				Expect(req.State).To(Equal(infrav1.RequestStatusQueued), "Request state is different than expected")
			})

			It("should return an error, when the API call fails", func() {
				deleteLANCall(lanID).Return("", errMock).Once()
				err := service.deleteLAN(lanID)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
			})
		})

		When("getting a LAN", func() {
			var lan sdk.Lan
			var lans *sdk.Lans

			BeforeEach(func() {
				lan = sdk.Lan{
					Id: ptr.To(lanID),
					Properties: &sdk.LanProperties{
						Name: ptr.To(service.lanName()),
					},
				}
				lans = &sdk.Lans{
					Items: &[]sdk.Lan{
						lan,
					},
				}
			})

			It("should return the LAN, when successful", func() {
				listLANsCall().Return(lans, nil).Once()
				foundLAN, err := service.GetLAN()
				Expect(err).ToNot(HaveOccurred())
				Expect(foundLAN).To(Equal(&lan))
			})

			It("should return an error, when the API call fails", func() {
				listLANsCall().Return(nil, errMock).Once()
				_, err := service.GetLAN()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
			})

			It("should not return an error, when the LAN is not found", func() {
				emptyLANs := &sdk.Lans{
					Items: &[]sdk.Lan{},
				}
				listLANsCall().Return(emptyLANs, nil).Once()
				foundLAN, err := service.GetLAN()
				Expect(err).ToNot(HaveOccurred())
				Expect(foundLAN).To(BeNil())
			})

			It("should return an error, when the LAN is not unique", func() {
				*lans.Items = append(*lans.Items, lan)
				listLANsCall().Return(lans, nil).Once()
				foundLAN, err := service.GetLAN()
				Expect(err).To(HaveOccurred())
				Expect(foundLAN).To(BeNil())
			})
		})

		When("checking for pending LAN requests", func() {
			It("should return an early error, when the method is not supported", func() {
				_, err := service.checkForPendingLANRequest(http.MethodTrace, lanID)
				Expect(err).To(HaveOccurred())
			})

			It("should return an early error, when trying to delete and the LAN ID is empty", func() {
				_, err := service.checkForPendingLANRequest(http.MethodDelete, "")
				Expect(err).To(HaveOccurred())
			})

			It("should return an early error, when trying to create and the LAN ID is not empty", func() {
				_, err := service.checkForPendingLANRequest(http.MethodPost, lanID)
				Expect(err).To(HaveOccurred())
			})

			It("should return an error, when the API call fails", func() {
				getRequestsCall().Return(nil, errMock).Once()
				_, err := service.checkForPendingLANRequest(http.MethodDelete, lanID)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
			})

			It("should return an empty status, when there are no pending requests", func() {
				var emptyRequests []sdk.Request
				getRequestsCall().Return(emptyRequests, nil).Once()
				status, err := service.checkForPendingLANRequest(http.MethodDelete, lanID)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(BeEmpty())
			})

			It("should return the status, when there is a matching DELETE request", func() {
				requests := []sdk.Request{
					{
						Id: ptr.To("1"),
						Metadata: &sdk.RequestMetadata{
							RequestStatus: &sdk.RequestStatus{
								Metadata: &sdk.RequestStatusMetadata{
									Targets: &[]sdk.RequestTarget{
										{
											Target: &sdk.ResourceReference{
												Id: ptr.To(lanID),
											},
										},
									},
									Status:  ptr.To(sdk.RequestStatusQueued),
									Message: ptr.To("test"),
								},
							},
						},
					},
				}
				getRequestsCall().Return(requests, nil).Once()
				status, err := service.checkForPendingLANRequest(http.MethodDelete, lanID)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(sdk.RequestStatusQueued))
			})

			It("should return the status, when there is a matching POST request", func() {
				reqStatus := sdk.RequestStatusQueued
				requests := examplePostRequest(reqStatus)
				getRequestsCall().Return(requests, nil).Once()
				status, err := service.checkForPendingLANRequest(http.MethodPost, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(reqStatus))
			})

			It("should return an empty status, when there is a POST request but the name is different", func() {
				requests := []sdk.Request{
					{
						Id: ptr.To("1"),
						Metadata: &sdk.RequestMetadata{
							RequestStatus: &sdk.RequestStatus{
								Metadata: &sdk.RequestStatusMetadata{
									Status:  ptr.To(sdk.RequestStatusQueued),
									Message: ptr.To("test"),
								},
							},
						},
						Properties: &sdk.RequestProperties{
							Method: ptr.To(http.MethodPost),
							Body:   ptr.To(`{"properties": {"name": "different"}}`),
						},
					},
				}
				getRequestsCall().Return(requests, nil).Once()
				status, err := service.checkForPendingLANRequest(http.MethodPost, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(BeEmpty())
			})

			It("should return an empty status, when there is a DELETE request but the ID is different", func() {
				requests := []sdk.Request{
					{
						Id: ptr.To("1"),
						Metadata: &sdk.RequestMetadata{
							RequestStatus: &sdk.RequestStatus{
								Metadata: &sdk.RequestStatusMetadata{
									Targets: &[]sdk.RequestTarget{
										{
											Target: &sdk.ResourceReference{
												Id: ptr.To("different"),
											},
										},
									},
									Status:  ptr.To(sdk.RequestStatusQueued),
									Message: ptr.To("test"),
								},
							},
						},
					},
				}
				getRequestsCall().Return(requests, nil).Once()
				status, err := service.checkForPendingLANRequest(http.MethodDelete, lanID)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(BeEmpty())
			})
		})

		When("removing a pending LAN request from the cluster", func() {
			It("should remove the request, when it exists", func() {
				infraCluster.Status.CurrentRequest = make(map[string]infrav1.ProvisioningRequest)
				infraCluster.Status.CurrentRequest[service.dataCenterID()] = infrav1.ProvisioningRequest{
					Method:      http.MethodDelete,
					RequestPath: "test",
					State:       infrav1.RequestStatusQueued,
				}
				Expect(service.removeLANPendingRequestFromCluster()).To(Succeed())
				Expect(infraCluster.Status.CurrentRequest).ToNot(HaveKey(service.dataCenterID()))
			})

			It("should not return an error, when the request does not exist", func() {
				Expect(service.removeLANPendingRequestFromCluster()).To(Succeed())
			})
		})
	})

	Context("reconciling the cluster LAN,", func() {
		When("the LAN does not exist", func() {
			It("should request the creation of the LAN, when there is no pending request", func() {
				listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
				getRequestsCall().Return([]sdk.Request{}, nil).Once()
				createLANCall().Return("requestPath", nil).Once()
				requeue, err := service.ReconcileLAN()
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeTrue())
				Expect(service.dataCenterID()).To(BeKeyOf(infraCluster.Status.CurrentRequest))
				req := infraCluster.Status.CurrentRequest[service.dataCenterID()]
				Expect(req.Method).To(Equal(http.MethodPost), "Request method is different than expected")
				Expect(req.State).To(Equal(infrav1.RequestStatusQueued), "Request state is different than expected")
			})

			When("there is a pending request", func() {
				DescribeTable("should not request the creation of the LAN",
					func(status string) {
						listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
						getRequestsCall().Return(examplePostRequest(status), nil).Once()
						requeue, err := service.ReconcileLAN()
						Expect(err).ToNot(HaveOccurred())
						Expect(requeue).To(BeTrue())
					},
					Entry("when the request is queued", sdk.RequestStatusQueued),
					Entry("when the request is running", sdk.RequestStatusRunning),
				)

				It("should request the creation of the LAN, when the request had failed", func() {
					listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
					getRequestsCall().Return(examplePostRequest(sdk.RequestStatusFailed), nil)
					createLANCall().Return("requestPath", nil).Once()
					requeue, err := service.ReconcileLAN()
					Expect(err).ToNot(HaveOccurred())
					Expect(requeue).To(BeTrue())
				})

				It("should retry to get the LAN, when the request has succeeded while reconciling", func() {
					listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
					getRequestsCall().Return(examplePostRequest(sdk.RequestStatusDone), nil)
					lan := sdk.Lan{
						Id: ptr.To("1"),
						Properties: &sdk.LanProperties{
							Name: ptr.To(service.lanName()),
						},
					}
					listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
					requeue, err := service.ReconcileLAN()
					Expect(err).ToNot(HaveOccurred())
					Expect(requeue).To(BeFalse())
				})
			})
		})

		It("should not request the creation of the LAN, if it already exists", func() {
			lan := sdk.Lan{
				Id: ptr.To("1"),
				Properties: &sdk.LanProperties{
					Name: ptr.To(service.lanName()),
				},
			}
			listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
			requeue, err := service.ReconcileLAN()
			Expect(err).ToNot(HaveOccurred())
			Expect(requeue).To(BeFalse())
		})

		Context("means an error should be returned and no requeue if the API call fails", func() {
			Specify("when listing the LANs", func() {
				listLANsCall().Return(nil, errMock).Once()
				requeue, err := service.ReconcileLAN()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})

			Specify("when getting the requests", func() {
				listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
				getRequestsCall().Return(nil, errMock).Once()
				requeue, err := service.ReconcileLAN()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})

			Specify("when creating the LAN", func() {
				listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
				getRequestsCall().Return([]sdk.Request{}, nil).Once()
				createLANCall().Return("", errMock).Once()
				requeue, err := service.ReconcileLAN()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})

			Specify("when listing the LANs again if the request has succeeded", func() {
				listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
				getRequestsCall().Return(examplePostRequest(sdk.RequestStatusDone), nil)
				listLANsCall().Return(nil, errMock).Once()
				requeue, err := service.ReconcileLAN()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})
		})
	})

	Context("reconciling the cluster LAN deletion,", func() {
		When("the LAN exists", func() {
			var lan sdk.Lan
			BeforeEach(func() {
				lan = sdk.Lan{
					Id: ptr.To("1"),
					Properties: &sdk.LanProperties{
						Name: ptr.To(service.lanName()),
					},
					Entities: &sdk.LanEntities{
						Nics: &sdk.LanNics{
							Items: &[]sdk.Nic{},
						},
					},
				}
				listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{lan}}, nil).Once()
			})

			It("should request the deletion of the LAN, when there is no pending request, and the LAN isn't being used by other resource", func() {
				getRequestsCall().Return([]sdk.Request{}, nil).Once()
				deleteLANCall(lanID).Return(reqPath, nil).Once()
				requeue, err := service.ReconcileLANDeletion()
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeTrue())
				Expect(service.dataCenterID()).To(BeKeyOf(infraCluster.Status.CurrentRequest))
				req := infraCluster.Status.CurrentRequest[service.dataCenterID()]
				Expect(req.Method).To(Equal(http.MethodDelete), "Request method is different than expected")
				Expect(req.State).To(Equal(infrav1.RequestStatusQueued), "Request state is different than expected")
			})

			It("should not request the deletion of the LAN, when there is no pending request, but the LAN is being used by another resource", func() {
				lan.Entities.Nics.Items = &[]sdk.Nic{
					{
						Id: ptr.To("1"),
					},
				}
				getRequestsCall().Return([]sdk.Request{}, nil).Once()
				requeue, err := service.ReconcileLANDeletion()
				Expect(err).ToNot(HaveOccurred())
				Expect(requeue).To(BeFalse())
			})

			When("there is a pending request", func() {
				DescribeTable("should not request the deletion of the LAN, but still requeue",
					func(status string) {
						getRequestsCall().Return([]sdk.Request{
							{
								Id: ptr.To("1"),
								Metadata: &sdk.RequestMetadata{
									RequestStatus: &sdk.RequestStatus{
										Metadata: &sdk.RequestStatusMetadata{
											Status:  ptr.To(status),
											Message: ptr.To("test"),
											Targets: &[]sdk.RequestTarget{
												{
													Target: &sdk.ResourceReference{Id: ptr.To(lanID)},
												},
											},
										},
									},
								},
							},
						}, nil).Once()
						requeue, err := service.ReconcileLANDeletion()
						Expect(err).ToNot(HaveOccurred())
						Expect(requeue).To(BeTrue())
					},
					Entry("when the request is queued", sdk.RequestStatusQueued),
					Entry("when the request is running", sdk.RequestStatusRunning),
				)

				It("should check if the LAN is indeed gone, if the request succeeded", func() {
					getRequestsCall().Return([]sdk.Request{
						{
							Id: ptr.To("1"),
							Metadata: &sdk.RequestMetadata{
								RequestStatus: &sdk.RequestStatus{
									Metadata: &sdk.RequestStatusMetadata{
										Status:  ptr.To(sdk.RequestStatusDone),
										Message: ptr.To("test"),
										Targets: &[]sdk.RequestTarget{
											{
												Target: &sdk.ResourceReference{Id: ptr.To(lanID)},
											},
										},
									},
								},
							},
						},
					}, nil)
					listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
					requeue, err := service.ReconcileLANDeletion()
					Expect(infraCluster.Status.CurrentRequest).ToNot(HaveKey(service.dataCenterID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(requeue).To(BeFalse())
				})
			})
		})

		It("should not request the deletion of the LAN, when the LAN does not exist", func() {
			listLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{}}, nil).Once()
			requeue, err := service.ReconcileLANDeletion()
			Expect(err).ToNot(HaveOccurred())
			Expect(requeue).To(BeFalse())
			Expect(infraCluster.Status.CurrentRequest).ToNot(HaveKey(service.dataCenterID()))
		})

		Context("means an error should be returned and no requeue if the API call fails", func() {
			var lans *[]sdk.Lan

			BeforeEach(func() {
				lans = &[]sdk.Lan{{
					Id: ptr.To("1"),
					Properties: &sdk.LanProperties{
						Name: ptr.To(service.lanName()),
					},
					Entities: &sdk.LanEntities{
						Nics: &sdk.LanNics{
							Items: &[]sdk.Nic{},
						},
					},
				}}
			})

			Specify("when listing the LANs", func() {
				listLANsCall().Return(nil, errMock).Once()
				requeue, err := service.ReconcileLANDeletion()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})

			Specify("when getting the requests", func() {
				listLANsCall().Return(&sdk.Lans{Items: lans}, nil).Once()
				getRequestsCall().Return(nil, errMock).Once()
				requeue, err := service.ReconcileLANDeletion()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})

			Specify("when deleting the LAN", func() {
				listLANsCall().Return(&sdk.Lans{Items: lans}, nil).Once()
				getRequestsCall().Return([]sdk.Request{}, nil).Once()
				deleteLANCall(lanID).Return("", errMock).Once()
				requeue, err := service.ReconcileLANDeletion()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})

			Specify("when listing the LANs again if the request has succeeded", func() {
				listLANsCall().Return(&sdk.Lans{Items: lans}, nil).Once()
				getRequestsCall().Return([]sdk.Request{
					{
						Id: ptr.To("1"),
						Metadata: &sdk.RequestMetadata{
							RequestStatus: &sdk.RequestStatus{
								Metadata: &sdk.RequestStatusMetadata{
									Status:  ptr.To(sdk.RequestStatusDone),
									Message: ptr.To("test"),
									Targets: &[]sdk.RequestTarget{
										{Target: &sdk.ResourceReference{Id: ptr.To(lanID)}},
									},
								},
							},
						},
						Properties: &sdk.RequestProperties{
							Method: ptr.To(http.MethodDelete),
							Body:   ptr.To(`{"properties": {"name": "k8s-default-test-cluster"}}`),
						},
					},
				}, nil)
				listLANsCall().Return(nil, errMock).Once()
				requeue, err := service.ReconcileLANDeletion()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errMock))
				Expect(requeue).To(BeFalse())
			})
		})
	})
})
