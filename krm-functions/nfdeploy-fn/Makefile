DOCKERFILE_DIRS = $(shell find . -iname 'Dockerfile' -printf "'%h' " )

# This includes the following targets:
#   test, unit, unit-clean,
#   gosec, lint,
#   fmt, vet
include ../../default-go.mk

.PHONY: all
all: fmt test docker-build docker-push



.PHONY: docker-build docker-push
docker-build: fmt test
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
