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
	"fmt"
	"os"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/test/framework"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdk "github.com/ionos-cloud/sdk-go/v6"
)

// NOTE(gfariasalves-ionos): This file exists to avoid having to use Terraform to create the resources needed for the tests,
// specially when we just need a data center (and for now also a IP Block).
// TODO(gfariasalves-ionos): Remove IP block reservation and deletion after automatic IP Block reservation is working.

const (
	apiLocationHeaderKey = "Location"
)

type ionosCloudEnv struct {
	api          *sdk.APIClient
	token        string
	datacenterID string
	ipBlock      *sdk.IpBlock
}

func (e *ionosCloudEnv) setup() {
	e.token = os.Getenv(sdk.IonosTokenEnvVar)
	Expect(e.token).ToNot(BeEmpty(), "Please set the IONOS_TOKEN environment variable")
	e.api = sdk.NewAPIClient(sdk.NewConfigurationFromEnv())

	location := os.Getenv("CONTROL_PLANE_ENDPOINT_LOCATION")

	By("Requesting a data center")
	dcRequest := e.createDatacenter(ctx, location)

	By("Requesting an IP block")
	ipbRequest := e.reserveIPBlock(ctx, location, 1)

	By("Waiting for requests to complete")
	e.waitForCreationRequests(ctx, dcRequest, ipbRequest)
}

func (e *ionosCloudEnv) teardown() {
	if !skipCleanup {
		By("Deleting environment resources")

		By("Requesting the deletion of the data center")
		datacenterRequest := e.deleteDatacenter(ctx)

		By("Requesting the deletion of the IP Block")
		ipBlockRequest := e.deleteIPBlock(ctx)

		By("Waiting for deletion requests to complete")
		e.waitForDeletionRequests(ctx, datacenterRequest, ipBlockRequest)
	}
}

func (e *ionosCloudEnv) createDatacenter(ctx context.Context, location string) (requestLocation string) {
	name := fmt.Sprintf("capic-e2e-test-%s", uuid.New().String())
	description := "used in a CACIC E2E test run"
	if os.Getenv("CI") == "true" {
		name = fmt.Sprintf("capic-e2e-test-%s", os.Getenv("GITHUB_RUN_ID"))
		description = fmt.Sprintf("CI run: %s", e.githubCIRunURL())
	}
	datacenter := sdk.Datacenter{
		Properties: &sdk.DatacenterProperties{
			Name:        &name,
			Location:    &location,
			Description: &description,
		},
	}
	datacenter, res, err := e.api.DataCentersApi.DatacentersPost(ctx).Datacenter(datacenter).Execute()
	Expect(err).ToNot(HaveOccurred(), "Failed requesting data center creation")
	e.datacenterID = *datacenter.Id
	if os.Getenv("CI") == "true" {
		e.writeToGithubOutput("DATACENTER_ID", e.datacenterID)
	}
	Expect(os.Setenv("IONOSCLOUD_DATACENTER_ID", e.datacenterID)).ToNot(HaveOccurred(), "Failed setting datacenter ID in environment variable")
	return res.Header.Get(apiLocationHeaderKey)
}

// deleteDatacenter requests the deletion of the data center that matches the provided id.
func (e *ionosCloudEnv) deleteDatacenter(ctx context.Context) (requestLocation string) {
	res, err := e.api.DataCentersApi.DatacentersDelete(ctx, e.datacenterID).Execute()
	Expect(err).ToNot(HaveOccurred(), "Failed requesting data center deletion")
	return res.Header.Get(apiLocationHeaderKey)
}

func (e *ionosCloudEnv) reserveIPBlock(ctx context.Context, location string, size int32) (requestLocation string) {
	name := fmt.Sprintf("capic-e2e-test-%s", uuid.New().String())
	if os.Getenv("CI") == "true" {
		name = fmt.Sprintf("capic-e2e-test - %s", e.githubCIRunURL())
	}
	ipBlock := sdk.IpBlock{
		Properties: &sdk.IpBlockProperties{
			Name:     &name,
			Size:     &size,
			Location: &location,
		},
	}
	ipb, res, err := e.api.IPBlocksApi.IpblocksPost(ctx).Ipblock(ipBlock).Execute()
	Expect(err).ToNot(HaveOccurred(), "Failed requesting IP block reservation")
	e.ipBlock = &ipb
	if os.Getenv("CI") == "true" {
		e.writeToGithubOutput("IP_BLOCK_ID", *e.ipBlock.Id)
	}
	Expect(os.Setenv("CONTROL_PLANE_ENDPOINT_IP", (*e.ipBlock.Properties.Ips)[0])).ToNot(HaveOccurred(), "Failed setting datacenter ID in environment variable")
	return res.Header.Get(apiLocationHeaderKey)
}

func (e *ionosCloudEnv) deleteIPBlock(ctx context.Context) (requestLocation string) {
	res, err := e.api.IPBlocksApi.IpblocksDelete(ctx, *e.ipBlock.Id).Execute()
	Expect(err).ToNot(HaveOccurred(), "Failed requesting IP block deletion")
	return res.Header.Get(apiLocationHeaderKey)
}

func (e *ionosCloudEnv) waitForCreationRequests(ctx context.Context, datacenterRequest, ipBlockRequest string) {
	GinkgoLogr.Info("Waiting for data center and IP block creation requests to complete",
		"datacenterRequest", datacenterRequest,
		"datacenterID", e.datacenterID,
		"ipBlockRequest", ipBlockRequest,
		"ipBlockID", *e.ipBlock.Id)

	_, err := e.api.WaitForRequest(ctx, datacenterRequest)
	Expect(err).ToNot(HaveOccurred(), "failed waiting for data center creation")
	_, err = e.api.WaitForRequest(ctx, ipBlockRequest)
	Expect(err).ToNot(HaveOccurred(), "failed waiting for IP block reservation")
}

func (e *ionosCloudEnv) waitForDeletionRequests(ctx context.Context, datacenterRequest, ipBlockRequest string) {
	GinkgoLogr.Info("Waiting for data center and IP block deletion requests to complete",
		"datacenterRequest", datacenterRequest,
		"datacenterID", e.datacenterID,
		"ipBlockRequest", ipBlockRequest,
		"ipBlockID", *e.ipBlock.Id)

	_, err := e.api.WaitForRequest(ctx, datacenterRequest)
	Expect(err).ToNot(HaveOccurred(), "failed waiting for data center deletion")
	_, err = e.api.WaitForRequest(ctx, ipBlockRequest)
	Expect(err).ToNot(HaveOccurred(), "failed waiting for IP block deletion")
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
	Expect(runtimeclient.IgnoreNotFound(err)).To(Succeed(), "could not get credentials secret")

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

// writeToGithubOutput writes a key-value pair to the GITHUB_OUTPUT in an action. This function is useful for the
// delete leftovers script.
func (e *ionosCloudEnv) writeToGithubOutput(key, value string) {
	f, err := os.OpenFile(os.Getenv("GITHUB_OUTPUT"), os.O_APPEND|os.O_WRONLY, 0644)
	Expect(err).ToNot(HaveOccurred(), "Failed opening GITHUB_OUTPUT file")
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	Expect(err).ToNot(HaveOccurred(), "Failed writing to GITHUB_OUTPUT file")
}
