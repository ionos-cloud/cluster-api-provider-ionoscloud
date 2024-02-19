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

package controller

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"
)

const (
	defaultReconcileDuration = time.Second * 20
)

type serviceReconcileStep struct {
	name          string
	reconcileFunc func() (requeue bool, err error)
}

// withStatus is a helper function to handle the different request states
// and provides a callback function to execute when the request is done or failed.
func withStatus(
	status string,
	message string,
	log *logr.Logger,
	doneOrFailedCallback func() error,
) (requeue bool, err error) {
	switch status {
	case sdk.RequestStatusQueued, sdk.RequestStatusRunning:
		return true, nil
	case sdk.RequestStatusFailed:
		// log the error message
		log.Error(nil, "Request status indicates a failure", "message", message)
		fallthrough // we run the same logic as for status done
	case sdk.RequestStatusDone:
		// we don't requeue
		return false, doneOrFailedCallback()
	}

	return false, fmt.Errorf("unknown request status %s", status)
}
