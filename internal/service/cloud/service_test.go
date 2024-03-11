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

package cloud

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"

	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

func TestNewServiceValid(t *testing.T) {
	cloud := clienttest.NewMockClient(t)
	log := ptr.To(logr.Discard())
	svc, err := NewService(NewServiceParams{cloud, log})
	require.NotNil(t, svc)
	require.Nil(t, err)
	require.Same(t, cloud, svc.cloud)
}

func TestNewServiceNilLogger(t *testing.T) {
	cloud := clienttest.NewMockClient(t)
	svc, err := NewService(NewServiceParams{cloud, nil})
	require.NotNil(t, svc)
	require.Nil(t, err)
	require.Equal(t, logr.Discard(), *svc.logger)
}

func TestNewServiceNilIONOSCloud(t *testing.T) {
	log := ptr.To(logr.Discard())
	svc, err := NewService(NewServiceParams{nil, log})
	require.Nil(t, svc)
	require.Error(t, err)
}
