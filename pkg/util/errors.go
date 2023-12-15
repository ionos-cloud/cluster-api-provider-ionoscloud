/*
Copyright 2023 IONOS Cloud.

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

// Package util brings some convenience functions that can be reused in the project
package util

import "fmt"

// WrapErrorf is a helper function for fmt.Errorf(). The returned message will have the format "%s: %w".
func WrapErrorf(err error, format string, args ...any) error {
	formattedMessage := fmt.Sprintf(format, args...)
	return WrapError(err, formattedMessage)
}

// WrapError is a helper function for wrapping errors using fmt.Errorf(). The returned
// message will have the format "%s: %w".
func WrapError(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}
