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

package ptr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPtr_To(t *testing.T) {
	type testType struct{}

	require.IsType(t, To(testType{}), (*testType)(nil))
	require.IsType(t, To((*testType)(nil)), (**testType)(nil))
}

func TestPtr_Deref(t *testing.T) {
	type testType struct{}

	testTypeInstance := &testType{}
	// check result types
	require.IsType(t, Deref(&testType{}, testType{}), testType{})
	require.IsType(t, Deref(&testTypeInstance, &testType{}), &testType{})
	// validate that deref returns default when passing a nil value
	var nilTestType *testType
	require.Equal(t, Deref[testType](nilTestType, testType{}), testType{})
	require.Equal(t, Deref[testType](nil, testType{}), testType{})
}
