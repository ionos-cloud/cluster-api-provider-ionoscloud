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

// RequestStatus shows the status of the current request.
type RequestStatus string

const (
	// RequestStatusQueued indicates, that the request is queued and not yet being processed.
	RequestStatusQueued RequestStatus = "QUEUED"

	// RequestStatusRunning indicates, that the request is currently being processed.
	RequestStatusRunning RequestStatus = "RUNNING"

	// RequestStatusDone indicates, that the request has been successfully processed.
	RequestStatusDone RequestStatus = "DONE"

	// RequestStatusFailed indicates, that the request has failed.
	RequestStatusFailed RequestStatus = "FAILED"
)

// ProvisioningRequest is a definition of a provisioning request
// in the IONOS Cloud.
type ProvisioningRequest struct {
	// Method is the request method
	Method string `json:"method"`

	// RequestPath is the sub path for the request URL
	RequestPath string `json:"requestPath"`

	// RequestStatus is the status of the request in the queue.
	// +kubebuilder:validation:Enum=QUEUED;RUNNING;DONE;FAILED
	// +optional
	State RequestStatus `json:"state"`

	// Message is the request message, which can also contain error information.
	// +optional
	Message string `json:"failureMessage,omitempty"`
}
