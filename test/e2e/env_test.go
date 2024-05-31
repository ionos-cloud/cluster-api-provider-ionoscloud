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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

// NOTE(gfariasalves-ionos): This file exists to avoid having to use Terraform to create the resources needed for the tests,
// specially when we just need a data center (and for now also a IP Block).
// TODO(gfariasalves-ionos): Remove IP block reservation and deletion after automatic IP Block reservation is working.

const (
	defaultLocation         = "de/txl"
	apiLocationHeaderKey    = "Location"
	apiCallErrWrapper       = "request to Cloud API has failed: %w"
	apiNoLocationErrMessage = "request to Cloud API did not return the request URL"
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
	token := os.Getenv(sdk.IonosTokenEnvVar)
	Expect(token).ToNot(BeEmpty(), "Please set the IONOS_TOKEN environment variable")
	e.api = sdk.NewAPIClient(sdk.NewConfigurationFromEnv())
	Expect(err).ToNot(HaveOccurred(), "Failed to create IONOS Cloud API client")

	By("Creating a datacenter")
	dcName := fmt.Sprintf("%s-%s", "e2e-test", uuid.New().String())
	datacenterID, dcRequest, err := e.createDatacenter(ctx, dcName, defaultLocation, "")
	Expect(err).ToNot(HaveOccurred())
	Expect(e.api.WaitForRequest(ctx, dcRequest)).ToNot(HaveOccurred(), "Failed waiting for datacenter creation")
	e.datacenterID = datacenterID

	By("Reserving an IP block")
	ipbName := fmt.Sprintf("%s-%s", "e2e-test", uuid.New().String())
	ipBlock, ipbRequest, err := e.reserveIPBlock(ctx, ipbName, defaultLocation, 1)
	Expect(err).ToNot(HaveOccurred())
	Expect(e.api.WaitForRequest(ctx, ipbRequest)).ToNot(HaveOccurred(), "Failed waiting for IP block reservation")
	e.ipBlock = ipBlock

}

func (e *ionosCloudEnv) teardown() {
	if !skipCleanup {
		By("Deleting environment resources")
		if e.datacenterID != "" {
			By("Deleting the datacenter")
			deleted, err := e.api.WaitForDeletion(ctx, e.deleteDatacenter(ctx), e.datacenterID)
			Expect(err).ToNot(HaveOccurred(), "Failed asking to delete data center")
			Expect(deleted).To(BeTrue(), fmt.Sprintf("Failed deleting data center %s", e.datacenterID))
		}
		if e.ipBlock != nil {
			By("Deleting the IP Block")
			deleted, err := e.api.WaitForDeletion(ctx, e.deleteIPBlock(ctx), ptr.Deref(e.ipBlock.Id, ""))
			Expect(err).ToNot(HaveOccurred(), "Failed asking to delete IP Block")
			Expect(deleted).To(BeTrue(), fmt.Sprintf("Failed deleting IP block %s", *e.ipBlock.Id))
		}
	}
}

// createDatacenter creates a new data center with the provided name, defaultLocation and description, returning its id and request path.
func (e *ionosCloudEnv) createDatacenter(ctx context.Context, name, location, description string) (datacenterID, requestID string, _ error) {
	if name == "" {
		return "", "", errors.New("name must be set")
	}
	if location == "" {
		return "", "", errors.New("defaultLocation must be set")
	}
	datacenter := sdk.Datacenter{
		Properties: &sdk.DatacenterProperties{
			Name:        &name,
			Location:    &location,
			Description: &description,
		},
	}
	datacenter, req, err := e.api.DataCentersApi.DatacentersPost(ctx).Datacenter(datacenter).Execute()
	if err != nil {
		return "", "", fmt.Errorf("request to Cloud API has failed: %w", err)
	}
	if datacenter.Id == nil {
		return "", "", errors.New("request to Cloud API did not return the data center")
	}
	if requestLocation := req.Header.Get("Location"); requestLocation != "" {
		return *datacenter.Id, requestLocation, nil
	}
	return "", "", errors.New("request to Cloud API did not return the request URL")
}

// deleteDatacenter returns a function that deletes the data center that matches the provided id.
func (e *ionosCloudEnv) deleteDatacenter(ctx context.Context) func(*sdk.APIClient, string) (*sdk.APIResponse, error) {
	return func(_ *sdk.APIClient, datacenterID string) (*sdk.APIResponse, error) {
		if datacenterID == "" {
			return nil, errors.New("data center ID must be set")
		}
		return e.api.DataCentersApi.DatacentersDelete(ctx, datacenterID).Execute()
	}
}

// reserveIPBlock reserves an IP block with the provided properties in the specified defaultLocation, returning the IP
// block and the request path.
func (e *ionosCloudEnv) reserveIPBlock(
	ctx context.Context, name, location string, size int32,
) (_ *sdk.IpBlock, requestID string, _ error) {
	if location == "" {
		return nil, "", errors.New("defaultLocation must be set")
	}
	if size <= 0 {
		return nil, "", errors.New("size must be greater than 0")
	}
	if name == "" {
		return nil, "", errors.New("name must be set")
	}
	newIPBlock := sdk.IpBlock{
		Properties: &sdk.IpBlockProperties{
			Name:     &name,
			Size:     &size,
			Location: &location,
		},
	}
	ipb, req, err := e.api.IPBlocksApi.IpblocksPost(ctx).Ipblock(newIPBlock).Execute()
	if err != nil {
		return nil, "", fmt.Errorf(apiCallErrWrapper, err)
	}
	if requestPath := req.Header.Get(apiLocationHeaderKey); requestPath != "" {
		return &ipb, requestPath, nil
	}
	return nil, "", errLocationHeaderEmpty
}

// deleteIPBlock returns a function that deletes the IP block that matches the provided id.
func (e *ionosCloudEnv) deleteIPBlock(ctx context.Context) func(*sdk.APIClient, string) (*sdk.APIResponse, error) {
	return func(_ *sdk.APIClient, ipBlockID string) (*sdk.APIResponse, error) {
		if ipBlockID == "" {
			return nil, errors.New("IP block ID must be set")
		}
		return e.api.IPBlocksApi.IpblocksDelete(ctx, ipBlockID).Execute()
	}
}
