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
	"k8s.io/utils/ptr"
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
	require.True(t, IsNullOrDefault(ptr.To(false)))
	require.False(t, IsNullOrDefault(ptr.To(true)))

	var nilInt *int
	require.True(t, IsNullOrDefault(nilInt))
	require.True(t, IsNullOrDefault(ptr.To(0)))
	require.False(t, IsNullOrDefault(ptr.To(1)))

	var nilInt8 *int8
	require.True(t, IsNullOrDefault(nilInt8))
	require.True(t, IsNullOrDefault(ptr.To(int8(0))))
	require.False(t, IsNullOrDefault(ptr.To(int8(1))))

	var nilInt16 *int16
	require.True(t, IsNullOrDefault(nilInt16))
	require.True(t, IsNullOrDefault(ptr.To(int16(0))))
	require.False(t, IsNullOrDefault(ptr.To(int16(1))))

	var nilInt32 *int32
	require.True(t, IsNullOrDefault(nilInt32))
	require.True(t, IsNullOrDefault(ptr.To(int32(0))))
	require.False(t, IsNullOrDefault(ptr.To(int32(1))))

	var nilInt64 *int64
	require.True(t, IsNullOrDefault(nilInt64))
	require.True(t, IsNullOrDefault(ptr.To(int64(0))))
	require.False(t, IsNullOrDefault(ptr.To(int64(1))))

	var nilUint *uint
	require.True(t, IsNullOrDefault(nilUint))
	require.True(t, IsNullOrDefault(ptr.To(uint(0))))
	require.False(t, IsNullOrDefault(ptr.To(uint(1))))

	var nilUint8 *uint8
	require.True(t, IsNullOrDefault(nilUint8))
	require.True(t, IsNullOrDefault(ptr.To(uint8(0))))
	require.False(t, IsNullOrDefault(ptr.To(uint8(1))))

	var nilUint16 *uint16
	require.True(t, IsNullOrDefault(nilUint16))
	require.True(t, IsNullOrDefault(ptr.To(uint16(0))))
	require.False(t, IsNullOrDefault(ptr.To(uint16(1))))

	var nilUint32 *uint32
	require.True(t, IsNullOrDefault(nilUint32))
	require.True(t, IsNullOrDefault(ptr.To(uint32(0))))
	require.False(t, IsNullOrDefault(ptr.To(uint32(1))))

	var nilUint64 *uint64
	require.True(t, IsNullOrDefault(nilUint64))
	require.True(t, IsNullOrDefault(ptr.To(uint64(0))))
	require.False(t, IsNullOrDefault(ptr.To(uint64(1))))

	var nilFloat32 *float32
	require.True(t, IsNullOrDefault(nilFloat32))
	require.True(t, IsNullOrDefault(ptr.To(float32(0))))
	require.False(t, IsNullOrDefault(ptr.To(float32(1))))

	var nilFloat64 *float64
	require.True(t, IsNullOrDefault(nilFloat64))
	require.True(t, IsNullOrDefault(ptr.To(float64(0))))
	require.False(t, IsNullOrDefault(ptr.To(float64(1))))

	var nilString *string
	require.True(t, IsNullOrDefault(nilString))
	require.True(t, IsNullOrDefault(ptr.To("")))
	require.False(t, IsNullOrDefault(ptr.To("test")))

	var nilByte *byte
	require.True(t, IsNullOrDefault(nilByte))
	require.True(t, IsNullOrDefault(ptr.To(byte(0))))
	require.False(t, IsNullOrDefault(ptr.To(byte(1))))

	var nilRune *rune
	require.True(t, IsNullOrDefault(nilRune))
	require.True(t, IsNullOrDefault(ptr.To(rune(0))))
	require.False(t, IsNullOrDefault(ptr.To(rune(1))))

	var nilComplex64 *complex64
	require.True(t, IsNullOrDefault(nilComplex64))
	require.True(t, IsNullOrDefault(ptr.To(complex64(0))))
	require.False(t, IsNullOrDefault(ptr.To(complex64(1))))

	var nilComplex128 *complex128
	require.True(t, IsNullOrDefault(nilComplex128))
	require.True(t, IsNullOrDefault(ptr.To(complex128(0))))
	require.False(t, IsNullOrDefault(ptr.To(complex128(1))))

	var nilInterface *interface{}
	nilInterfaceTyped := (*any)(nil)
	require.True(t, IsNullOrDefault(nilInterface))
	require.True(t, IsNullOrDefault(nilInterfaceTyped))
	require.True(t, IsNullOrDefault(ptr.To(any(nil))))
	require.True(t, IsNullOrDefault[any](nil))
	require.False(t, IsNullOrDefault(ptr.To(any(1))))
}
