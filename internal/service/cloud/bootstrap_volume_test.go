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
	"encoding/base64"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
)

const (
	exampleBootstrapVolumeID      = "4a1f6c8e-2b7d-4e3a-9c5f-8d2e1b0a7f63"
	exampleBootstrapVolumeImageID = "7c9e2d4b-1a3f-4b5e-8d6c-0f2a9e7b3c51"
	exampleIgnitionConfig         = `{"ignition":{"version":"3.4.0"}}`
)

func (s *serverSuite) TestBootstrapVolumeName() {
	s.Equal("cfg-"+string(s.infraMachine.UID), s.service.bootstrapVolumeName(s.infraMachine))
}

func (s *serverSuite) TestReconcileServerIgnitionCreatesBootstrapVolume() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.prepareServerCreation("ignition", exampleIgnitionConfig)
	s.mockGetImageCall(exampleBootstrapVolumeImageID).
		Return(makeTestImageWithSize(exampleBootstrapVolumeImageID, "cfg", exampleLocation, 1), nil).Once()

	props, entities := s.ignitionServerComponents()
	s.ionosClient.EXPECT().CreateServer(s.ctx, s.machineScope.DatacenterID(), props, entities).
		Return(&sdk.Server{Id: new(exampleServerID)}, "location/to/server", nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
	s.Equal("ionos://"+exampleServerID, s.infraMachine.Spec.ProviderID)
	s.Equal(http.MethodPost, s.infraMachine.Status.CurrentRequest.Method)
}

func (s *serverSuite) TestReconcileServerIgnitionBootstrapVolumeSelectorIgnoresMachineVersion() {
	s.setBootstrapVolume(infrav1.ImageSpec{Selector: &infrav1.ImageSelector{
		MatchLabels:       map[string]string{"flatcar": "config"},
		ResolutionPolicy:  infrav1.ResolutionPolicyExact,
		UseMachineVersion: new(true),
	}})
	s.infraMachine.Spec.BootstrapVolume.DiskType = infrav1.VolumeDiskTypeSSDStandard
	s.prepareServerCreation("ignition", exampleIgnitionConfig)
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return([]sdk.Label{
		makeTestLabel("image", exampleBootstrapVolumeImageID, "flatcar", "config"),
	}, nil).Once()
	s.mockGetDatacenterLocationByIDCall(s.infraMachine.Spec.DatacenterID).Return(exampleLocation, nil).Once()
	s.mockGetImageCall(exampleBootstrapVolumeImageID).
		Return(makeTestImageWithSize(exampleBootstrapVolumeImageID, "flatcar-config", exampleLocation, 1), nil).Once()

	props, entities := s.ignitionServerComponents()
	(*entities.Volumes.Items)[1].Properties.Type = new(infrav1.VolumeDiskTypeSSDStandard.String())
	s.ionosClient.EXPECT().CreateServer(s.ctx, s.machineScope.DatacenterID(), props, entities).
		Return(&sdk.Server{Id: new(exampleServerID)}, "location/to/server", nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerIgnitionBootstrapVolumeImageNotFound() {
	tests := []struct {
		name   string
		labels []sdk.Label
		images []*sdk.Image
	}{{
		name: "no match",
	}, {
		name: "multiple matches",
		labels: []sdk.Label{
			makeTestLabel("image", "image-1", "flatcar", "config"),
			makeTestLabel("image", "image-2", "flatcar", "config"),
		},
		images: []*sdk.Image{
			makeTestImageWithSize("image-1", "cfg-1", exampleLocation, 1),
			makeTestImageWithSize("image-2", "cfg-2", exampleLocation, 1),
		},
	}}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			s.setBootstrapVolume(infrav1.ImageSpec{Selector: &infrav1.ImageSelector{
				MatchLabels:      map[string]string{"flatcar": "config"},
				ResolutionPolicy: infrav1.ResolutionPolicyExact,
			}})
			s.prepareServerCreation("ignition", exampleIgnitionConfig)
			s.ionosClient.EXPECT().ListLabels(s.ctx).Return(tt.labels, nil).Once()
			s.mockGetDatacenterLocationByIDCall(s.infraMachine.Spec.DatacenterID).Return(exampleLocation, nil).Once()
			for _, image := range tt.images {
				s.mockGetImageCall(*image.Id).Return(image, nil).Once()
			}

			requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
			s.ErrorAs(err, new(imageMatchError))
			s.ErrorContains(err, "bootstrap volume image lookup")
			s.False(requeue)
			s.Empty(s.infraMachine.Spec.ProviderID)
		})
	}
}

func (s *serverSuite) TestReconcileServerIgnitionBootstrapVolumeImageWithoutSize() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.prepareServerCreation("ignition", exampleIgnitionConfig)
	s.mockGetImageCall(exampleBootstrapVolumeImageID).
		Return(makeTestImage(exampleBootstrapVolumeImageID, "cfg", exampleLocation), nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.ErrorContains(err, "has no size")
	s.False(requeue)
}

func (s *serverSuite) TestReconcileServerIgnitionWithoutBootstrapVolume() {
	s.prepareServerCreation("ignition", exampleIgnitionConfig)

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.ErrorIs(err, ErrBootstrapVolumeRequired)
	s.False(requeue)
	s.Empty(s.infraMachine.Spec.ProviderID)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileServerCloudConfigIgnoresBootstrapVolume() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.prepareServerCreation("cloud-config", "test")

	props, entities := s.defaultServerComponents()
	(*entities.Volumes.Items)[0].Properties.UserData = new(base64.StdEncoding.EncodeToString([]byte(
		"test\nbootcmd:\n  - echo test-machine > /etc/hostname\n  - hostname test-machine\n")))
	s.ionosClient.EXPECT().CreateServer(s.ctx, s.machineScope.DatacenterID(), props, entities).
		Return(&sdk.Server{Id: new(exampleServerID)}, "location/to/server", nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
}

func (s *serverSuite) TestReconcileServerKeepsBootstrapVolumeAfterNodeRef() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.prepareRunningServer()
	s.setNodeRef()
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(sdk.Available)), nil).Once()

	requeue, err := s.service.ReconcileServer(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeWithoutNodeRef() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeWithoutBootstrapVolumeSpec() {
	s.setNodeRef()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeRemovesVolumeAfterNodeRef() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.setNodeRef()

	// Nothing is kept in memory between create and cleanup, so this also covers a controller restart.
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(sdk.Available)), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()
	s.mockDeleteVolumeCall(exampleBootstrapVolumeID).Return("delete/volume", nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(requeue, err, "delete/volume")

	s.infraMachine.DeleteCurrentRequest()
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(), nil).Once()

	requeue, err = s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeRequestPending() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.setNodeRef()
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(sdk.Busy)), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).
		Return([]sdk.Request{s.exampleVolumeDeleteRequest(sdk.RequestStatusQueued)}, nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(requeue, err, path.Join(exampleRequestPath, exampleBootstrapVolumeID))
	s.Equal(sdk.RequestStatusQueued, s.infraMachine.Status.CurrentRequest.State)
}

func (s *serverSuite) TestReconcileBootstrapVolumeRequestFailed() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.setNodeRef()
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(sdk.Available)), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).
		Return([]sdk.Request{s.exampleVolumeDeleteRequest(sdk.RequestStatusFailed)}, nil).Once()
	s.mockDeleteVolumeCall(exampleBootstrapVolumeID).Return("delete/volume", nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(requeue, err, "delete/volume")
	s.Equal(sdk.RequestStatusQueued, s.infraMachine.Status.CurrentRequest.State)
}

func (s *serverSuite) TestReconcileBootstrapVolumeBusy() {
	for _, state := range []string{sdk.Busy, sdk.Destroying} {
		s.Run(state, func() {
			s.SetupTest()
			s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
			s.setNodeRef()
			s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(state)), nil).Once()
			s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()

			requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
			s.NoError(err)
			s.True(requeue)
			s.Nil(s.infraMachine.Status.CurrentRequest)
		})
	}
}

func (s *serverSuite) TestReconcileBootstrapVolumeAlreadyGone() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.setNodeRef()
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(sdk.Available)), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()
	s.mockDeleteVolumeCall(exampleBootstrapVolumeID).
		Return("", sdk.NewGenericOpenAPIError("not found", nil, nil, http.StatusNotFound)).Once()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeleteError() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.setNodeRef()
	s.mockGetServerCall(exampleServerID).Return(s.runningServer(s.bootstrapVolume(sdk.Available)), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()
	s.mockDeleteVolumeCall(exampleBootstrapVolumeID).
		Return("", sdk.NewGenericOpenAPIError("error", nil, nil, http.StatusInternalServerError)).Once()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.ErrorContains(err, "failed to request bootstrap volume deletion")
	s.True(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeServerNotFound() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.setNodeRef()
	s.mockGetServerCall(exampleServerID).
		Return(nil, sdk.NewGenericOpenAPIError("not found", nil, nil, http.StatusNotFound)).Once()
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolume(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileServerDeletionIgnoresAttachedBootstrapVolume() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})

	server := s.runningServer(s.bootstrapVolume(sdk.Available))
	server.Properties.BootVolume = &sdk.ResourceReference{Id: new(exampleBootVolumeID)}
	s.mockGetServerCall(exampleServerID).Return(server, nil).Once()

	serverWithoutBootVolume := s.runningServer(s.bootstrapVolume(sdk.Available))
	s.mockGetServerCall(exampleServerID).Return(serverWithoutBootVolume, nil).Once()

	s.mockGetServerDeletionRequestCall(exampleServerID).Return(nil, nil).Times(2)
	s.mockDeleteVolumeCall(exampleBootVolumeID).Return("delete/boot", nil).Once()
	s.mockDeleteServerCall(exampleServerID, false).Return("delete/server", nil).Once()

	for _, location := range []string{"delete/boot", "delete/server"} {
		requeue, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
		s.validateSuccessfulDeletionResponse(requeue, err, location)
	}
}

func (s *serverSuite) TestReconcileServerDeletionClusterDeletedWithAttachedBootstrapVolume() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.clusterScope.Cluster.DeletionTimestamp = new(metav1.Now())

	server := s.runningServer(s.bootstrapVolume(sdk.Available))
	server.Properties.BootVolume = &sdk.ResourceReference{Id: new(exampleBootVolumeID)}
	s.mockGetServerCall(exampleServerID).Return(server, nil).Once()
	s.mockGetServerDeletionRequestCall(exampleServerID).Return(nil, nil).Once()
	s.mockDeleteServerCall(exampleServerID, true).Return("delete/server", nil).Once()

	requeue, err := s.service.ReconcileServerDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(requeue, err, "delete/server")
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionWithoutBootstrapVolumeSpec() {
	requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionDetachedVolume() {
	tests := []struct {
		name           string
		clusterDeleted bool
	}{
		{name: "machine deletion"},
		{name: "cluster deletion", clusterDeleted: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
			if tt.clusterDeleted {
				s.clusterScope.Cluster.DeletionTimestamp = new(metav1.Now())
			}

			s.mockListVolumesCall().Return(s.datacenterVolumes(), nil).Once()
			s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()
			s.mockDeleteVolumeCall(exampleBootstrapVolumeID).Return("delete/bootstrap", nil).Once()

			requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
			s.validateSuccessfulDeletionResponse(requeue, err, "delete/bootstrap")
		})
	}
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionRequestPending() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.mockListVolumesCall().Return(s.datacenterVolumes(), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).
		Return([]sdk.Request{s.exampleVolumeDeleteRequest(sdk.RequestStatusRunning)}, nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(requeue, err, path.Join(exampleRequestPath, exampleBootstrapVolumeID))
	s.Equal(sdk.RequestStatusRunning, s.infraMachine.Status.CurrentRequest.State)
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionRequestFailed() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.mockListVolumesCall().Return(s.datacenterVolumes(), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).
		Return([]sdk.Request{s.exampleVolumeDeleteRequest(sdk.RequestStatusFailed)}, nil).Once()
	s.mockDeleteVolumeCall(exampleBootstrapVolumeID).Return("delete/bootstrap", nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
	s.validateSuccessfulDeletionResponse(requeue, err, "delete/bootstrap")
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionVolumeAlreadyGone() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.mockListVolumesCall().Return(s.datacenterVolumes(), nil).Once()
	s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()
	s.mockDeleteVolumeCall(exampleBootstrapVolumeID).
		Return("", sdk.NewGenericOpenAPIError("not found", nil, nil, http.StatusNotFound)).Once()

	requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.True(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionVolumeBusy() {
	for _, state := range []string{sdk.Busy, sdk.Destroying} {
		s.Run(state, func() {
			s.SetupTest()
			s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
			volumes := s.datacenterVolumes()
			(*volumes.Items)[2] = s.bootstrapVolume(state)
			s.mockListVolumesCall().Return(volumes, nil).Once()
			s.mockGetVolumeDeletionRequestCall(exampleBootstrapVolumeID).Return(nil, nil).Once()

			requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
			s.NoError(err)
			s.True(requeue)
			s.Nil(s.infraMachine.Status.CurrentRequest)
		})
	}
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionNoVolume() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	volumes := s.datacenterVolumes()
	*volumes.Items = (*volumes.Items)[:2]
	s.mockListVolumesCall().Return(volumes, nil).Once()

	requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
	s.NoError(err)
	s.False(requeue)
	s.Nil(s.infraMachine.Status.CurrentRequest)
}

func (s *serverSuite) TestReconcileBootstrapVolumeDeletionListError() {
	s.setBootstrapVolume(infrav1.ImageSpec{ID: exampleBootstrapVolumeImageID})
	s.mockListVolumesCall().Return(nil, sdk.NewGenericOpenAPIError("error", nil, nil, http.StatusInternalServerError)).Once()

	requeue, err := s.service.ReconcileBootstrapVolumeDeletion(s.ctx, s.machineScope)
	s.Error(err)
	s.False(requeue)
}

func (s *serverSuite) setBootstrapVolume(image infrav1.ImageSpec) {
	s.infraMachine.Spec.BootstrapVolume = &infrav1.BootstrapVolume{
		Image:    &image,
		DiskType: infrav1.VolumeDiskTypeHDD,
	}
}

func (s *serverSuite) setNodeRef() {
	s.capiMachine.Status.NodeRef = clusterv1.MachineNodeReference{Name: s.capiMachine.Name}
}

func (s *serverSuite) createBootstrapSecret(data map[string][]byte) {
	s.T().Helper()
	s.NoError(s.k8sClient.Create(s.ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: metav1.NamespaceDefault,
		},
		Data: data,
	}))
	s.machineScope.Machine.Spec.Bootstrap.DataSecretName = new("test")
}

// prepareServerCreation prepares a machine without a server and expects the server creation up to the image lookup.
func (s *serverSuite) prepareServerCreation(format, value string) {
	s.T().Helper()
	s.createBootstrapSecret(map[string][]byte{
		"format": []byte(format),
		"value":  []byte(value),
	})
	s.infraMachine.Spec.ProviderID = ""
	s.mockListServersCall().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil).Once()
	s.mockGetServerCreationRequestCall().Return([]sdk.Request{}, nil).Once()
	if format == "ignition" && s.infraMachine.Spec.BootstrapVolume == nil {
		return
	}
	s.mockListLANsCall().Return(&sdk.Lans{Items: &[]sdk.Lan{s.exampleLAN()}}, nil).Once()
}

func (s *serverSuite) prepareRunningServer() {
	s.T().Helper()
	s.createBootstrapSecret(map[string][]byte{
		"format": []byte("ignition"),
		"value":  []byte(exampleIgnitionConfig),
	})
}

func (s *serverSuite) ignitionServerComponents() (sdk.ServerProperties, sdk.ServerEntities) {
	props, entities := s.defaultServerComponents()
	(*entities.Volumes.Items)[0].Properties.BootOrder = new("PRIMARY")
	*entities.Volumes.Items = append(*entities.Volumes.Items, sdk.Volume{
		Properties: &sdk.VolumeProperties{
			AvailabilityZone:    new("AUTO"),
			BootOrder:           new("NONE"),
			DiscVirtioHotUnplug: new(true),
			Image:               new(exampleBootstrapVolumeImageID),
			Name:                new("cfg-" + string(s.infraMachine.UID)),
			Size:                new(float32(1)),
			Type:                new("HDD"),
			UserData:            new(base64.StdEncoding.EncodeToString([]byte(exampleIgnitionConfig))),
		},
	})
	return props, entities
}

func (s *serverSuite) runningServer(volumes ...sdk.Volume) *sdk.Server {
	return &sdk.Server{
		Id: new(exampleServerID),
		Metadata: &sdk.DatacenterElementMetadata{
			State: new(sdk.Available),
		},
		Properties: &sdk.ServerProperties{
			Name:    new(s.infraMachine.Name),
			VmState: new("RUNNING"),
		},
		Entities: &sdk.ServerEntities{
			Volumes: &sdk.AttachedVolumes{
				Items: new(append([]sdk.Volume{{
					Id:         new(exampleBootVolumeID),
					Properties: &sdk.VolumeProperties{Name: new(s.service.volumeName(s.infraMachine))},
				}}, volumes...)),
			},
		},
	}
}

func (s *serverSuite) bootstrapVolume(state string) sdk.Volume {
	return sdk.Volume{
		Id:         new(exampleBootstrapVolumeID),
		Metadata:   &sdk.DatacenterElementMetadata{State: &state},
		Properties: &sdk.VolumeProperties{Name: new(s.service.bootstrapVolumeName(s.infraMachine))},
	}
}

// datacenterVolumes returns a list of volumes, where the bootstrap volume of the machine is the last one.
func (s *serverSuite) datacenterVolumes() *sdk.Volumes {
	return &sdk.Volumes{Items: &[]sdk.Volume{{
		Id:         new("other-machine-bootstrap-volume"),
		Properties: &sdk.VolumeProperties{Name: new("cfg-other-machine")},
	}, {
		Id:         new("csi-volume"),
		Properties: &sdk.VolumeProperties{Name: new("pvc-1234")},
	}, s.bootstrapVolume(sdk.Available)}}
}

func (s *serverSuite) mockGetImageCall(imageID string) *clienttest.MockClient_GetImage_Call {
	return s.ionosClient.EXPECT().GetImage(s.ctx, imageID)
}

func (s *serverSuite) mockListVolumesCall() *clienttest.MockClient_ListVolumes_Call {
	return s.ionosClient.EXPECT().ListVolumes(s.ctx, s.machineScope.DatacenterID())
}

func (s *serverSuite) mockGetVolumeDeletionRequestCall(volumeID string) *clienttest.MockClient_GetRequests_Call {
	return s.ionosClient.EXPECT().GetRequests(s.ctx, http.MethodDelete,
		path.Join("datacenters", s.machineScope.DatacenterID(), "volumes", volumeID))
}

func (s *serverSuite) exampleVolumeDeleteRequest(status string) sdk.Request {
	return s.exampleRequest(requestBuildOptions{
		status:     status,
		method:     http.MethodDelete,
		url:        path.Join("datacenters", s.machineScope.DatacenterID(), "volumes", exampleBootstrapVolumeID),
		href:       path.Join(exampleRequestPath, exampleBootstrapVolumeID),
		targetID:   exampleBootstrapVolumeID,
		targetType: sdk.VOLUME,
	})
}

func makeTestImageWithSize(id, name, location string, size float32) *sdk.Image {
	image := makeTestImage(id, name, location)
	image.Properties.Size = &size
	return image
}
