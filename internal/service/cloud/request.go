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
	"sigs.k8s.io/cluster-api/util"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// GetRequestStatus returns the status of a request for a given request URL.
// If the request contains a message, it is also returned.
func (s *Service) GetRequestStatus(ctx context.Context, requestURL string) (requestStatus string, statusMessage string, err error) {
	status, err := s.ionosClient.CheckRequestStatus(ctx, requestURL)
	if err != nil {
		return "", "", fmt.Errorf("unable to retrieve the request status: %w", err)
	}

	if !status.HasMetadata() || !status.Metadata.HasStatus() {
		return "", "", errors.New("request status metadata is missing")
	}

	message := ""
	if status.Metadata.HasMessage() {
		message = *status.Metadata.Message
	}

	return *status.Metadata.Status, message, nil
}

// mapResourceType maps a cloud resource to its corresponding IONOS Cloud type identifier.
// Each type mapping for usage in getMatchingRequest() needs to be present here.
func mapResourceType(cloudResource any) sdk.Type {
	switch cloudResource.(type) {
	case sdk.Nic, *sdk.Nic:
		return sdk.NIC
	case sdk.Lan, *sdk.Lan:
		return sdk.LAN
	case sdk.Server, *sdk.Server:
		return sdk.SERVER
	case sdk.IpBlock, *sdk.IpBlock:
		return sdk.IPBLOCK
	default:
		return ""
	}
}

type matcherFunc[T any] func(resource T, request sdk.Request) bool

type propertiesHolder[T nameHolder] interface {
	GetProperties() T
}

type nameHolder interface {
	GetName() *string
}

// matchByName is a generic matcher function intended for finding a single resource based on its name.
// The sdk resources provide a Properties field which in turn contains a Name field.
// A compile time check will validate, if the generic types fulfill the interface constraints.
func matchByName[T propertiesHolder[U], U nameHolder](name string) matcherFunc[T] {
	return func(resource T, request sdk.Request) bool {
		properties := resource.GetProperties()
		if util.IsNil(properties) {
			return false
		}

		resourceName := properties.GetName()
		if resourceName != nil && *resourceName == name {
			return true
		}

		return false
	}
}

// getMatchingRequest is a helper function intended for finding a single request based on certain filtering constraints,
// such as HTTP method and URL.
// Requests only containing the given URL, but not ending with it, will be ignored. Note that query parameters are
// stripped before the comparison.
// The function is generic, but only supports types that are present in mapResourceType.
// Moreover, it optionally allows to specify additional matcher functions for the found resource and the request it was
// found in. If multiple matchers are given, all need to match.
// If no matching request is found, nil is returned.
func getMatchingRequest[T any](
	ctx context.Context,
	s *Service,
	method string,
	url string,
	matchers ...matcherFunc[*T],
) (*requestInfo, error) {
	var zeroResource T
	resourceType := mapResourceType(zeroResource)
	if resourceType == "" {
		return nil, fmt.Errorf("unsupported resource type %T", zeroResource)
	}

	// As we later on ignore query parameters in found requests, we need to do the same here for consistency.
	urlWithoutQueryParams := strings.Split(url, "?")[0]

	requests, err := s.ionosClient.GetRequests(ctx, method, urlWithoutQueryParams)
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
		trimmedRequestURL := strings.Split(*req.GetProperties().GetUrl(), "?")[0]
		trimmedRequestURL = strings.TrimSuffix(trimmedRequestURL, "/")
		trimmedURL := strings.TrimSuffix(urlWithoutQueryParams, "/")
		if !strings.HasSuffix(trimmedRequestURL, trimmedURL) {
			continue
		}

		if len(matchers) > 0 {
			// As at least 1 additional matcher function is given, reconstruct the resource from the request body.
			var unmarshalled T
			if err = json.Unmarshal([]byte(ptr.Deref(req.GetProperties().GetBody(), "")), &unmarshalled); err != nil {
				s.logger.WithValues(
					"requestID", ptr.Deref(req.GetId(), ""),
					"body", ptr.Deref(req.GetProperties().GetBody(), ""),
				).Info("could not unmarshal request")

				return nil, fmt.Errorf("could not unmarshal request into %T: %w", unmarshalled, err)
			}

			for _, matcher := range matchers {
				if !matcher(&unmarshalled, req) {
					// All matcher functions need to match the current resource.
					continue requestLoop
				}
			}
		}

		status := ptr.Deref(req.GetMetadata().GetRequestStatus().GetMetadata().GetStatus(), "")
		if status == sdk.RequestStatusFailed {
			message := ptr.Deref(req.GetMetadata().GetRequestStatus().GetMetadata().GetMessage(), "")
			s.logger.Error(nil,
				"Last request has failed, logging it for debugging purposes",
				"resourceType", resourceType,
				"requestID", req.Id, "requestStatus", status,
				"message", message,
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
	targets := ptr.Deref(req.GetMetadata().GetRequestStatus().GetMetadata().GetTargets(), nil)

	for _, target := range targets {
		if ptr.Deref(target.GetTarget().GetType(), "") == typeName {
			return true
		}
	}

	return false
}

type (
	tryLookupResourceFunc[T any] func(context.Context) (*T, error)
	checkQueueFunc               func(context.Context) (*requestInfo, error)
)

// findResource is a helper function intended for finding a single resource.
// It performs a lookup for cloud resources and checks the request queue for matching entries.
// A lookup can consist of GET and LIST operations, including filtering by name or other constraints
// to match exactly one entry.
//
// The function expects two callbacks:
//   - A function looking for a single resource, returning its pointer if found. If errors occur or multiple
//     resources match the constraints, it returns an error. If no resource it found, it returns NotFound or nil.
//   - A function checking the request queue for a matching resource creation and returning information about the
//     request if found. If no request is found, nil is returned. If errors occur, they are returned.
//
// As request queue lookups are rather expensive, we perform the lookup first. If no resource is found, we check the
// request queue. If a request is found and if it is reported as DONE, we assume a possible race condition:
//
// When initially performing the lookup, the request was possibly not DONE yet.
// However, it might be completed right before we checked the request queue.
// In this case, we perform the lookup again, to see if the resource was created in the meantime.
//
// As this process is similar independent of the resource type, this generic function helps to reduce boilerplate.
func findResource[T any](
	ctx context.Context,
	tryLookupResource tryLookupResourceFunc[T],
	checkQueue checkQueueFunc,
) (
	resource *T,
	request *requestInfo,
	err error,
) {
	resource, err = tryLookupResource(ctx)
	if ignoreNotFound(err) != nil {
		return nil, nil, err // Found multiple resources or another error occurred.
	}
	if resource != nil {
		return resource, nil, nil // Something was found, great. No need to look any further.
	}

	request, err = checkQueue(ctx)
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
		resource, err = tryLookupResource(ctx)
		if ignoreNotFound(err) != nil {
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
	return ptr.Deref(resource.GetMetadata().GetState(), unknownValue)
}

func getVMState(resource propertiesHolder[*sdk.ServerProperties]) string {
	return ptr.Deref(resource.GetProperties().GetVmState(), "Unknown")
}

func isRunning(state string) bool {
	return state == "RUNNING"
}

// isAvailable returns true if the resource is available. Note that not all resource types have this state.
func isAvailable(state string) bool {
	return state == sdk.Available
}
