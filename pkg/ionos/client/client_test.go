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

package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	w := "SET"
	tests := []struct {
		username   string
		password   string
		token      string
		apiURL     string
		shouldPass bool
	}{
		{w, w, w, w, true},
		{w, w, w, "", true},
		{"", "", w, w, true},
		{"", "", w, "", true},
		{w, w, "", w, true},
		{w, "", w, w, false},
		{w, "", "", w, false},
		{"", w, w, w, false},
		{"", w, "", w, false},
		{"", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("username=%s password=%s token=%s apiURL=%s (shouldPass=%t)",
			tt.username, tt.password, tt.token, tt.apiURL, tt.shouldPass), func(t *testing.T) {
			c, err := NewClient(tt.username, tt.password, tt.token, tt.apiURL)
			if tt.shouldPass {
				require.NotNil(t, c, "NewClient returned a nil IonosClient")
				require.NoError(t, err, "NewClient returned an error")
				cfg := (c.(*IonosClient)).API.GetConfig()
				require.Equal(t, tt.username, cfg.Username, "username didn't match")
				require.Equal(t, tt.password, cfg.Password, "password didn't match")
				require.Equal(t, tt.token, cfg.Token, "token didn't match")
				require.Equal(t, tt.apiURL, cfg.Host, "apiURL didn't match")
			} else {
				require.Nil(t, c, "NewClient returned a non-nil client")
				require.Error(t, err, "NewClient did not return an error")
			}
		})
	}
}
