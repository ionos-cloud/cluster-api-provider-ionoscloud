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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/ionos-cloud/sdk-go/v6"
)

// GetRequestStatus returns the status of a request for a given request URL.
func (s *Service) GetRequestStatus(ctx context.Context, requestURL string) (string, string, error) {
	status, err := s.api().CheckRequestStatus(ctx, requestURL)
	if err != nil {
		return "", "", fmt.Errorf("unable to retrieve the reqest status: %w", err)
	}

	if !status.HasMetadata() && !status.Metadata.HasStatus() {
		return "", "", errors.New("request status metadata is missing")
	}

	message := ""
	if status.Metadata.HasMessage() {
		message = *status.Metadata.Message
	}

	return *status.Metadata.Status, message, nil
}

// resourceTypeMap maps a resource type to its corresponding IONOS Cloud type identifier.
// Each type mapping for usage in getMatchingRequest() needs to be present here.
var resourceTypeMap = map[any]sdk.Type{
	sdk.Lan{}: sdk.LAN,
}

// getMatchingRequest is a helper function intended for finding a single request based on certain filtering constraints,
// such as HTTP method and URL.
// Requests only containing the given URL, but not ending with it, will be ignored. Note that query parameters are
// stripped before the comparison.
// The function is generic, but only supports types that are present in resourceTypeMap.
// Moreover, it optionally allows to specify additional matcher functions for the found resource and the request it was
// found in. If multiple matchers are given, all need to match.
// If no matching request is found, nil is returned.
func getMatchingRequest[T any](
	s *Service,
	method string,
	url string,
	matchers ...func(resource T, request sdk.Request) bool,
) (*requestInfo, error) {
	var zeroResource T
	resourceType := resourceTypeMap[zeroResource]
	if resourceType == "" {
		return nil, fmt.Errorf("unsupported resource type %T", zeroResource)
	}

	// As we later on ignore query parameters in found requests, we need to do the same here for consistency.
	urlWithoutQueryParams := strings.Split(url, "?")[0]

	requests, err := s.api().GetRequests(s.ctx, method, urlWithoutQueryParams)
	if err != nil {
		return nil, fmt.Errorf("could not get requests: %w", err)
	}

requestLoop:
	for _, req := range requests {
		// We only want to look at requests for the desired resource type. We ignore requests for other types.
		// We perform this check because we might deal with a request for URL /resource/123/subresource/456 when looking
		// for /resource/123.
		// In theory the other URL check below is sufficient, but better be safe than sorry.
		if !hasRequestTargetType(req, resourceType) {
			continue
		}

		// Compare the given URL with the one found in the request. We ignore query parameters here as well.
		// For normalization, we also remove trailing slashes.
		// The reason for comparing here at all is that the received requests can contain some that only contain the
		// desired URL as substring, e.g. a request for /resource/123/action will also be returned when looking
		// for /resource/123. We want to ignore those.
		trimmedRequestURL := strings.Split(*req.Properties.Url, "?")[0]
		trimmedRequestURL = strings.TrimSuffix(trimmedRequestURL, "/")
		trimmedURL := strings.TrimSuffix(urlWithoutQueryParams, "/")
		if !strings.HasSuffix(trimmedRequestURL, trimmedURL) {
			continue
		}

		if len(matchers) > 0 {
			// As at least 1 additional matcher function is given, reconstruct the resource from the request body.
			var unmarshalled T
			if err = json.Unmarshal([]byte(*req.Properties.Body), &unmarshalled); err != nil {
				s.scope.Logger.WithValues("requestID", *req.Id, "body", *req.Properties.Body).
					Info("could not unmarshal request")
				return nil, fmt.Errorf("could not unmarshal request into %T: %w", unmarshalled, err)
			}

			for _, matcher := range matchers {
				if !matcher(unmarshalled, req) {
					// All matcher functions need to match the current resource.
					continue requestLoop
				}
			}
		}

		status := *req.Metadata.RequestStatus.Metadata.Status
		if status == sdk.RequestStatusFailed {
			message := req.Metadata.RequestStatus.Metadata.Message
			s.scope.Logger.Error(nil,
				"Last request has failed, logging it for debugging purposes",
				"resourceType", resourceType,
				"requestID", req.Id, "requestStatus", status,
				"message", *message,
			)
		}

		return &requestInfo{
			status:   status,
			location: *req.Metadata.RequestStatus.Href,
		}, nil
	}
	return nil, nil
}

func hasRequestTargetType(req sdk.Request, typeName sdk.Type) bool {
	if req.Metadata.RequestStatus.Metadata.Targets == nil || len(*req.Metadata.RequestStatus.Metadata.Targets) == 0 {
		return false
	}
	return *(*req.Metadata.RequestStatus.Metadata.Targets)[0].Target.Type == typeName
}

// findResource is a helper function intended for finding a single resource based on certain filtering constraints,
// such as a unique name. It lists and filters the existing resources and checks the request queue for matching
// creations.
// The function expects two callbacks:
//   - A function listing and looking for a single resource, returning its pointer if found. If errors occur or multiple
//     resources match the constraints, it returns an error. If no resource it found, it returns nil.
//   - A function checking the request queue for a matching resource creation and returning information about the
//     request if found. If no request is found, nil is returned. If errors occur, they are returned.
//
// As request queue lookups are rather expensive, we list and filter first. If no resource is found, we check the
// request queue. If a request is found and if it is reported as DONE, we assume a possible race condition:
// When initially listing the resources, the request was possibly not DONE yet. Not it is, so we list and filter again
// to see if the resource was created in the meantime.
// As this process is similar independent of the resource type, this generic function helps to reduce boilerplate.
func findResource[T any](
	listAndFilter func() (*T, error),
	checkQueue func() (*requestInfo, error),
) (
	resource *T,
	request *requestInfo,
	err error,
) {
	resource, err = listAndFilter()
	if err != nil {
		return nil, nil, err // Found multiple resources or another error occurred.
	}
	if resource != nil {
		return resource, nil, nil // Something was found, great. No need to look any further.
	}

	request, err = checkQueue()
	if err != nil {
		return nil, nil, err
	}

	if request == nil {
		return nil, nil, nil // resource not found
	}

	if request.isDone() {
		// To prevent duplicate creation, check again. The request might have been completed right after we listed
		// initially.
		// Note that it can happen that even now we don't find a resource. This can happen if we found an old creation
		// request, but the resource was already deleted later on.
		resource, err = listAndFilter()
		if err != nil {
			return nil, nil, err // Found multiple resources or another error occurred.
		}
		if resource != nil {
			return resource, nil, nil // Something was found this time, great.
		}
	}

	return nil, request, nil // found no existing resource, but at least a matching request
}

// requestInfo is a stripped-down version of sdk.RequestStatus, containing only the request status and the request's
// location URL which can be used for polling.
type requestInfo struct {
	status   string
	location string
}

// isPending returns true if the request was not finished yet, i.e. it's still queued or currently running.
func (ri *requestInfo) isPending() bool {
	return ri.status == sdk.RequestStatusQueued || ri.status == sdk.RequestStatusRunning
}

// isDone returns true if the request was finished successfully.
func (ri *requestInfo) isDone() bool {
	return ri.status == sdk.RequestStatusDone
}

type metadataHolder interface {
	GetMetadata() *sdk.DatacenterElementMetadata
}

func getState(resource metadataHolder) string {
	return *resource.GetMetadata().State
}

const stateAvailable = "AVAILABLE"

// isAvailable returns true if the resource is available. Note that not all resource types have this state.
func isAvailable(state string) bool {
	return state == stateAvailable
}
