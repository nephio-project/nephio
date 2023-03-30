.PHONY: all
 all: test

 fmt: ## Run go fmt against code.
 	go fmt ./...

 vet: ## Run go vet against code.
 	go vet ./...

 test: fmt vet ## Run tests.
