.PHONY: all
all: test

.PHONY: fmt
fmt: ## Run go fmt against code.
  go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
  go vet ./...

HONY: test
test: fmt vet ## Run tests.
  go test ./... -coverprofile cover.out
