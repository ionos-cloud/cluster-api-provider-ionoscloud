/*
Copyright 2023-2024 IONOS Cloud.

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

package client

import "errors"

var (
	errDataCenterIDIsEmpty = errors.New("error parsing data center ID: value cannot be empty")
	errServerIDIsEmpty     = errors.New("error parsing server ID: value cannot be empty")
	errLanIDIsEmpty        = errors.New("error parsing lan ID: value cannot be empty")
	errVolumeIDIsEmpty     = errors.New("error parsing volume ID: value cannot be empty")
	errRequestURLIsEmpty   = errors.New("a request url is necessary for the operation")
)

const (
	apiCallErrWrapper       = "request to Cloud API has failed: %w"
	apiNoLocationErrWrapper = "request to Cloud API did not return the request url"
)
