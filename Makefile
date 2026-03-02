.PHONY: generate build run clean tidy tidy-check lint format check-format check-generate validate-all test help infra.up infra.down

PODMAN ?= podman
GIT_COMMIT ?= $(shell git rev-list -1 HEAD --abbrev-commit)
VERSION ?= "DEV"

BINARY_NAME=dcm-agent
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=./main.go

IMAGE_NAME ?= dcm-agent
IMAGE_TAG ?= latest

NATS_CONTAINER_NAME ?= dcm-nats
NATS_PORT ?= 4222
NATS_SUBJECT ?= work

GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GO_BUILD_FLAGS := ${GO_BUILD_FLAGS}

.EXPORT_ALL_VARIABLES:

help:
	@echo "Targets:"
	@echo "    build:           build the agent binary"
	@echo "    run:             run the agent"
	@echo "    image:           build container image"
	@echo "    clean:           clean up binaries and tools"
	@echo "    generate:        generate code"
	@echo "    check-generate:  check that generated code is up to date"
	@echo "    validate-all:    run all validations (lint, format check, tidy check)"
	@echo "    lint:            run golangci-lint"
	@echo "    format:          format Go code using gofmt and goimports"
	@echo "    check-format:    check that formatting does not introduce changes"
	@echo "    tidy:            tidy go mod"
	@echo "    tidy-check:      check that go.mod and go.sum are tidy"
	@echo "    test:            run tests"
	@echo "    infra.up:        start NATS container (NATS_SUBJECT=work)"
	@echo "    infra.down:      stop NATS container"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-X main.gitCommit=${GIT_COMMIT} -X main.version=${VERSION}" -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Build container image
image:
	@echo "Building container image $(IMAGE_NAME):$(IMAGE_TAG)..."
	$(PODMAN) build --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg VERSION=$(VERSION) -t $(IMAGE_NAME):$(IMAGE_TAG) -f Containerfile .
	@echo "Image built: $(IMAGE_NAME):$(IMAGE_TAG)"

clean:
	@echo "Removing $(BINARY_PATH)..."
	- rm -f $(BINARY_PATH)
	@echo "Clean complete."

run:
	$(BINARY_PATH) run

generate:
	@echo "Generating code..."
	go generate ./...
	@$(MAKE) format
	@echo "Code generation complete."

tidy:
	@echo "Tidying go modules..."
	git ls-files go.mod '**/*go.mod' -z | xargs -0 -I{} bash -xc 'cd $$(dirname {}) && go mod tidy'
	@echo "Go modules tidied successfully."

# Check that go mod tidy does not introduce changes
tidy-check: tidy
	@echo "Checking if go.mod and go.sum are tidy..."
	@git diff --quiet go.mod go.sum || (echo "Detected uncommitted changes after tidy. Run 'make tidy' and commit the result." && git diff go.mod go.sum && exit 1)
	@echo "go.mod and go.sum are tidy."

##################### "make lint" support start ##########################
GOLANGCI_LINT_VERSION := v2.10.1
GOLANGCI_LINT := $(GOBIN)/golangci-lint

.PHONY: check-golangci-lint-version
check-golangci-lint-version:
	@if [ -f '$(GOLANGCI_LINT)' ]; then \
		installed=$$('$(GOLANGCI_LINT)' version 2>/dev/null | sed -n 's/.*version \([0-9.]*\).*/\1/p' | head -1); \
		required=$$(echo '$(GOLANGCI_LINT_VERSION)' | sed 's/^v//'); \
		if [ -n "$$installed" ] && [ "$$installed" != "$$required" ]; then \
			echo "Installed golangci-lint $$installed != required $(GOLANGCI_LINT_VERSION), re-installing..."; \
			rm -f '$(GOLANGCI_LINT)'; \
		fi; \
	fi

$(GOLANGCI_LINT):
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@mkdir -p $(GOBIN)
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
	@echo "'golangci-lint' installed successfully."

lint: check-golangci-lint-version $(GOLANGCI_LINT)
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run --timeout=5m
	@echo "Lint passed successfully!"
##################### "make lint" support end   ##########################

##################### "make format" support start ##########################
GOIMPORTS := $(GOBIN)/goimports

$(GOIMPORTS):
	@echo "Installing goimports..."
	@mkdir -p $(GOBIN)
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "'goimports' installed successfully."

format: $(GOIMPORTS)
	@echo "Formatting Go code..."
	@gofmt -s -w .
	@$(GOIMPORTS) -local github.com/tupyy/dcm-agent -w .
	@echo "Format complete."

check-format: format
	@echo "Checking if formatting is up to date..."
	@git diff --quiet || (echo "Detected uncommitted changes after format. Run 'make format' and commit the result." && git status && exit 1)
	@echo "All formatted files are up to date."
##################### "make format" support end   ##########################

check-generate: generate
	@echo "Checking if generated files are up to date..."
	@git diff --quiet || (echo "Detected uncommitted changes after generate. Run 'make generate' and commit the result." && git status && exit 1)
	@echo "All generated files are up to date."

validate-all: lint check-format tidy-check check-generate

##################### tests support start ##########################
GINKGO := $(GOBIN)/ginkgo
UNIT_TEST_PACKAGES := ./...
UNIT_TEST_GINKGO_OPTIONS ?=

$(GINKGO):
	@echo "Installing ginkgo..."
	@go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.22.0
	@echo "'ginkgo' installed successfully."

test: $(GINKGO)
	@echo "Running tests..."
	@$(GINKGO) -v --show-node-events $(UNIT_TEST_GINKGO_OPTIONS) $(UNIT_TEST_PACKAGES)
	@echo "All tests passed successfully."
##################### tests support end   ##########################

##################### infra support start ##########################
infra.up:
	@echo "Starting NATS container..."
	@$(PODMAN) run -d --name $(NATS_CONTAINER_NAME) -p $(NATS_PORT):4222 nats:latest
	@echo "NATS container started on port $(NATS_PORT)"
	@echo "Subject: $(NATS_SUBJECT)"
	@echo "To publish: nats pub $(NATS_SUBJECT) 'message'"

infra.down:
	@echo "Stopping NATS container..."
	-@$(PODMAN) stop $(NATS_CONTAINER_NAME)
	-@$(PODMAN) rm $(NATS_CONTAINER_NAME)
	@echo "NATS container stopped and removed."
##################### infra support end   ##########################
