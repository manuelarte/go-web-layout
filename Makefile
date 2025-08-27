default: help

APP_VERSION := "LOCAL"
BRANCH := $(shell git branch --show)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
COMMIT_ID := $(shell git rev-list -1 HEAD)

help:
	@echo "Please use 'make <target>' where <target> is one of"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z\._-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.PHONY: help

dr: docker-run
docker-run:
	docker build \
		--build-arg="BRANCH=$(BRANCH)" \
		--build-arg="BUILD_TIME=$(BUILD_TIME)" \
		--build-arg="COMMIT_ID=$(COMMIT_ID)" \
		--tag github.com/manuelarte/go-web-layout .
	docker run --publish 3001:3001 github.com/manuelarte/go-web-layout

tidy: ## Run go mod tidy in all directories
	go mod tidy
.PHONY: tidy

build: ## Build
	go build -ldflags=" \
	-X github.com/manuelarte/go-web-layout/internal/info.Branch=$(BRANCH) \
	-X github.com/manuelarte/go-web-layout/internal/info.BuildTime=$(BUILD_TIME) \
	-X github.com/manuelarte/go-web-layout/internal/info.CommitID=$(COMMIT_ID) \
	-X github.com/manuelarte/go-web-layout/internal/info.Version=$(APP_VERSION)" \
	./cmd/go-web-layout/.
.PHONY: build

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
	spectral lint .\resources\openapi.yml
	buf lint

tools:: ## install tools needed to build KachING
	@go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@go install go.uber.org/mock/mockgen@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/bufbuild/buf/cmd/buf@v1.56.0

.PHONY: tools

