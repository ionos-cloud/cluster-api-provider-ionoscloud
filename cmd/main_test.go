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

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/cluster-api/controllers/crdmigrator"
)

func TestValidateSkipCRDMigrationPhases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		phases  []string
		wantErr string
	}{
		{name: "no phases skipped", phases: nil},
		{
			name:   "both valid phases",
			phases: []string{"StorageVersionMigration", "CleanupManagedFields"},
		},
		{
			// "All" was advertised by the flag's help text but crdmigrator.setup() rejects it,
			// which made the documented usage crash the manager at startup.
			name:    "All is not a phase",
			phases:  []string{"All"},
			wantErr: `invalid --skip-crd-migration-phases value "All"`,
		},
		{
			name:    "unknown phase among valid ones",
			phases:  []string{"StorageVersionMigration", "Nope"},
			wantErr: `invalid --skip-crd-migration-phases value "Nope"`,
		},
		{
			name:    "phases are case sensitive",
			phases:  []string{"storageversionmigration"},
			wantErr: `invalid --skip-crd-migration-phases value "storageversionmigration"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validateSkipCRDMigrationPhases(test.phases)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

// TestValidSkipCRDMigrationPhasesMatchesCRDMigrator guards against the accepted set drifting away
// from the phases crdmigrator.setup() actually switches on.
func TestValidSkipCRDMigrationPhasesMatchesCRDMigrator(t *testing.T) {
	t.Parallel()

	require.ElementsMatch(t, []string{
		string(crdmigrator.StorageVersionMigrationPhase),
		string(crdmigrator.CleanupManagedFieldsPhase),
	}, validSkipCRDMigrationPhases)
}
