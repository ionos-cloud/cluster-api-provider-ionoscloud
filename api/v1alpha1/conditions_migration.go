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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// LegacyConditionMigratedReason is backfilled onto conditions written by CAPIC <= v0.7, which used
// the v1beta1 conditions API and left Reason unset. The v1beta2 metav1.Condition CRD schema requires
// a non-empty, CamelCase Reason; without backfilling it, the first status patch issued against a
// pre-upgrade object after upgrading to v0.8 fails, because the patch helper resends the whole
// status.conditions array, including untouched legacy entries.
const LegacyConditionMigratedReason = "MigratedFromV1Beta1"

// BackfillLegacyConditionReasons sets LegacyConditionMigratedReason on any condition with an empty
// Reason. It mutates and returns the given slice, so it must be called before a patch helper is
// asked to compute a diff, on the same object the helper's "before" snapshot was taken from
// (otherwise the backfill would look like a no-op change to the helper and the invalid legacy
// value would still be resent verbatim).
func BackfillLegacyConditionReasons(conditions []metav1.Condition) []metav1.Condition {
	for i := range conditions {
		if conditions[i].Reason == "" {
			conditions[i].Reason = LegacyConditionMigratedReason
		}
	}
	return conditions
}
