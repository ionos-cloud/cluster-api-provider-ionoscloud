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

package ionosclient

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	*require.Assertions
	suite.Suite
}

func TestIonosCloudClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) SetupSuite() {
	s.Assertions = s.Suite.Require()
}

func (s *ClientTestSuite) TestNewClient() {
	w := "set"
	tcs := []struct {
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
		{w, "", w, w, true},
		{w, "", "", w, false},
		{"", w, w, w, true},
		{"", w, "", w, false},
		{"", "", "", "", false},
	}
	for _, tc := range tcs {
		s.Run(fmt.Sprintf("username=%s password=%s token=%s apiURL=%s (shouldPass=%t)",
			tc.username, tc.password, tc.token, tc.apiURL, tc.shouldPass), func() {
			c, err := NewClient(tc.username, tc.password, tc.token, tc.apiURL)
			if tc.shouldPass {
				s.NotNil(c, "NewClient did not return a nil IonosClient")
				s.NoError(err, "NewClient returned an error")
				cfg := (c.(*IonosClient)).API.GetConfig()
				s.Equal(tc.username, cfg.Username, "username didn't match")
				s.Equal(tc.password, cfg.Password, "password didn't match")
				s.Equal(tc.token, cfg.Token, "token didn't match")
				s.Equal(tc.apiURL, cfg.Host, "apiURL didn't match")
			} else {
				s.Nil(c, "NewClient returned a non-nil client")
				s.Error(err, "NewClient did not return an error")
			}
		})
	}
}
