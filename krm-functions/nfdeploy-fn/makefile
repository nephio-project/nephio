VERSION ?= latest
REGISTRY ?= docker.io/nephio
UPFIMG ?= $(REGISTRY)/upf-deploy-fn:${VERSION}
AMFIMG ?= $(REGISTRY)/amf-deploy-fn:${VERSION}
SMFIMG ?= $(REGISTRY)/smf-deploy-fn:${VERSION}


.PHONY: all
all: test

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: fmt vet ## Run tests.
	go test ./...

upf-docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${UPFIMG} -f ./upfdeployfn/Dockerfile .

upf-docker-push: ## Build docker images.
	docker buildx build --push --tag  ${UPFIMG} -f ./upfdeployfn/Dockerfile .

smf-docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${SMGIMG} -f ./smfdeployfn/Dockerfile .

smf-docker-push: ## Build docker images.
	docker buildx build --push --tag  ${SMGIMG} -f ./smfdeployfn/Dockerfile .

amf-docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${AMFIMG} -f ./amfdeployfn/Dockerfile .

amf-docker-push: ## Build docker images.
	docker buildx build --push --tag  ${AMFIMG} -f ./amfdeployfn/Dockerfile .
