VERSION ?= latest
REGISTRY ?= docker.io/nephio
UPFIMG ?= $(REGISTRY)/upf-deploy-fn:${VERSION}
AMFIMG ?= $(REGISTRY)/amf-deploy-fn:${VERSION}
SMFIMG ?= $(REGISTRY)/smf-deploy-fn:${VERSION}

include ../../default-go-test.mk
include ../../default-go-lint.mk
include ../../default-gosec.mk
include ../../default-go-misc.mk

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
