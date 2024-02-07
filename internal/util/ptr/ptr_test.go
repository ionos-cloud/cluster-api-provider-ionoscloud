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

func Test_PtrTo(t *testing.T) {
	type testType struct{}

	require.IsType(t, To(testType{}), (*testType)(nil))
	require.IsType(t, To((*testType)(nil)), (**testType)(nil))
}

func Test_PtrDeref(t *testing.T) {
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

func Test_IsNullOrDefault(t *testing.T) {
	// Test for a custom type
	type testType struct {
		value string
	}
	emptyTestType := &testType{}
	testTypeInstance := &testType{
		value: "test",
	}
	// validate that nil is considered null or default
	var nilTestType *testType
	require.True(t, IsNullOrDefault(nilTestType))
	require.True(t, IsNullOrDefault(emptyTestType))
	// validate that a non-nil value is not considered null or default
	require.False(t, IsNullOrDefault(testTypeInstance))

	// test all primitive types
	var nilBool *bool
	require.True(t, IsNullOrDefault(nilBool))
	require.True(t, IsNullOrDefault(To(false)))
	require.False(t, IsNullOrDefault(To(true)))

	var nilInt *int
	require.True(t, IsNullOrDefault(nilInt))
	require.True(t, IsNullOrDefault(To(0)))
	require.False(t, IsNullOrDefault(To(1)))

	var nilUint *uint
	require.True(t, IsNullOrDefault(nilUint))
	require.True(t, IsNullOrDefault(To(uint(0))))
	require.False(t, IsNullOrDefault(To(uint(1))))

	var nilFloat32 *float32
	require.True(t, IsNullOrDefault(nilFloat32))
	require.True(t, IsNullOrDefault(To(float32(0))))
	require.False(t, IsNullOrDefault(To(float32(1))))

	var nilFloat64 *float64
	require.True(t, IsNullOrDefault(nilFloat64))
	require.True(t, IsNullOrDefault(To(float64(0))))
	require.False(t, IsNullOrDefault(To(float64(1))))

	var nilString *string
	require.True(t, IsNullOrDefault(nilString))
	require.True(t, IsNullOrDefault(To("")))
	require.False(t, IsNullOrDefault(To("test")))

	var nilByte *byte
	require.True(t, IsNullOrDefault(nilByte))
	require.True(t, IsNullOrDefault(To(byte(0))))
	require.False(t, IsNullOrDefault(To(byte(1))))

	var nilInterface *interface{}
	nilInterfaceTyped := (*any)(nil)
	require.True(t, IsNullOrDefault(nilInterface))
	require.True(t, IsNullOrDefault(nilInterfaceTyped))
	require.True(t, IsNullOrDefault(To(any(nil))))
	require.True(t, IsNullOrDefault[any](nil))
	require.False(t, IsNullOrDefault(To(any(1))))
}
