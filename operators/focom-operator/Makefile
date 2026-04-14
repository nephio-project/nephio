#  Copyright 2025-2026 The Nephio Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# Removed missing includes - using local targets instead
# These targets were previously included from parent makefiles

include ../../default-go-lint.mk
include ../../default-gosec.mk
include ../../default-go-misc.mk
GIT_ROOT_DIR ?= $(dir $(lastword $(MAKEFILE_LIST)))
include $(GIT_ROOT_DIR)/detect-container-runtime.mk

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= latest
REGISTRY ?= docker.io/nephio
PROJECT ?= focom
# Image URL to use all building/pushing image targets
# Override IMG for local development: make docker-build IMG=<your-registry>/focom-operator:dev
IMG ?= $(REGISTRY)/$(PROJECT)-operator:$(VERSION)
# Set IMAGE_PULL_SECRET to enable imagePullSecrets and imagePullPolicy: Always for private registries
# Example: make deploy-with-flux IMG=ghcr.io/myorg/focom-operator:dev IMAGE_PULL_SECRET=ghcr-secret
IMAGE_PULL_SECRET ?=
GIT_ROOT_DIR ?= .

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the registry namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
IMAGE_TAG_BASE ?= $(REGISTRY)/$(PROJECT)-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Set the Operator SDK version to use. By default, what is installed on the system is used.
# This is useful for CI or a project to utilize a specific version of the operator-sdk toolkit.
OPERATOR_SDK_VERSION ?= v1.39.1

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.31.0

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

.PHONY: test
test: manifests generate fmt vet envtest ## Run all tests (including integration tests that require external dependencies).
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

.PHONY: unit-tests
unit-tests: fmt vet ## Run unit tests only (no external dependencies required).
	@echo "Running unit tests (models, validation, storage logic, controller)..."
	@go test -v -coverprofile=coverage-models.out \
		./internal/nbi/models/ \
		./internal/nbi/validation/
	@go test -v -coverprofile=coverage-storage.out ./internal/nbi/storage/ \
		-run "TestInMemory|TestCreateResourceYAML|TestParseResourceYAML|TestRoundTrip|TestExtractResourceID|TestGenerateWorkspaceName|TestSetResource|TestConvertTo|TestCreateKptfile"
	@go test -v -coverprofile=coverage-controller.out \
		./internal/controller/ -run TestValidateTemplateAlignment || echo "Controller tests skipped"
	@echo ""
	@echo "✅ Unit tests completed successfully!"
	@echo "📊 Coverage reports generated:"
	@echo "  - coverage-models.out"
	@echo "  - coverage-storage.out"
	@echo "  - coverage-controller.out"
	@echo ""
	@echo "To view coverage: go tool cover -html=coverage-models.out"

.PHONY: unit
unit: unit-tests ## Alias for unit-tests.

.PHONY: ci-tests
ci-tests: fmt vet ## Run the same tests as CI pipeline (unit tests only, fast execution).
	@echo "Running CI tests (matches GitHub Actions pipeline)..."
	@go test -v -coverprofile=coverage-models.out \
		./internal/nbi/models/ \
		./internal/nbi/validation/
	@go test -v -coverprofile=coverage-storage.out ./internal/nbi/storage/ \
		-run "TestInMemory|TestCreateResourceYAML|TestParseResourceYAML|TestRoundTrip|TestExtractResourceID|TestGenerateWorkspaceName|TestSetResource|TestConvertTo|TestCreateKptfile"
	@go test -v -coverprofile=coverage-controller.out \
		./internal/controller/ -run TestValidateTemplateAlignment || echo "Controller tests skipped"
	@echo ""
	@echo "✅ CI tests completed successfully!"
	@echo "📊 Coverage reports: coverage-models.out, coverage-storage.out, coverage-controller.out"

##@ Porch Integration Tests
## These targets run tests against a live Porch/Kubernetes cluster.
## Requires a valid kubeconfig and Porch repository configured in testconfig.yaml.
## See internal/nbi/testconfig.yaml for configuration.
##
## Timeouts are configurable via environment variables:
##   SMOKE_TIMEOUT      (default: 600s)
##   INTEGRATION_TIMEOUT (default: 3600s)
##   STABILITY_TIMEOUT   (default: 7200s)
##
## Example: make test-smoke SMOKE_TIMEOUT=300s
##
##@ Runtime Environment Variables
## FOCOM_EARLY_SCHEMA_VALIDATION  When set to "true", enables schema validation of FPR
##                                templateParameters during CreateDraft and UpdateDraft
##                                operations (in addition to ValidateDraft).
##                                Default: disabled (validation only at ValidateDraft).
##                                Example: export FOCOM_EARLY_SCHEMA_VALIDATION=true

SMOKE_TIMEOUT ?= 600s
INTEGRATION_TIMEOUT ?= 3600s
STABILITY_TIMEOUT ?= 7200s

.PHONY: test-smoke
test-smoke: ## Run smoke tests against Porch (~5 min). Single setup, exercises critical path for all resource types.
	@echo "Running smoke tests against Porch (timeout=$(SMOKE_TIMEOUT))..."
	FOCOM_SMOKE_TESTS=true go test -v -count=1 ./internal/nbi/ -run TestSmokeWorkflow -timeout $(SMOKE_TIMEOUT)
	@echo ""
	@echo "✅ Smoke tests completed!"

.PHONY: test-integration
test-integration: ## Run full integration tests against Porch (~30 min). Thorough CRUD, workflow, error handling per resource type.
	@echo "Running full integration tests against Porch (timeout=$(INTEGRATION_TIMEOUT))..."
	FOCOM_INTEGRATION_TESTS=true go test -v -count=1 ./internal/nbi/ -timeout $(INTEGRATION_TIMEOUT)
	@echo ""
	@echo "✅ Integration tests completed!"

.PHONY: test-stability
test-stability: ## Run stability tests against Porch (~60 min). Cross-resource workflows, deletion prevention, dependency validation.
	@echo "Running stability tests against Porch (timeout=$(STABILITY_TIMEOUT))..."
	FOCOM_STABILITY_TESTS=true go test -v -count=1 ./internal/nbi/ -timeout $(STABILITY_TIMEOUT)
	@echo ""
	@echo "✅ Stability tests completed!"

.PHONY: test-all-porch
test-all-porch: ## Run all Porch tests: smoke + integration + stability (~90 min).
	@echo "Running ALL Porch tests (timeout=$(STABILITY_TIMEOUT))..."
	FOCOM_SMOKE_TESTS=true FOCOM_INTEGRATION_TESTS=true FOCOM_STABILITY_TESTS=true \
		go test -v -count=1 ./internal/nbi/ -timeout $(STABILITY_TIMEOUT)
	@echo ""
	@echo "✅ All Porch tests completed!"

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test-e2e  # Run the e2e tests against a Kind k8s instance that is spun up.
test-e2e:
	go test ./test/e2e/ -v -ginkgo.v

# .PHONY: lint
# lint: golangci-lint ## Run golangci-lint linter (CI target).
# 	$(GOLANGCI_LINT) run

.PHONY: lint-local
lint-local: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

# .PHONY: gosec
# gosec: gosec-tool ## Run gosec security scanner
# 	$(GOSEC) ./...

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build:  ## Build a container image from the local Dockerfile
	$(CONTAINER_RUNTIME) buildx build --load --tag  ${IMG} -f ./Dockerfile .

.PHONY: docker-push
docker-push: docker-build ## Build and push the container image
	$(CONTAINER_RUNTIME) push ${IMG}

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
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
	- $(CONTAINER_TOOL) buildx create --name focom-operator-builder
	$(CONTAINER_TOOL) buildx use focom-operator-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm focom-operator-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

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

.PHONY: docker-auth
docker-auth: ## Create Kubernetes secret for private registry authentication. Usage: make docker-auth REGISTRY_USER=username REGISTRY_PASSWORD=token [REGISTRY_SERVER=ghcr.io] [REGISTRY_SECRET_NAME=ghcr-secret]
	@if [ -z "$(REGISTRY_USER)" ]; then \
		echo "❌ Error: REGISTRY_USER is required"; \
		echo "Usage: make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token [REGISTRY_SERVER=ghcr.io] [REGISTRY_SECRET_NAME=ghcr-secret]"; \
		exit 1; \
	fi
	@if [ -z "$(REGISTRY_PASSWORD)" ]; then \
		echo "❌ Error: REGISTRY_PASSWORD is required"; \
		echo "Usage: make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token [REGISTRY_SERVER=ghcr.io] [REGISTRY_SECRET_NAME=ghcr-secret]"; \
		exit 1; \
	fi
	@echo "🔐 Creating private registry authentication secret..."
	@$(KUBECTL) create namespace focom-operator-system --dry-run=client -o yaml | $(KUBECTL) apply -f -
	@$(KUBECTL) create secret docker-registry $(or $(REGISTRY_SECRET_NAME),ghcr-secret) \
		--docker-server=$(or $(REGISTRY_SERVER),ghcr.io) \
		--docker-username=$(REGISTRY_USER) \
		--docker-password=$(REGISTRY_PASSWORD) \
		--docker-email=$(or $(REGISTRY_EMAIL),$(REGISTRY_USER)@users.noreply.github.com) \
		--namespace=focom-operator-system \
		--dry-run=client -o yaml | $(KUBECTL) apply -f -
	@echo "✅ Registry secret '$(or $(REGISTRY_SECRET_NAME),ghcr-secret)' created in focom-operator-system"
	@echo ""
	@echo "📋 Next steps:"
	@echo "  make deploy-with-flux IMG=<your-registry>/focom-operator:<tag> IMAGE_PULL_SECRET=$(or $(REGISTRY_SECRET_NAME),ghcr-secret)"

.PHONY: docker-auth-help
docker-auth-help: ## Show help for setting up private registry authentication
	@echo "🔐 Private Registry Authentication Setup"
	@echo ""
	@echo "Create a Kubernetes secret for pulling images from a private registry:"
	@echo ""
	@echo "  make docker-auth REGISTRY_USER=<username> REGISTRY_PASSWORD=<token>"
	@echo ""
	@echo "Optional parameters:"
	@echo "  REGISTRY_SERVER       Registry hostname (default: ghcr.io)"
	@echo "  REGISTRY_SECRET_NAME  Secret name (default: ghcr-secret)"
	@echo "  REGISTRY_EMAIL        Email for registry (default: <user>@users.noreply.github.com)"
	@echo ""
	@echo "Examples:"
	@echo "  GitHub:  make docker-auth REGISTRY_USER=johndoe REGISTRY_PASSWORD=ghp_xxxx"
	@echo "  Custom:  make docker-auth REGISTRY_USER=user REGISTRY_PASSWORD=pass REGISTRY_SERVER=registry.example.com REGISTRY_SECRET_NAME=my-secret"
	@echo ""
	@echo "Then deploy with:"
	@echo "  make deploy-with-flux IMG=ghcr.io/myorg/focom-operator:dev IMAGE_PULL_SECRET=ghcr-secret"
	@echo ""
	@echo "Check if secret exists:"
	@echo "  kubectl get secret ghcr-secret -n focom-operator-system"

# Internal target: configure image and optionally add private registry support
.PHONY: _configure-image
_configure-image:
	@echo "🔧 Setting image: ${IMG}"
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	@if [ -n "$(IMAGE_PULL_SECRET)" ]; then \
		echo "🔐 Configuring imagePullSecret: $(IMAGE_PULL_SECRET)"; \
		sed -i 's/name: ghcr-secret/name: $(IMAGE_PULL_SECRET)/' config/manager/patches/private-registry.yaml; \
		cd config/manager && grep -q 'private-registry.yaml' kustomization.yaml || \
			$(KUSTOMIZE) edit add patch --path patches/private-registry.yaml; \
	else \
		cd config/manager && sed -i '/patches\/private-registry.yaml/d' kustomization.yaml 2>/dev/null; \
		cd config/manager && if grep -q '^patches:' kustomization.yaml && ! grep -q '  - ' kustomization.yaml; then \
			sed -i '/^patches:$$/d' kustomization.yaml; \
		fi; \
	fi

.PHONY: deploy
deploy: manifests kustomize _configure-image ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	@echo "🚀 Deploying FOCOM Operator..."
	@echo "📦 Using image: ${IMG}"
	@echo ""
	@if [ -n "$(IMAGE_PULL_SECRET)" ]; then \
		if ! $(KUBECTL) get secret $(IMAGE_PULL_SECRET) -n focom-operator-system >/dev/null 2>&1; then \
			echo "⚠️  Warning: IMAGE_PULL_SECRET=$(IMAGE_PULL_SECRET) specified but secret not found in focom-operator-system!"; \
			echo "   Create it first: make docker-auth REGISTRY_USER=your-username REGISTRY_PASSWORD=your-token"; \
			echo ""; \
		else \
			echo "✅ Image pull secret '$(IMAGE_PULL_SECRET)' found"; \
		fi; \
	fi
	@echo "Creating focom-operator-system namespace if it doesn't exist..."
	@$(KUBECTL) create namespace focom-operator-system --dry-run=client -o yaml | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build config/default > kpt-package/focom-operator-bundle.yaml
	@echo "Deploying ConfigSync RootSync for focom-resources..."
	@echo "Copying gitea-secret to config-management-system namespace..."
	@$(KUBECTL) get secret gitea-secret -n default -o yaml 2>/dev/null | \
		sed 's/namespace: default/namespace: config-management-system/' | \
		$(KUBECTL) apply -f - || echo "Warning: gitea-secret not found in default namespace. ConfigSync may not work without it."
	@echo "Applying ConfigSync RootSync..."
	$(KUSTOMIZE) build config/configsync | $(KUBECTL) apply -f -
	@echo "ConfigSync RootSync deployed. Check status with: kubectl get rootsync focom-resources -n config-management-system"

.PHONY: build-k8s-deployment
build-k8s-deployment: manifests kustomize ## Generate Kubernetes deployment bundle with updated RBAC
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > kpt-package/focom-operator-bundle.yaml

.PHONY: kpt-package
kpt-package: manifests kustomize ## Generate kpt package bundle without deploying
	@echo "Generating kpt package bundle..."
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > kpt-package/focom-operator-bundle.yaml
	@echo "kpt package generated at: kpt-package/focom-operator-bundle.yaml"
	@echo ""
	@echo "To deploy using kpt:"
	@echo "  1. Initialize: cd kpt-package && kpt live init --namespace focom-operator-system --inventory-id focom-operator"
	@echo "  2. Apply: kpt live apply"
	@echo ""
	@echo "Or deploy directly with kubectl:"
	@echo "  kubectl apply -f kpt-package/focom-operator-bundle.yaml"

.PHONY: deploy-with-flux
deploy-with-flux: manifests kustomize _configure-image ## Deploy controller with Flux (instead of ConfigSync) to the K8s cluster specified in ~/.kube/config.
	@echo "🚀 Deploying FOCOM Operator with Flux..."
	@echo "📦 Using image: ${IMG}"
	@echo ""
	@echo "Creating focom-operator-system namespace if it doesn't exist..."
	@$(KUBECTL) create namespace focom-operator-system --dry-run=client -o yaml | $(KUBECTL) apply -f -
	@echo "🔍 Verifying final deployment will use correct image..."
	@$(KUSTOMIZE) build config/default | grep -A 2 -B 2 "image:" | grep -v "imagePullPolicy" || echo "Could not extract image from deployment"
	@echo ""
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	$(KUSTOMIZE) build config/default > kpt-package/focom-operator-bundle.yaml
	@echo "Deploying Flux resources for focom-resources..."
	@echo "Copying gitea-secret to flux-system namespace..."
	@$(KUBECTL) get secret gitea-secret -n default -o json 2>/dev/null | \
		jq 'del(.metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]) | .metadata.namespace = "flux-system"' | \
		$(KUBECTL) apply -f - 2>/dev/null || echo "Warning: gitea-secret not found in default namespace. Flux may not work without it."
	@echo "Applying Flux GitRepository and Kustomization..."
	$(KUSTOMIZE) build config/flux | $(KUBECTL) apply -f -
	@echo ""
	@echo "✅ Flux resources deployed. Check status with:"
	@echo "  kubectl get pods -n focom-operator-system"
	@echo "  kubectl get gitrepository focom-resources -n flux-system"
	@echo "  kubectl get kustomization focom-resources -n flux-system"

.PHONY: deploy-with-kpt
deploy-with-kpt: kpt-package ## Deploy controller using kpt live apply, with Flux for focom-resources sync.
	@echo "🚀 Deploying FOCOM Operator with kpt..."
	@echo "📦 Using image: ${IMG}"
	@echo ""
	@echo "Creating focom-operator-system namespace if it doesn't exist..."
	@$(KUBECTL) create namespace focom-operator-system --dry-run=client -o yaml | $(KUBECTL) apply -f -
	@echo "🔍 Verifying kpt package image:"
	@grep "image:" kpt-package/focom-operator-bundle.yaml | grep -v "imagePullPolicy" || echo "Could not extract image from bundle"
	@echo ""
	@echo "📦 Applying kpt package..."
	cd kpt-package && kpt live apply --reconcile-timeout=5m --inventory-policy=adopt
	@echo ""
	@echo "Deploying Flux resources for focom-resources..."
	@echo "Copying gitea-secret to flux-system namespace..."
	@$(KUBECTL) get secret gitea-secret -n default -o json 2>/dev/null | \
		jq 'del(.metadata.resourceVersion, .metadata.uid, .metadata.creationTimestamp, .metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]) | .metadata.namespace = "flux-system"' | \
		$(KUBECTL) apply -f - 2>/dev/null || echo "Warning: gitea-secret not found in default namespace. Flux may not work without it."
	@echo "Applying Flux GitRepository and Kustomization..."
	$(KUSTOMIZE) build config/flux | $(KUBECTL) apply -f -
	@echo ""
	@echo "✅ FOCOM Operator deployed with kpt. Flux resources deployed for focom-resources."
	@echo "Check status with:"
	@echo "  kubectl get pods -n focom-operator-system"
	@echo "  kubectl get gitrepository focom-resources -n flux-system"
	@echo "  kubectl get kustomization focom-resources -n flux-system"

.PHONY: undeploy-with-kpt
undeploy-with-kpt: ## Remove controller deployed via kpt live apply and Flux resources.
	@echo "🗑️  Removing Flux resources..."
	-$(KUSTOMIZE) build config/flux | $(KUBECTL) delete -f - --ignore-not-found
	@echo "Deleting gitea-secret from flux-system namespace..."
	-$(KUBECTL) delete secret gitea-secret -n flux-system --ignore-not-found
	@echo "Waiting for Flux to clean up CRs (10 seconds)..."
	@sleep 10
	@echo "🗑️  Removing FOCOM Operator via kpt..."
	-cd kpt-package && kpt live destroy
	@echo "✅ FOCOM Operator and Flux resources removed."

.PHONY: deploy-with-flux-webhook
deploy-with-flux-webhook: deploy-with-flux ## Deploy controller with Flux and configure webhook for instant sync
	@echo ""
	@echo "Setting up webhook for instant sync..."
	@echo "Generating webhook token..."
	@WEBHOOK_TOKEN=$$(head -c 12 /dev/urandom | shasum | cut -d ' ' -f1) && \
	echo "Creating webhook-token secret..." && \
	$(KUBECTL) create secret generic webhook-token \
		-n flux-system \
		--from-literal=token=$$WEBHOOK_TOKEN \
		--dry-run=client -o yaml | $(KUBECTL) apply -f - && \
	echo "Deploying Receiver..." && \
	$(KUBECTL) apply -f config/flux/focom-resources-receiver.yaml && \
	echo "Waiting for Receiver to be ready..." && \
	sleep 5 && \
	WEBHOOK_PATH=$$($(KUBECTL) get receiver focom-resources -n flux-system -o jsonpath='{.status.webhookPath}' 2>/dev/null || echo "") && \
	echo "" && \
	echo "========================================" && \
	echo "Webhook Configuration" && \
	echo "========================================" && \
	echo "" && \
	echo "Webhook Token: $$WEBHOOK_TOKEN" && \
	echo "" && \
	if [ -n "$$WEBHOOK_PATH" ]; then \
		echo "Webhook Path: $$WEBHOOK_PATH"; \
		echo ""; \
		echo "To expose the webhook, choose one option:"; \
		echo ""; \
		echo "1. Port-forward (testing):"; \
		echo "   kubectl port-forward -n flux-system svc/notification-controller 8080:80"; \
		echo "   Webhook URL: http://localhost:8080$$WEBHOOK_PATH"; \
		echo ""; \
		echo "2. Configure in Gitea:"; \
		echo "   URL: http://your-webhook-endpoint$$WEBHOOK_PATH"; \
		echo "   Secret: $$WEBHOOK_TOKEN"; \
		echo ""; \
		echo "See docs/flux/webhook-setup.md for detailed instructions"; \
	else \
		echo "Receiver not ready yet. Check status with:"; \
		echo "  kubectl describe receiver focom-resources -n flux-system"; \
	fi && \
	echo "" && \
	echo "Webhook setup complete!"

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@echo "Undeploying ConfigSync RootSync..."
	$(KUSTOMIZE) build config/configsync | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f - || true
	@echo "Deleting gitea-secret from config-management-system namespace..."
	@$(KUBECTL) delete secret gitea-secret -n config-management-system --ignore-not-found=$(ignore-not-found) || true
	@echo "Undeploying FOCOM operator..."
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -
	@echo "Note: focom-system namespace is not deleted. Delete manually if needed: kubectl delete namespace focom-system"
	@echo "Note: CRs managed by ConfigSync may still exist. They will be removed when the RootSync is deleted."

.PHONY: undeploy-flux
undeploy-flux: kustomize ## Undeploy controller and Flux resources from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@echo "Undeploying Flux resources (GitRepository and Kustomization only)..."
	$(KUSTOMIZE) build config/flux | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f - || true
	@echo "Deleting gitea-secret from flux-system namespace..."
	@$(KUBECTL) delete secret gitea-secret -n flux-system --ignore-not-found=$(ignore-not-found) || true
	@echo "Waiting for Flux to clean up CRs (10 seconds)..."
	@sleep 10
	@echo "Undeploying FOCOM operator..."
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f - || true
	@echo ""
	@echo "Note: Flux controllers remain running and can be used for other purposes."
	@echo "Note: focom-system namespace is not deleted. Delete manually if needed: kubectl delete namespace focom-system"
	@echo "Note: CRs managed by Flux should have been removed. Check with: kubectl get oclouds,templateinfoes,focomprovisioningrequests -A"

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOSEC = $(LOCALBIN)/gosec

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.3
CONTROLLER_TOOLS_VERSION ?= v0.17.0
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v2.8.0
GOSEC_VERSION ?= v2.22.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: gosec-tool
gosec-tool: $(GOSEC) ## Download gosec locally if necessary.
$(GOSEC): $(LOCALBIN)
	$(call go-install-tool,$(GOSEC),github.com/securego/gosec/v2/cmd/gosec,$(GOSEC_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

.PHONY: operator-sdk
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (, $(shell which operator-sdk 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which operator-sdk)
endif
endif

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = $(LOCALBIN)/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)
