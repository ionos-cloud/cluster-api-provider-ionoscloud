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

func Test_IsNilOrZero(t *testing.T) {
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
	require.True(t, IsNilOrZero(nilTestType))
	require.True(t, IsNilOrZero(emptyTestType))
	// validate that a non-nil value is not considered null or default
	require.False(t, IsNilOrZero(testTypeInstance))

	// test all primitive types
	var nilBool *bool
	require.True(t, IsNilOrZero(nilBool))
	require.True(t, IsNilOrZero(To(false)))
	require.False(t, IsNilOrZero(To(true)))

	var nilInt *int
	require.True(t, IsNilOrZero(nilInt))
	require.True(t, IsNilOrZero(To(0)))
	require.False(t, IsNilOrZero(To(1)))

	var nilUint *uint
	require.True(t, IsNilOrZero(nilUint))
	require.True(t, IsNilOrZero(To(uint(0))))
	require.False(t, IsNilOrZero(To(uint(1))))

	var nilFloat32 *float32
	require.True(t, IsNilOrZero(nilFloat32))
	require.True(t, IsNilOrZero(To(float32(0))))
	require.False(t, IsNilOrZero(To(float32(1))))

	var nilFloat64 *float64
	require.True(t, IsNilOrZero(nilFloat64))
	require.True(t, IsNilOrZero(To(float64(0))))
	require.False(t, IsNilOrZero(To(float64(1))))

	var nilString *string
	require.True(t, IsNilOrZero(nilString))
	require.True(t, IsNilOrZero(To("")))
	require.False(t, IsNilOrZero(To("test")))

	var nilByte *byte
	require.True(t, IsNilOrZero(nilByte))
	require.True(t, IsNilOrZero(To(byte(0))))
	require.False(t, IsNilOrZero(To(byte(1))))

	var nilInterface *interface{}
	nilInterfaceTyped := (*any)(nil)
	require.True(t, IsNilOrZero(nilInterface))
	require.True(t, IsNilOrZero(nilInterfaceTyped))
	require.True(t, IsNilOrZero(To(any(nil))))
	require.True(t, IsNilOrZero[any](nil))
	require.False(t, IsNilOrZero(To(any(1))))
}
