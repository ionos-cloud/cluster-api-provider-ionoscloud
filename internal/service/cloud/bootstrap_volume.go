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
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	bootOrderPrimary = "PRIMARY"
	bootOrderNone    = "NONE"
)

// ErrBootstrapVolumeRequired is returned if the bootstrap data is in Ignition format,
// but the IonosCloudMachine has no bootstrap volume configured.
var ErrBootstrapVolumeRequired = errors.New(
	"bootstrap data is in Ignition format, which requires spec.bootstrapVolume to be set")

// ReconcileBootstrapVolume deletes the bootstrap volume attached to the server once the Machine has a node reference,
// as the bootstrap data is no longer needed.
func (s *Service) ReconcileBootstrapVolume(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	if ms.IonosMachine.Spec.BootstrapVolume == nil || !ms.Machine.Status.NodeRef.IsDefined() {
		return false, nil
	}

	server, err := s.getServer(ctx, ms)
	if server == nil || err != nil {
		return false, ignoreNotFound(err)
	}

	volume := s.attachedBootstrapVolume(ms, server)
	if volume == nil {
		return false, nil
	}

	return true, s.removeBootstrapVolume(ctx, ms, volume)
}

// ReconcileBootstrapVolumeDeletion deletes bootstrap volumes of the machine that are not attached to a server.
// It must only run after the server has been deleted.
func (s *Service) ReconcileBootstrapVolumeDeletion(ctx context.Context, ms *scope.Machine) (requeue bool, err error) {
	if ms.IonosMachine.Spec.BootstrapVolume == nil {
		return false, nil
	}

	volumes, err := s.apiWithDepth(1).ListVolumes(ctx, ms.DatacenterID())
	if err != nil {
		return false, fmt.Errorf("failed to list volumes in data center %s: %w", ms.DatacenterID(), err)
	}

	name := s.bootstrapVolumeName(ms.IonosMachine)
	for _, volume := range ptr.Deref(volumes.GetItems(), []sdk.Volume{}) {
		if ptr.Deref(volume.GetProperties().GetName(), "") == name {
			return true, s.removeBootstrapVolume(ctx, ms, &volume)
		}
	}

	return false, nil
}

// removeBootstrapVolume requests the deletion of the bootstrap volume, unless its deletion is already in progress.
func (s *Service) removeBootstrapVolume(ctx context.Context, ms *scope.Machine, volume *sdk.Volume) error {
	log := s.logger.WithName("removeBootstrapVolume")

	volumeID := ptr.Deref(volume.GetId(), "")
	request, err := s.getLatestVolumeDeletionRequest(ctx, ms.DatacenterID(), volumeID)
	if err != nil {
		return err
	}

	if request != nil && request.isPending() {
		ms.IonosMachine.SetCurrentRequest(http.MethodDelete, request.status, request.location)
		log.Info("Bootstrap volume deletion request is pending", "location", request.location)
		return nil
	}

	if state := getState(volume); state == sdk.Busy || state == sdk.Destroying {
		log.V(4).Info("Bootstrap volume is busy", "volumeID", volumeID, "state", state)
		return nil
	}

	return s.deleteBootstrapVolume(ctx, ms, volumeID)
}

func (s *Service) attachedBootstrapVolume(ms *scope.Machine, server *sdk.Server) *sdk.Volume {
	if ms.IonosMachine.Spec.BootstrapVolume == nil {
		return nil
	}

	name := s.bootstrapVolumeName(ms.IonosMachine)
	for _, volume := range ptr.Deref(server.GetEntities().GetVolumes().GetItems(), []sdk.Volume{}) {
		if ptr.Deref(volume.GetProperties().GetName(), "") == name {
			return &volume
		}
	}

	return nil
}

func (s *Service) deleteBootstrapVolume(ctx context.Context, ms *scope.Machine, volumeID string) error {
	log := s.logger.WithName("deleteBootstrapVolume")

	requestLocation, err := s.ionosClient.DeleteVolume(ctx, ms.DatacenterID(), volumeID)
	if isNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to request bootstrap volume deletion: %w", err)
	}

	ms.IonosMachine.SetCurrentRequest(http.MethodDelete, sdk.RequestStatusQueued, requestLocation)
	log.Info("Successfully requested for bootstrap volume deletion", "volumeID", volumeID, "location", requestLocation)
	return nil
}

func (s *Service) getLatestVolumeDeletionRequest(
	ctx context.Context, datacenterID, volumeID string,
) (*requestInfo, error) {
	return getMatchingRequest[sdk.Volume](
		ctx,
		s,
		http.MethodDelete,
		path.Join("datacenters", datacenterID, "volumes", volumeID),
	)
}

// buildBootstrapVolume returns the volume that carries the Ignition config. IONOS Cloud writes the
// userData to this volume, from where the OS reads it on first boot.
func (s *Service) buildBootstrapVolume(
	ctx context.Context, ms *scope.Machine, bootstrapData []byte,
) (*sdk.Volume, error) {
	image, err := s.lookupBootstrapVolumeImage(ctx, ms)
	if err != nil {
		return nil, fmt.Errorf("bootstrap volume image lookup: %w", err)
	}

	imageID := ptr.Deref(image.GetId(), "")
	size := ptr.Deref(image.GetProperties().GetSize(), 0)
	if size <= 0 {
		return nil, fmt.Errorf("bootstrap volume image %s has no size", imageID)
	}

	diskType := cmp.Or(ms.IonosMachine.Spec.BootstrapVolume.DiskType, infrav1.VolumeDiskTypeHDD)

	return &sdk.Volume{
		Properties: &sdk.VolumeProperties{
			AvailabilityZone:    new(infrav1.AvailabilityZoneAuto.String()),
			BootOrder:           new(bootOrderNone),
			DiscVirtioHotUnplug: new(true),
			Image:               &imageID,
			Name:                new(s.bootstrapVolumeName(ms.IonosMachine)),
			Size:                &size,
			Type:                new(diskType.String()),
			UserData:            new(base64.StdEncoding.EncodeToString(bootstrapData)),
		},
	}, nil
}

// bootstrapVolumeName uses the UID, as machine names are only unique within a namespace,
// while leftover bootstrap volumes are looked up in the whole data center.
func (*Service) bootstrapVolumeName(m *infrav1.IonosCloudMachine) string {
	return "cfg-" + string(m.UID)
}
