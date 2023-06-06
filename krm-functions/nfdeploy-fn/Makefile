IMAGE_TAG ?= latest
REGISTRY ?= docker.io/nephio
UPFIMG ?= $(REGISTRY)/upf-deploy-fn:${IMAGE_TAG}
AMFIMG ?= $(REGISTRY)/amf-deploy-fn:${IMAGE_TAG}
SMFIMG ?= $(REGISTRY)/smf-deploy-fn:${IMAGE_TAG}

include ../../default-all.mk

.PHONY: all
all: modsync fmt test upf-docker-build smf-docker-build amf-docker-build upf-docker-push smf-docker-push amf-docker-push

.PHONY: modsync
modsync: 
	go mod tidy
	go mod vendor

.PHONY: docker-build
docker-build: modsync fmt test upf-docker-build smf-docker-build amf-docker-build

.PHONY: docker-push
docker-push: upf-docker-push smf-docker-push amf-docker-push

.PHONY: upf-docker-build
upf-docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${UPFIMG} -f ./upfdeployfn/Dockerfile .

.PHONY: upf-docker-push
upf-docker-push: ## Build docker images.
	docker buildx build --push --tag  ${UPFIMG} -f ./upfdeployfn/Dockerfile .

.PHONY: smf-docker-build
smf-docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${SMFIMG} -f ./smfdeployfn/Dockerfile .

.PHONY: smf-docker-push
smf-docker-push: ## Build docker images.
	docker buildx build --push --tag  ${SMFIMG} -f ./smfdeployfn/Dockerfile .

.PHONY: amf-docker-build
amf-docker-build:  ## Build docker images.
	docker buildx build --load --tag  ${AMFIMG} -f ./amfdeployfn/Dockerfile .

.PHONY: amf-docker-push
amf-docker-push: ## Build docker images.
	docker buildx build --push --tag  ${AMFIMG} -f amfdeployfn/Dockerfile .
