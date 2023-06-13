DOCKERFILE_DIRS = $(shell find . -iname 'Dockerfile' -printf "'%h' " )

include ../../default-all.mk

.PHONY: all
all: modsync fmt test docker-build docker-push

.PHONY: modsync
modsync: 
	go mod tidy
	go mod vendor


.PHONY: docker-build docker-push
docker-build: modsync fmt test
	for dir in $(DOCKERFILE_DIRS); do \
		$(MAKE) -C "$$dir" $@  ; \
	done

docker-push: 
	for dir in $(DOCKERFILE_DIRS); do \
		$(MAKE) -C "$$dir" $@  ; \
	done

.PHONY: upf-docker-build
upf-docker-build:  ## Build docker images.
	$(MAKE) -C upfdeployfn docker-build

.PHONY: upf-docker-push
upf-docker-push: ## Build docker images.
	$(MAKE) -C upfdeployfn docker-push

.PHONY: smf-docker-build
smf-docker-build:  ## Build docker images.
	$(MAKE) -C smfdeployfn docker-build

.PHONY: smf-docker-push
smf-docker-push: ## Build docker images.
	$(MAKE) -C smfdeployfn docker-push

.PHONY: amf-docker-build
amf-docker-build:  ## Build docker images.
	$(MAKE) -C amfdeployfn docker-build

.PHONY: amf-docker-push
amf-docker-push: ## Build docker images.
	$(MAKE) -C amfdeployfn docker-push
