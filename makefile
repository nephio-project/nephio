GO_VERSION ?= 1.20.2
GOLANG_CI_VER ?= v1.52
GOSEC_VER ?= 2.15.0
TEST_COVERAGE_FILE=lcov.info
TEST_COVERAGE_HTML_FILE=coverage_unit.html
TEST_COVERAGE_FUNC_FILE=func_coverage.out

# CONTAINER_RUNNABLE checks if tests and lint check can be run inside container.
PODMAN ?= $(shell podman -v > /dev/null 2>&1; echo $$?)
ifeq ($(PODMAN), 0)
CONTAINER_RUNTIME=podman
else
CONTAINER_RUNTIME=docker
endif
CONTAINER_RUNNABLE ?= $(shell $(CONTAINER_RUNTIME) -v > /dev/null 2>&1; echo $$?)

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
         /bin/sh -c "go test ./... -v -coverprofile ${TEST_COVERAGE_FILE}; \
         go tool cover -html=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_HTML_FILE}; \
         go tool cover -func=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_FUNC_FILE}"
else
		go test ./... -v -coverprofile ${TEST_COVERAGE_FILE}
		go tool cover -html=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_HTML_FILE}
		go tool cover -func=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_FUNC_FILE}
endif

# Install link at https://golangci-lint.run/usage/install/ if not running inside a container
.PHONY: lint
lint: ## Run lint  against code.
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/golangci/golangci-lint:${GOLANG_CI_VER}-alpine golangci-lint run ./... -v
else
		golangci-lint run ./... -v
endif

# Install link at https://github.com/securego/gosec#install if not running inside a container
.PHONY: gosec
gosec: ## inspects source code for security problem by scanning the Go Abstract Syntax Tree
ifeq ($(CONTAINER_RUNNABLE), 0)
		$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/securego/gosec:${GOSEC_VER} ./...
else
		gosec ./...
endif