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

package ionoscloud

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type ClientTestSuite struct {
	*require.Assertions
	suite.Suite
	env *envtest.Environment
}

func TestIonosCloudClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) SetupSuite() {
	s.Assertions = s.Suite.Require()
}

func (s *ClientTestSuite) TearDownTest() {
	if s.env != nil {
		s.NoError(s.env.Stop(), "could not stop test environment")
		s.env = nil
	}
}

func (s *ClientTestSuite) TestNewClientFromEnv() {
	s.T().Setenv(ionoscloud.IonosUsernameEnvVar, "username")
	s.T().Setenv(ionoscloud.IonosPasswordEnvVar, "password")
	s.T().Setenv(ionoscloud.IonosTokenEnvVar, "token")
	s.T().Setenv(ionoscloud.IonosApiUrlEnvVar, "localhost")
	ionos, err := NewClient(context.Background(), nil, nil)
	s.NoError(err, "could not create ionos cloud client")
	s.NotNil(ionos, "NewClient returned nil client")
	cfg := (ionos.(*Client)).api.GetConfig()
	s.Equal("username", cfg.Username)
	s.Equal("password", cfg.Password)
	s.Equal("token", cfg.Token)
	s.Equal("localhost", cfg.Host)
}

func getEnvTest() *envtest.Environment {
	return &envtest.Environment{
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}
}

func (s *ClientTestSuite) TestNewClientFromSecretOk() {
	var err error
	s.T().Setenv(ionoscloud.IonosApiUrlEnvVar, "localhost")
	s.env = getEnvTest()
	cfg, err := s.env.Start()
	s.NoError(err, "could not create kubernetes test environment")
	k8sClient, err := client.New(cfg, client.Options{})
	s.NoError(err, "could not create kubernetes ionos")
	s.NotNil(k8sClient, "ionos.New() returned nil k8sClient")
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "cluster-123",
	}}
	err = k8sClient.Create(context.Background(), namespace)
	s.NoError(err, "could not create namespace")
	secretObjMetadata := metav1.ObjectMeta{
		Name:      "cluster-secret",
		Namespace: "cluster-123",
	}
	secret := &corev1.Secret{
		ObjectMeta: secretObjMetadata,
		StringData: map[string]string{
			"username": "username",
			"password": "password",
			"token":    "token",
		},
	}
	err = k8sClient.Create(context.Background(), secret)
	s.NoError(err, "could not create secret")
	secretRef := &corev1.SecretReference{
		Namespace: secretObjMetadata.Namespace,
		Name:      secretObjMetadata.Name,
	}
	ionos, err := NewClient(context.Background(), k8sClient, secretRef)
	s.NoError(err, "could not create ionos cloud ionos")
	s.NotNil(ionos, "NewClient returned nil ionos")
	ionosCfg := (ionos.(*Client)).api.GetConfig()
	s.Equal("username", ionosCfg.Username)
	s.Equal("password", ionosCfg.Password)
	s.Equal("token", ionosCfg.Token)
	s.Equal("localhost", ionosCfg.Host)
}

func (s *ClientTestSuite) TestNewClientFromSecretError() {
	var err error
	s.T().Setenv(ionoscloud.IonosApiUrlEnvVar, "localhost")
	s.env = getEnvTest()
	cfg, err := s.env.Start()
	s.NoError(err, "could not create kubernetes test environment")
	k8sClient, err := client.New(cfg, client.Options{})
	s.NoError(err, "could not create kubernetes ionos")
	s.NotNil(k8sClient, "ionos.New() returned nil k8sClient")
	s.NoError(err, "could not create secret")
	secretRef := &corev1.SecretReference{
		Namespace: "default",
		Name:      "a-non-secret-secret-name",
	}
	ionos, err := NewClient(context.Background(), k8sClient, secretRef)
	s.Error(err, "error should have been returned from NewClient")
	s.Nil(ionos, "no client should have been returned from NewClient")
}
