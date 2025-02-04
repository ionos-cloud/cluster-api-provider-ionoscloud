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

// Package ptr offers generic pointer utility functions.
//
// It is implementing the generic ptr functionality from the k8s.io/utils package,
// because the dependency is not versioned and every upstream commit would cause
// Dependabot to create a new PR for a version update.
package ptr

// To returns a pointer to the given value.
func To[T any](v T) *T {
	return &v
}

// Deref attempts to dereference a pointer and return the value.
// If the pointer is nil, the provided default value will be returned instead.
func Deref[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}

// Equal returns true if both arguments are nil or both arguments
// dereference to the same value.
func Equal[T comparable](a, b *T) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return *a == *b
}
