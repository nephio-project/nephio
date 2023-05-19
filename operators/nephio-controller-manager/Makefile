VERSION ?= latest
REGISTRY ?= docker.io/nephio
PROJECT ?= nephio

KPT_BLUEPRINT_CFG_DIR ?= blueprint/fn-config
KPT_BLUEPRINT_PKG_DIR ?= blueprint/${PROJECT}


# Image URL to use all building/pushing image targets
IMG ?= $(REGISTRY)/${PROJECT}-operator:$(VERSION)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
include ../../default-go-test.mk
include ../../default-go-lint.mk
include ../../default-gosec.mk
include ../../default-go-misc.mk
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
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
manifests: controller-gen kpt kptgen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	mkdir -p ${KPT_BLUEPRINT_CFG_DIR}
	mkdir -p ${KPT_BLUEPRINT_PKG_DIR}/crd/bases
	mkdir -p ${KPT_BLUEPRINT_PKG_DIR}/app
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=${KPT_BLUEPRINT_PKG_DIR}/crd/bases
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(KPT) pkg init ${KPT_BLUEPRINT_PKG_DIR} --description "${PROJECT} controller"
	$(KPT) pkg init ${KPT_BLUEPRINT_PKG_DIR}/crd --description "${PROJECT} crd"
	$(KPT) pkg init ${KPT_BLUEPRINT_PKG_DIR}/app --description "${PROJECT} app"
	$(KPTGEN) apply config ${KPT_BLUEPRINT_PKG_DIR} --fn-config-dir ${KPT_BLUEPRINT_CFG_DIR}
	rm ${KPT_BLUEPRINT_PKG_DIR}/package-context.yaml
	rm ${KPT_BLUEPRINT_PKG_DIR}/crd/package-context.yaml
	rm ${KPT_BLUEPRINT_PKG_DIR}/app/package-context.yaml

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: unit_clean
unit_clean: ## clean up the unit test artifacts created
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) system prune -f
endif
		rm ${TEST_COVERAGE_FILE} ${TEST_COVERAGE_HTML_FILE} ${TEST_COVERAGE_FUNC_FILE} > /dev/null 2>&1

.PHONY: unit
unit: ## Run unit tests against code.
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/library/golang:${GO_VERSION}-alpine3.17 \
         go test ./... -v -coverprofile ${TEST_COVERAGE_FILE}; \
         go tool cover -html=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_HTML_FILE}; \
         go tool cover -func=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_FUNC_FILE}
else
		go test ./... -v -coverprofile ${TEST_COVERAGE_FILE}
		go tool cover -html=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_HTML_FILE} 
		go tool cover -func=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_FUNC_FILE}
endif

# Install link at https://golangci-lint.run/usage/install/ if not running inside a container
.PHONY: lint
lint: ## Run lint  against code.
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/golangci/golangci-lint:${GOLANG_CI_VER}-alpine golangci-lint run ./... -v --timeout 10m
else
		golangci-lint run ./... -v --timeout 10m
endif

# Install link at https://github.com/securego/gosec#install if not running inside a container
.PHONY: gosec
gosec: ## inspects source code for security problem by scanning the Go Abstract Syntax Tree
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/securego/gosec:${GOSEC_VER} ./...
else
		gosec ./...
endif

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
KPT ?= $(LOCALBIN)/kpt
KPTGEN ?= $(LOCALBIN)/kptgen

## Tool Versions
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.25.0
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.2
KPT_VERSION ?= main
KPTGEN_VERSION ?= v0.0.9

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: kpt
kpt: $(KPT) ## Download kpt locally if necessary.
$(KPT): $(LOCALBIN)
	test -s $(LOCALBIN)/kpt || GOBIN=$(LOCALBIN) go install -v github.com/GoogleContainerTools/kpt@$(KPT_VERSION)

.PHONY: kptgen
kptgen: $(KPTGEN) ## Download kptgen locally if necessary.
$(KPTGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/kptgen || GOBIN=$(LOCALBIN) go install -v github.com/henderiw-kpt/kptgen@$(KPTGEN_VERSION)
