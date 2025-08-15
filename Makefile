default: help

help:
	@echo "Please use 'make <target>' where <target> is one of"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z\._-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.PHONY: help

tidy: ## Run go mod tidy in all directories
	go mod tidy
.PHONY: tidy

build: ## Build
	go build .
.PHONY: tidy

t: test
test: ## Run unit tests, alias: t
	go test --cover -timeout=300s -parallel=16 ./...
.PHONY: t test

fmt: format-code
format-code: tidy ## Format go code and run the fixer, alias: fmt
	golangci-lint fmt
	buf format -w
.PHONY: fmt format-code

lint:
	golangci-lint run --fix ./...
	buf lint

tools:: ## install tools needed to build KachING
	@go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go install github.com/bufbuild/buf/cmd/buf@v1.56.0

.PHONY: tools

