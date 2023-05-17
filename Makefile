GO_VERSION ?= 1.20.2

# CONTAINER_RUNNABLE checks if tests and lint check can be run inside container.
PODMAN ?= $(shell podman -v > /dev/null 2>&1; echo $$?)
ifeq ($(PODMAN), 0)
CONTAINER_RUNTIME=podman
else
CONTAINER_RUNTIME=docker
endif
CONTAINER_RUNNABLE ?= $(shell $(CONTAINER_RUNTIME) -v > /dev/null 2>&1; echo $$?)

export CONTAINER_RUNTIME CONTAINER_RUNNABLE

GO_MOD_DIRS = $(shell find . -name 'go.mod' -printf "'%h' ")


.PHONY: unit lint gosec unit_clean test
# delegate these commands to the Makefiles next to the go.mod files
unit lint gosec unit_clean test: 
	for dir in $(GO_MOD_DIRS); do \
		$(MAKE) -C "$$dir" $@ ; \
	done

