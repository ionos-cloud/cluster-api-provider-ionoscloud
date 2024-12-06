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
	"slices"
	"strings"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/labels"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

var errMissingMachineVersion = errors.New("machine is missing version field")

// imageMatchError indicates that either 0 or more than 1 images matched a selector.
// This error signals that user action is required.
type imageMatchError struct {
	imageIDs []string
	selector *infrav1.ImageSelector
}

func (e imageMatchError) Error() string {
	return fmt.Sprintf("found %d images matching selector %q",
		len(e.imageIDs), labels.SelectorFromSet(e.selector.MatchLabels))
}

func (s *Service) lookupImageID(ctx context.Context, ms *scope.Machine) (string, error) {
	imageSpec := ms.IonosMachine.Spec.Disk.Image

	if imageSpec.ID != "" {
		return imageSpec.ID, nil
	}

	location, err := s.getLocation(ctx, ms)
	if err != nil {
		return "", err
	}

	images, err := s.lookupImagesBySelector(ctx, location, imageSpec.Selector)
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

	switch imageSpec.Selector.ResolutionPolicy {
	case infrav1.ResolutionPolicyExact:
		if len(images) != 1 {
			return "", imageMatchError{imageIDs: getImageIDs(images), selector: imageSpec.Selector}
		}
	case infrav1.ResolutionPolicyNewest:
		slices.SortFunc(images, func(lhs, rhs *sdk.Image) int {
			// swap lhs and rhs to produce reverse order
			return rhs.Metadata.CreatedDate.Compare(lhs.Metadata.CreatedDate.Time)
		})
	}

	if len(images) == 0 {
		return "", imageMatchError{selector: imageSpec.Selector}
	}

	return ptr.Deref(images[0].GetId(), ""), nil
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
		if ptr.Deref(label.GetProperties().GetResourceType(), "") != "image" {
			continue
		}

		id := ptr.Deref(label.GetProperties().GetResourceId(), "")
		if _, ok := imageLabelMap[id]; !ok {
			imageLabelMap[id] = make(map[string]string)
		}
		key := ptr.Deref(label.GetProperties().GetKey(), "")
		value := ptr.Deref(label.GetProperties().GetValue(), "")
		imageLabelMap[id][key] = value
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

		if ptr.Deref(image.GetProperties().GetLocation(), "") == location {
			images = append(images, image)
		}
	}

	return images, nil
}

func filterImagesByName(images []*sdk.Image, namePart string) []*sdk.Image {
	var result []*sdk.Image

	for _, image := range images {
		if strings.Contains(ptr.Deref(image.GetProperties().GetName(), ""), namePart) {
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
