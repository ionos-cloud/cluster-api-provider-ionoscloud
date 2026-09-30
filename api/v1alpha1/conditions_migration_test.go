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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_BackfillLegacyConditionReasons(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionTrue, Reason: ""},
		{Type: "MachineProvisioned", Status: metav1.ConditionTrue, Reason: "Provisioned"},
	}

	got := BackfillLegacyConditionReasons(conditions)

	require.Equal(t, LegacyConditionMigratedReason, got[0].Reason,
		"a condition written by CAPIC <= v0.7 (empty Reason) must be backfilled")
	require.Equal(t, "Provisioned", got[1].Reason,
		"a condition that already carries a Reason must be left untouched")
}
