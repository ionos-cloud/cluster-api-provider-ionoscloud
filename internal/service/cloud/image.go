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
	"errors"
	"fmt"
	"strings"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/labels"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

var errMissingMachineVersion = errors.New("machine is missing version field")

type noImageMatchedError struct {
	selector *infrav1.ImageSelector
}

func (e noImageMatchedError) Error() string {
	return fmt.Sprintf("could not find any images matching selector %q",
		labels.SelectorFromSet(e.selector.MatchLabels))
}

type tooManyImagesMatchError struct {
	imageIDs []string
	selector *infrav1.ImageSelector
}

func (e tooManyImagesMatchError) Error() string {
	return fmt.Sprintf("found %d images that matched selector %q",
		len(e.imageIDs), labels.SelectorFromSet(e.selector.MatchLabels))
}

func (s *Service) lookupImageID(ctx context.Context, ms *scope.Machine) (string, error) {
	imageSpec := ms.IonosMachine.Spec.Disk.Image

	if imageSpec.ID != "" {
		return imageSpec.ID, nil
	}

	images, err := s.lookupImagesBySelector(ctx, ms.ClusterScope.Location(), imageSpec.Selector)
	if err != nil {
		return "", err
	}

	if ptr.Deref(imageSpec.Selector.UseMachineVersion, true) {
		version := ptr.Deref(ms.Machine.Spec.Version, "")
		if version == "" {
			return "", errMissingMachineVersion
		}

		images = filterImagesByName(images, version)
	}

	if len(images) == 0 {
		return "", noImageMatchedError{selector: imageSpec.Selector}
	}

	if len(images) > 1 {
		return "", tooManyImagesMatchError{imageIDs: getImageIDs(images), selector: imageSpec.Selector}
	}

	return *images[0].Id, nil
}

func (s *Service) lookupImagesBySelector(
	ctx context.Context, location string, selector *infrav1.ImageSelector,
) ([]*sdk.Image, error) {
	resourceLabels, err := s.ionosClient.ListLabels(ctx)
	if err != nil {
		return nil, err
	}

	// mapping from image ID to labels
	imageLabelMap := make(map[string]map[string]string)

	for _, label := range resourceLabels {
		if *label.Properties.ResourceType != "image" {
			continue
		}

		id := *label.Properties.ResourceId
		if _, ok := imageLabelMap[id]; !ok {
			imageLabelMap[id] = make(map[string]string)
		}
		imageLabelMap[id][*label.Properties.Key] = *label.Properties.Value
	}

	var imageIDs []string
	for imageID, imageLabels := range imageLabelMap {
		if mapContains(imageLabels, selector.MatchLabels) {
			imageIDs = append(imageIDs, imageID)
		}
	}

	var images []*sdk.Image
	for _, imageID := range imageIDs {
		image, err := s.ionosClient.GetImage(ctx, imageID)
		if err != nil {
			return nil, err
		}

		if *image.Properties.Location == location {
			images = append(images, image)
		}
	}

	return images, nil
}

func filterImagesByName(images []*sdk.Image, namePart string) []*sdk.Image {
	var result []*sdk.Image

	for _, image := range images {
		if strings.Contains(*image.Properties.Name, namePart) {
			result = append(result, image)
		}
	}

	return result
}

func getImageIDs(images []*sdk.Image) []string {
	ids := make([]string, len(images))
	for i, image := range images {
		ids[i] = *image.Id
	}
	return ids
}

// check if b is wholly contained in a.
func mapContains[K, V comparable](a, b map[K]V) bool {
	for k, bv := range b {
		if av, ok := a[k]; !ok || bv != av {
			return false
		}
	}
	return true
}
