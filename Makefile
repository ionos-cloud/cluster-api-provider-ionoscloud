
# Image URL to use all building/pushing image targets
IMG ?= controller:dev
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.5

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

override COVERAGE = coverage.out

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: cover
cover: ## Print the test coverage.
	go tool cover -func=$(COVERAGE)

.PHONY: lint
lint: ## Run lint.
	go run -modfile ./tools/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout 5m -c .golangci.yml

.PHONY: lint-fix
lint-fix: ## Fix linter problems.
	# gci collides with gofumpt. But if we run gci before gofumpt, this will solve the issue.
	go run -modfile ./tools/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout 5m -c .golangci.yml --enable-only gci --fix
	go run -modfile ./tools/go.mod github.com/golangci/golangci-lint/cmd/golangci-lint run --timeout 5m -c .golangci.yml --fix

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: unit-test vet integration-test ## Run tests.

.PHONY: unit-test
unit-test: generate mocks ## Run unit tests.
	CGO_ENABLED=1 go test ./... -v -race -shuffle on -coverprofile $(COVERAGE).tmp -timeout=5m -short
	@grep -vE 'generated|mock' $(COVERAGE).tmp > $(COVERAGE)
	@rm $(COVERAGE).tmp

.PHONY: integration-test
integration-test: manifests api-integration-test ## Run integration tests.

.PHONY: api-integration-test
api-integration-test: envtest ## Run API integration tests.
	cd api && $(call run-ginkgo,v1alpha1)

.PHONY: controller-integration-test
controller-integration-test: envtest ## Run controller integration tests.
	$(call run-ginkgo,controllers)

define run-ginkgo
KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	CGO_ENABLED=1 go run github.com/onsi/ginkgo/v2/ginkgo \
	--randomize-all --randomize-suites \
	--fail-fast --fail-on-pending \
	-v --trace --show-node-events \
	--race \
	$(1)
endef

##@ Build

.PHONY: build
build: manifests generate lint-fix vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate lint-fix vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.1.1
CONTROLLER_TOOLS_VERSION ?= v0.14.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: mocks
mocks:
	go run -modfile ./tools/go.mod github.com/vektra/mockery/v2

# CI

.PHONY: tidy
tidy: ## Run go mod tidy.
	go mod tidy -v && go -C tools mod tidy -v

.PHONY: verify-tidy
verify-tidy: tidy ## Verify that the dependencies are tidied.
	@if !(git diff --quiet HEAD); then \
		echo "run go mod tidy"; PAGER= git diff HEAD; exit 1; \
	fi

.PHONY: verify-gen
verify-gen: manifests generate ## Verify that the generated files are up to date.
	@if !(git diff --quiet HEAD); then \
		echo "generated files are out of date"; PAGER= git diff HEAD; exit 1; \
	fi

.PHONY: verify
verify: verify-gen verify-tidy ## Run all verifications.


##@ Release
## --------------------------------------
## Release
## --------------------------------------

REPOSITORY ?= ghcr.io/ionos-cloud/cluster-api-provider-ionoscloud
RELEASE_DIR ?= out
RELEASE_VERSION ?= v0.0.1

.PHONY: release-manifests
RELEASE_MANIFEST_SOURCE_BASE ?= config/default
release-manifests: $(KUSTOMIZE) ## Create kustomized release manifest in $RELEASE_DIR (defaults to out).
	@mkdir -p $(RELEASE_DIR)
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml
	## change the image tag to the release version
	cd $(RELEASE_MANIFEST_SOURCE_BASE) && $(KUSTOMIZE) edit set image $(REPOSITORY):$(RELEASE_VERSION)
	## generate the release manifest
	$(KUSTOMIZE) build $(RELEASE_MANIFEST_SOURCE_BASE) > $(RELEASE_DIR)/infrastructure-components.yaml

.PHONY: release-templates
release-templates: ## Generate release templates
	@mkdir -p $(RELEASE_DIR)
	cp templates/cluster-template*.yaml $(RELEASE_DIR)/


##@ CRS
## --------------------------------------
## CRS
## --------------------------------------

CALICO_VERSION ?= v3.26.3

.PHONY: crs-calico
crs-calico: ## Generates crs manifests for Calico.
	curl -o templates/crs/cni/calico.yaml https://raw.githubusercontent.com/projectcalico/calico/$(CALICO_VERSION)/manifests/calico.yaml

##@ End-to-End Tests

# Directories.
#
# Full directory of where the Makefile resides
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TEST_DIR := test
ARTIFACTS ?= ${ROOT_DIR}/_artifacts


E2E_CONF_FILE ?= $(ROOT_DIR)/$(TEST_DIR)/e2e/config/ionoscloud.yaml
SKIP_RESOURCE_CLEANUP ?= false
USE_EXISTING_CLUSTER ?= false

GINKGO_POLL_PROGRESS_AFTER ?= 10m
GINKGO_POLL_PROGRESS_INTERVAL ?= 1m
GINKGO_NODES ?= 1
GINKGO_FOCUS ?=
GINKGO_TIMEOUT ?= 2h
GINKGO_NOCOLOR ?= false
GINKGO_LABEL ?= "!Conformance"

GINKGO_SKIP ?=
# To set multiple ginkgo skip flags, if any
ifneq ($(strip $(GINKGO_SKIP)),)
_SKIP_ARGS := $(foreach arg,$(strip $(GINKGO_SKIP)),-skip="$(arg)")
endif

.PHONY: docker-build-e2e
docker-build-e2e: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ghcr.io/ionos-cloud/cluster-api-provider-ionoscloud:e2e .

.PHONY: test-e2e
test-e2e: docker-build-e2e ## Run the end-to-end tests
	CGO_ENABLED=1 go run github.com/onsi/ginkgo/v2/ginkgo -v --trace \
	-poll-progress-after=$(GINKGO_POLL_PROGRESS_AFTER) \
	-poll-progress-interval=$(GINKGO_POLL_PROGRESS_INTERVAL) --tags=e2e --focus="$(GINKGO_FOCUS)" --fail-fast \
	$(_SKIP_ARGS) --nodes=$(GINKGO_NODES) --label-filter=$(GINKGO_LABEL) --timeout=$(GINKGO_TIMEOUT) --no-color=$(GINKGO_NOCOLOR) \
	--output-dir="$(ARTIFACTS)" --junit-report="junit.e2e_suite.1.xml" $(GINKGO_ARGS) $(ROOT_DIR)/$(TEST_DIR)/e2e -- \
	-e2e.artifacts-folder="$(ARTIFACTS)" -e2e.config="$(E2E_CONF_FILE)" \
	-e2e.skip-resource-cleanup=$(SKIP_RESOURCE_CLEANUP) -e2e.use-existing-cluster=$(USE_EXISTING_CLUSTER)

.PHONY: ionosctl
ionosctl: $(LOCALBIN) ## Download the latest release of ionosctl and add to the bin folder
	@if [ ! -f $(LOCALBIN)/ionosctl ]; then \
  		LATEST_RELEASE_TAG=$$(curl -sI https://github.com/ionos-cloud/ionosctl/releases/latest | grep -Fi Location | sed -E 's/.*\/tag\/v(.*)/\1/' | tr -d '\r'); \
  		IONOSCTL_DOWNLOAD_URL="https://github.com/ionos-cloud/ionosctl/releases/download/v$${LATEST_RELEASE_TAG}/ionosctl-$${LATEST_RELEASE_TAG}-linux-amd64.tar.gz"; \
  		curl -sL $$IONOSCTL_DOWNLOAD_URL | tar xz -C $(LOCALBIN) ionosctl; \
	fi

.PHONY: remove-cancelled-e2e-leftovers
remove-cancelled-e2e-leftovers: $(ionosctl) ## Remove any leftover resources from cancelled e2e tests
	$(ROOT_DIR)/hack/scripts/cancelled-workflow.sh
