//go:build e2e
// +build e2e

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

package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdk "github.com/ionos-cloud/sdk-go/v6"
)

// NOTE(gfariasalves-ionos): This file exists to avoid having to use Terraform to create the resources needed for the tests,
// specially when we just need a data center (and for now also a IP Block).
// TODO(gfariasalves-ionos): Remove IP block reservation and deletion after automatic IP Block reservation is working.

const (
	apiCallErrWrapper       = "request to Cloud API has failed: %w"
	apiNoLocationErrMessage = "request to Cloud API did not return the request URL"

	apiLocationHeaderKey = "Location"
)

var (
	errLocationHeaderEmpty = errors.New(apiNoLocationErrMessage)
)

type ionosCloudEnv struct {
	api          *sdk.APIClient
	token        string
	datacenterID string
	ipBlock      *sdk.IpBlock
}

func (e *ionosCloudEnv) setup() {
	var err error

	e.token = os.Getenv(sdk.IonosTokenEnvVar)
	Expect(e.token).ToNot(BeEmpty(), "Please set the IONOS_TOKEN environment variable")
	e.api = sdk.NewAPIClient(sdk.NewConfigurationFromEnv())

	location := os.Getenv("CONTROL_PLANE_ENDPOINT_LOCATION")

	By("Requesting a data center")
	dcRequest, err := e.createDatacenter(ctx, location)
	Expect(err).ToNot(HaveOccurred(), "Failed requesting data center creation")

	By("Requesting an IP block")
	ipbRequest, err := e.reserveIPBlock(ctx, location, 1)
	Expect(err).ToNot(HaveOccurred(), "Failed requesting IP block reservation")

	By("Waiting for requests to complete")
	Expect(e.waitForCreationRequests(ctx, dcRequest, ipbRequest)).ToNot(HaveOccurred(), "Failed waiting for requests to complete")
}

func (e *ionosCloudEnv) teardown() {
	if !skipCleanup {
		By("Deleting environment resources")

		By("Requesting the deletion of the data center")
		datacenterRequest, err := e.deleteDatacenter(ctx)
		Expect(err).ToNot(HaveOccurred(), "Failed asking to delete data center")

		By("Requesting the deletion of the IP Block")
		ipBlockRequest, err := e.deleteIPBlock(ctx)
		Expect(err).ToNot(HaveOccurred(), "Failed asking to delete IP Block")

		By("Waiting for deletion requests to complete")
		Expect(e.waitForDeletionRequests(ctx, datacenterRequest, ipBlockRequest)).ToNot(HaveOccurred(), "Failed waiting for deletion requests to complete")
	}
}

func (e *ionosCloudEnv) createDatacenter(ctx context.Context, location string) (requestID string, _ error) {
	if location == "" {
		return "", errors.New("location must be set")
	}
	name := fmt.Sprintf("%s-%s", "capic-e2e-test", uuid.New().String())
	description := "used in a CACIC E2E test run"
	if os.Getenv("CI") == "true" {
		name = fmt.Sprintf("%s-%s", "capic-e2e-test", os.Getenv("GITHUB_RUN_ID"))
		description = fmt.Sprintf("CI run URL: %s", e.githubCIRunURL())
	}

	datacenter := sdk.Datacenter{
		Properties: &sdk.DatacenterProperties{
			Name:        &name,
			Location:    &location,
			Description: &description,
		},
	}
	datacenter, res, err := e.api.DataCentersApi.DatacentersPost(ctx).Datacenter(datacenter).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if datacenter.Id == nil {
		return "", fmt.Errorf(apiCallErrWrapper, errors.New("request to Cloud API did not return the data center"))
	}
	e.datacenterID = *datacenter.Id
	Expect(os.Setenv("IONOSCLOUD_DATACENTER_ID", e.datacenterID)).ToNot(HaveOccurred(), "Failed setting datacenter ID in environment variable")
	if requestLocation := res.Header.Get(apiLocationHeaderKey); requestLocation != "" {
		return requestLocation, nil
	}
	return "", errLocationHeaderEmpty
}

// deleteDatacenter requests the deletion of the data center that matches the provided id.
func (e *ionosCloudEnv) deleteDatacenter(ctx context.Context) (requestLocation string, _ error) {
	res, err := e.api.DataCentersApi.DatacentersDelete(ctx, e.datacenterID).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if requestLocation := res.Header.Get(apiLocationHeaderKey); requestLocation != "" {
		return requestLocation, nil
	}
	return "", errLocationHeaderEmpty
}

func (e *ionosCloudEnv) reserveIPBlock(ctx context.Context, location string, size int32) (requestID string, _ error) {
	name := fmt.Sprintf("%s", "CAPIC E2E Test")
	if os.Getenv("CI") == "true" {
		name = fmt.Sprintf("CAPIC E2E Test - %s", e.githubCIRunURL())
	}
	ipBlock := sdk.IpBlock{
		Properties: &sdk.IpBlockProperties{
			Name:     &name,
			Size:     &size,
			Location: &location,
		},
	}
	ipb, req, err := e.api.IPBlocksApi.IpblocksPost(ctx).Ipblock(ipBlock).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}

	e.ipBlock = &ipb
	Expect(os.Setenv("CONTROL_PLANE_ENDPOINT_IP", (*e.ipBlock.Properties.Ips)[0])).ToNot(HaveOccurred(), "Failed setting datacenter ID in environment variable")

	if requestPath := req.Header.Get(apiLocationHeaderKey); requestPath != "" {
		return requestPath, nil
	}
	return "", errLocationHeaderEmpty
}

func (e *ionosCloudEnv) deleteIPBlock(ctx context.Context) (requestLocation string, _ error) {
	res, err := e.api.IPBlocksApi.IpblocksDelete(ctx, *e.ipBlock.Id).Execute()
	if err != nil {
		return "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if requestLocation := res.Header.Get(apiLocationHeaderKey); requestLocation != "" {
		return requestLocation, nil
	}
	return "", errLocationHeaderEmpty
}

func (e *ionosCloudEnv) waitForCreationRequests(ctx context.Context, datacenterRequest, ipBlockRequest string) error {
	GinkgoLogr.Info("Waiting for data center and IP block creation requests to complete",
		"datacenterRequest", datacenterRequest,
		"datacenterID", e.datacenterID,
		"ipBlockRequest", ipBlockRequest,
		"ipBlockID", *e.ipBlock.Id)

	_, err := e.api.WaitForRequest(ctx, datacenterRequest)
	if err != nil {
		return fmt.Errorf("failed waiting for data center creation: %w", err)
	}
	_, err = e.api.WaitForRequest(ctx, ipBlockRequest)
	if err != nil {
		return fmt.Errorf("failed waiting for IP block reservation: %w", err)
	}
	return nil
}

func (e *ionosCloudEnv) waitForDeletionRequests(ctx context.Context, datacenterRequest, ipBlockRequest string) error {
	GinkgoLogr.Info("Waiting for data center and IP block deletion requests to complete",
		"datacenterRequest", datacenterRequest,
		"datacenterID", e.datacenterID,
		"ipBlockRequest", ipBlockRequest,
		"ipBlockID", *e.ipBlock.Id)

	_, err := e.api.WaitForRequest(ctx, datacenterRequest)
	if err != nil {
		return fmt.Errorf("failed waiting for data center deletion: %w", err)
	}
	_, err = e.api.WaitForRequest(ctx, ipBlockRequest)
	if err != nil {
		return fmt.Errorf("failed waiting for IP block deletion: %w", err)
	}
	return nil
}

// createCredentialsSecretPNC creates a secret with the IONOS Cloud credentials. This secret should be used as the
// argument for the PostNamespaceCreation attribute of the spec input.
func (e *ionosCloudEnv) createCredentialsSecretPNC(clusterProxy framework.ClusterProxy, namespace string) {
	k8sClient := clusterProxy.GetClient()

	namespacedName := types.NamespacedName{
		Name:      "ionoscloud-credentials",
		Namespace: namespace,
	}

	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, namespacedName, secret)
	Expect(ignoreNotFound(err)).To(Succeed(), "could not get credentials secret")

	if apierrors.IsNotFound(err) {
		By(fmt.Sprintf("Creating credentials secret for namespace %q", namespace))
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
			StringData: map[string]string{
				"token": e.token,
			},
		}
		Expect(clusterProxy.GetClient().Create(ctx, secret)).To(Succeed(), "could not create credentials secret")
	} else {
		By(fmt.Sprintf("Skipping creation of credentials secret for namespace %q as it already exists", namespace))
	}
}

// githubCIRunURL returns the URL of the current GitHub CI run.
func (e *ionosCloudEnv) githubCIRunURL() string {
	return fmt.Sprintf("%s/%s/actions/runs/%s",
		os.Getenv("GITHUB_SERVER_URL"),
		os.Getenv("GITHUB_REPOSITORY"),
		os.Getenv("GITHUB_RUN_ID"))
}

// ignoreNotFound returns nil if the client.Client.Create() error is a not found error, otherwise it returns the error.
func ignoreNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
