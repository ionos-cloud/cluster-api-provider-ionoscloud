package ptr

import (
	"github.com/stretchr/testify/require"
	"testing"
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
