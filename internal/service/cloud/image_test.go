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

	_, err := s.service.lookupImageID(s.ctx, s.machineScope)
	s.ErrorAs(err, new(noImageMatchedError))
}

func (s *imageTestSuite) TestLookupImageTooManyMatches() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{
			makeTestLabel("image", "image-1", "test", "image"),
			makeTestLabel("image", "image-2", "test", "image"),
			makeTestLabel("image", "image-3", "test", "image"),
		}, nil,
	).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-1").Return(makeTestImage("img-foo-v1.2.3"), nil).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-2").Return(makeTestImage("img-foo-"+*s.capiMachine.Spec.Version), nil).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-3").Return(makeTestImage("img-bar-"+*s.capiMachine.Spec.Version), nil).Once()

	_, err := s.service.lookupImageID(s.ctx, s.machineScope)
	typedErr := new(tooManyImagesMatchError)
	s.ErrorAs(err, typedErr)
	slices.Sort(typedErr.imageIDs)
	s.Equal([]string{"image-2", "image-3"}, typedErr.imageIDs)
}

func (s *imageTestSuite) TestLookupImageOK() {
	s.ionosClient.EXPECT().ListLabels(s.ctx).Return(
		[]sdk.Label{
			makeTestLabel("image", "image-1", "test", "image"),
		}, nil,
	).Once()
	s.ionosClient.EXPECT().GetImage(s.ctx, "image-1").Return(makeTestImage("img-"+*s.capiMachine.Spec.Version), nil).Once()

	imageID, err := s.service.lookupImageID(s.ctx, s.machineScope)
	s.NoError(err)
	s.Equal("image-1", imageID)
}

func TestFilterImagesByName(t *testing.T) {
	ctx := context.Background()
	ionosClient := clienttest.NewMockClient(t)
	imageIDs := []string{"image-1", "image-2", "image-3"}
	ionosClient.EXPECT().GetImage(ctx, "image-1").Return(makeTestImage("img-foo-v1.1.qcow2"), nil).Twice()
	ionosClient.EXPECT().GetImage(ctx, "image-2").Return(makeTestImage("img-foo-v1.2.qcow2"), nil).Twice()
	ionosClient.EXPECT().GetImage(ctx, "image-3").Return(makeTestImage("img-bar-1.2.3.qcow2"), nil).Twice()

	gotIDs, err := filterImagesByName(ctx, ionosClient, imageIDs, "1.2")
	require.NoError(t, err)
	require.Equal(t, []string{"image-2", "image-3"}, gotIDs)

	gotIDs, err = filterImagesByName(ctx, ionosClient, imageIDs, "")
	require.NoError(t, err)
	require.Equal(t, imageIDs, gotIDs)
}

func TestFilterImagesBySelect(t *testing.T) {
	labels := []sdk.Label{
		makeTestLabel("image", "image-1", "foo", "bar"),
		makeTestLabel("server", "server-1", "foo", "bar"),
		makeTestLabel("image", "image-1", "baz", "qux"),
		makeTestLabel("volume", "volume-1", "foo", "bar"),
		makeTestLabel("image", "image-2", "baz", "qux"),
		makeTestLabel("image", "image-2", "lazy", "brown"),
		makeTestLabel("image", "image-1", "fox", "jumps"),
		makeTestLabel("image", "image-2", "fox", "jumps"),
		makeTestLabel("image", "image-1", "over", "the"),
	}
	selector := &infrav1.ImageSelector{
		MatchLabels: map[string]string{
			"foo": "bar",
			"fox": "jumps",
		},
	}
	expectIDs := []string{"image-1"}

	ids := filterImagesBySelector(labels, selector)
	slices.Sort(ids)
	require.Equal(t, expectIDs, ids)
}

func makeTestImage(name string) *sdk.Image {
	return &sdk.Image{
		Properties: &sdk.ImageProperties{
			Name: &name,
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
