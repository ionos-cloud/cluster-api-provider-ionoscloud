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
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type imageTestSuite struct {
	ServiceTestSuite
}

func TestImageService(t *testing.T) {
	suite.Run(t, new(imageTestSuite))
}

func (s *imageTestSuite) SetupTest() {
	s.ServiceTestSuite.SetupTest()
	s.infraMachine.Spec.Disk.Image.ID = ""
	s.infraMachine.Spec.Disk.Image.Selector = &infrav1.ImageSelector{
		MatchLabels: map[string]string{
			"test": "image",
		},
	}
}

func (s *imageTestSuite) TestLookupImageUseImageSpecID() {
	s.infraMachine.Spec.Disk.Image.ID = "someid"
	imageID, err := s.service.lookupImageID(s.ctx, s.machineScope)
	s.NoError(err)
	s.Equal("someid", imageID)
}

func (s *imageTestSuite) TestLookupImageNoMatch() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{makeTestLabel("image", "image-1", "no", "match")}, nil,
	).Once()
	s.ionosClient.EXPECT().GetDatacenterLocationByID(s.ctx, s.infraMachine.Spec.DatacenterID).Return("loc", nil).Once()

	_, err := s.service.lookupImageID(s.ctx, s.machineScope)
	typedErr := new(imageMatchError)
	s.ErrorAs(err, typedErr)
	s.Empty(typedErr.imageIDs)
}

func (s *imageTestSuite) TestLookupImageTooManyMatches() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{
			makeTestLabel("image", "image-1", "test", "image"),
			makeTestLabel("image", "image-2", "test", "image"),
			makeTestLabel("image", "image-3", "test", "image"),
		}, nil,
	).Once()
	s.ionosClient.EXPECT().GetDatacenterLocationByID(s.ctx, s.infraMachine.Spec.DatacenterID).Return("loc", nil).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-1").Return(makeTestImage("image-1", "img-foo-v1.2.3", "test"), nil).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-2").Return(s.makeTestImage("image-2", "img-foo-", "loc"), nil).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-3").Return(s.makeTestImage("image-3", "img-bar-", "loc"), nil).Once()

	_, err := s.service.lookupImageID(s.ctx, s.machineScope)
	typedErr := new(imageMatchError)
	s.ErrorAs(err, typedErr)
	slices.Sort(typedErr.imageIDs)
	s.Equal([]string{"image-2", "image-3"}, typedErr.imageIDs)
}

func (s *imageTestSuite) TestLookupImageMissingMachineVersion() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{
			makeTestLabel("image", "image-1", "test", "image"),
		}, nil,
	).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-1").Return(s.makeTestImage("image-1", "test", "loc"), nil).Once()
	s.ionosClient.EXPECT().GetDatacenterLocationByID(s.ctx, s.infraMachine.Spec.DatacenterID).Return("loc", nil).Once()

	s.capiMachine.Spec.Version = ptr.To("")

	_, err := s.service.lookupImageID(s.ctx, s.machineScope)
	s.ErrorIs(err, errMissingMachineVersion)
}

func (s *imageTestSuite) TestLookupImageIgnoreMissingMachineVersion() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{
			makeTestLabel("image", "image-1", "test", "image"),
		}, nil,
	).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-1").Return(s.makeTestImage("image-1", "test", "loc"), nil).Once()
	s.ionosClient.EXPECT().GetDatacenterLocationByID(s.ctx, s.infraMachine.Spec.DatacenterID).Return("loc", nil).Once()

	s.infraMachine.Spec.Disk.Image.Selector.UseMachineVersion = ptr.To(false)
	s.capiMachine.Spec.Version = ptr.To("")

	imageID, err := s.service.lookupImageID(s.ctx, s.machineScope)
	s.NoError(err)
	s.Equal("image-1", imageID)
}

func (s *imageTestSuite) TestLookupImageOK() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{
			makeTestLabel("image", "image-1", "test", "image"),
		}, nil,
	).Once()
	s.ionosClient.EXPECT().GetDatacenterLocationByID(s.ctx, s.infraMachine.Spec.DatacenterID).Return("loc", nil).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-1").Return(s.makeTestImage("image-1", "img-", "loc"), nil).Once()

	imageID, err := s.service.lookupImageID(s.ctx, s.machineScope)
	s.NoError(err)
	s.Equal("image-1", imageID)
}

func (s *imageTestSuite) makeTestImage(id, namePrefix, location string) *sdk.Image {
	return makeTestImage(id, namePrefix+*s.capiMachine.Spec.Version, location)
}

func TestFilterImagesByName(t *testing.T) {
	images := []*sdk.Image{
		makeTestImage("image-1", "img-foo-v1.1.qcow2", "test"),
		makeTestImage("image-2", "img-foo-v1.2.qcow2", "test"),
		makeTestImage("image-3", "img-bar-1.2.3.qcow2", "test"),
	}
	expectImages := []*sdk.Image{
		makeTestImage("image-2", "img-foo-v1.2.qcow2", "test"),
		makeTestImage("image-3", "img-bar-1.2.3.qcow2", "test"),
	}

	require.Equal(t, expectImages, filterImagesByName(images, "1.2"))
	require.Equal(t, images, filterImagesByName(images, ""))
}

func TestLookupImagesBySelector(t *testing.T) {
	ctx := context.Background()
	ionosClient := clienttest.NewMockClient(t)
	ionosClient.EXPECT().ListLabels(ctx).Return([]sdk.Label{
		// wrong resource type
		makeTestLabel("server", "server-1", "foo", "bar"),
		makeTestLabel("volume", "volume-1", "foo", "bar"),
		// no match
		makeTestLabel("image", "image-1", "baz", "qux"),
		makeTestLabel("image", "image-2", "baz", "qux"),
		makeTestLabel("image", "image-2", "lazy", "brown"),
		// partial matche
		makeTestLabel("image", "image-1", "over", "the"),
		makeTestLabel("image", "image-2", "fox", "jumps"),
		// full match
		makeTestLabel("image", "image-1", "foo", "bar"),
		makeTestLabel("image", "image-4", "foo", "bar"),
		makeTestLabel("image", "image-1", "fox", "jumps"),
		makeTestLabel("image", "image-4", "fox", "jumps"),
	}, nil).Once()
	ionosClient.EXPECT().GetImage(ctx, "image-1").Return(makeTestImage("image-1", "img-foo-v1.1.qcow2", "loc-1"), nil).Once()
	ionosClient.EXPECT().GetImage(ctx, "image-4").Return(makeTestImage("image-4", "img-foo-v1.1.qcow2", "loc-2"), nil).Once()

	selector := &infrav1.ImageSelector{
		MatchLabels: map[string]string{
			"foo": "bar",
			"fox": "jumps",
		},
	}

	expectImages := []*sdk.Image{makeTestImage("image-1", "img-foo-v1.1.qcow2", "loc-1")}

	svc, _ := NewService(ionosClient, logr.Discard())
	images, err := svc.lookupImagesBySelector(ctx, "loc-1", selector)
	require.NoError(t, err)
	require.Equal(t, expectImages, images)
}

func makeTestImage(id, name, location string) *sdk.Image {
	return &sdk.Image{
		Id: &id,
		Properties: &sdk.ImageProperties{
			Name:     &name,
			Location: &location,
		},
	}
}

func makeTestLabel(typ, id, key, value string) sdk.Label {
	return sdk.Label{
		Id: ptr.To(fmt.Sprintf("urn:label:%s:%s:%s", typ, id, key)),
		Properties: &sdk.LabelProperties{
			Key:          &key,
			Value:        &value,
			ResourceId:   &id,
			ResourceType: &typ,
		},
	}
}
