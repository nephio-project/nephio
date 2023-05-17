IMG_VERSION ?= latest
IMG_REGISTRY ?= docker.io/nephio
IMG ?= $(IMG_REGISTRY)/dnn-fn:${IMG_VERSION}

.PHONY: all
all: fmt test docker-build

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: docker-build
docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${IMG} -f ./Dockerfile .

.PHONY: docker-push
docker-push: ## Build and push docker images to Nephio repo
	docker buildx build --push --tag  ${IMG} -f ./Dockerfile .

include ../../default-all.mk