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

package client

import "errors"

var (
	errDatacenterIDIsEmpty = errors.New("error parsing data center ID: value cannot be empty")
	errServerIDIsEmpty     = errors.New("error parsing server ID: value cannot be empty")
	errLANIDIsEmpty        = errors.New("error parsing LAN ID: value cannot be empty")
	errNICIDIsEmpty        = errors.New("error parsing NIC ID: value cannot be empty")
	errRequestURLIsEmpty   = errors.New("a request URL is necessary for the operation")
	errLocationHeaderEmpty = errors.New(apiNoLocationErrMessage)
)

const (
	apiCallErrWrapper       = "request to Cloud API has failed: %w"
	apiNoLocationErrMessage = "request to Cloud API did not return the request URL"
)
