############################################################################
# OS/ARCH detection
############################################################################

os1=$(shell uname -s)
os2=
ifeq ($(os1),Darwin)
os1=darwin
os2=osx
else ifeq ($(os1),Linux)
os1=linux
os2=linux
else ifeq (,$(findstring MYSYS_NT-10-0-, $(os1)))
os1=windows
os2=windows
else
$(error unsupported OS: $(os1))
endif

arch1=$(shell uname -m)
ifeq ($(arch1),x86_64)
arch2=amd64
else ifeq ($(arch1),aarch64)
arch2=arm64
else ifeq ($(arch1),arm64)
arch2=arm64
else
$(error unsupported ARCH: $(arch1))
endif

############################################################################
# Vars
############################################################################

DIR := ${CURDIR}
build_dir := $(DIR)/.build/$(os1)-$(arch1)
PLATFORMS ?= linux/amd64,linux/arm64
IMAGE_NAME ?= ghcr.io/clarendonjbbp/casd
TAG ?= latest
go_version := $(shell sed -En 's/^go[ ]+([0-9.]+).*/\1/p' go.mod)
ALPINE_VERSION ?= 3.23

E:=@
ifeq ($(V),1)
	E=
endif

go_version := $(shell sed -En 's/^go[ ]+([0-9.]+).*/\1/p' go.mod)
go_dir := $(build_dir)/go/$(go_version)
ifeq ($(os1),windows)
	go_bin_dir = $(go_dir)/go/bin
	go_url = https://storage.googleapis.com/golang/go$(go_version).$(os1)-$(arch2).zip
	exe=".exe"
else
	go_bin_dir = $(go_dir)/bin
	go_url = https://go.dev/dl/go$(go_version).$(os1)-$(arch2).tar.gz
	exe=
endif
go_path := PATH="$(go_bin_dir):$(PATH)"
go_test_cache := $(DIR)/.build/gocache
go_mod_cache := $(DIR)/.build/gomodcache
golangci_lint_local_cache := $(DIR)/.build/golangci-lint-cache
coverage_profile := $(DIR)/.build/coverage.out
go_run_env := GOTOOLCHAIN=local GOCACHE="$(go_test_cache)" GOMODCACHE="$(go_mod_cache)"

golangci_lint_version = v2.11.4
golangci_lint_dir = $(build_dir)/golangci_lint/$(golangci_lint_version)
golangci_lint_bin = $(golangci_lint_dir)/golangci-lint
golangci_lint_cache = $(golangci_lint_dir)/cache

############################################################################
# Install toolchain
############################################################################

go-check:
ifeq (go$(go_version), $(shell $(go_path) go version 2>/dev/null | cut -f3 -d' '))
else ifeq ($(os1),windows)
	@echo "Installing go$(go_version)..."
	$(E)rm -rf $(dir $(go_dir))
	$(E)mkdir -p $(go_dir)
	$(E)curl -o $(go_dir)\go.zip -sSfL $(go_url)
	$(E)unzip -qq $(go_dir)\go.zip -d $(go_dir)
else
	@echo "Installing go$(go_version)..."
	$(E)rm -rf $(dir $(go_dir))
	$(E)mkdir -p $(go_dir)
	$(E)curl -sSfL $(go_url) | tar xz -C $(go_dir) --strip-components=1
endif

install-toolchain: install-golangci-lint | go-check

install-golangci-lint: $(golangci_lint_bin)

$(golangci_lint_bin): | go-check
	@echo "Installing golangci-lint $(golangci_lint_version)..."
	$(E)rm -rf $(dir $(golangci_lint_dir))
	$(E)mkdir -p $(golangci_lint_dir)
	$(E)mkdir -p $(golangci_lint_cache)
	$(E)GOBIN=$(golangci_lint_dir) $(go_path) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_lint_version)

############################################################################
# Build targets
############################################################################

.PHONY: build clean

build: build-sorter build-web

build-sorter:
	go build -o sorter ./cmd/sorter

build-web:
	go build -o sorter-web ./cmd/web

build-randomizer:
	go build -o randomize ./cmd/randomizer

run: build-web
	$(E)./sorter-web

clean:
	go clean ./cmd/sorter

#############################################################################
# Docker
#############################################################################

.PHONY: container-builder docker-build docker-push

docker_buildx_args = \
	--platform $(PLATFORMS) \
	--build-arg go_version=$(go_version) \
	--build-arg alpine_version=$(ALPINE_VERSION) \
	--tag "${IMAGE_NAME}:${TAG}"

container-builder:
	$(E)docker buildx create --platform $(PLATFORMS) --name container-builder --driver docker-container  --node container-builder0  --use

docker-build: container-builder
	$(E)echo "Validating multi-architecture image build..."
	docker buildx build $(docker_buildx_args) .

docker-push: container-builder
	$(E)echo "Building and pushing multi-architecture images..."
	docker buildx build $(docker_buildx_args) --push .
	

#############################################################################
# Code cleanliness
#############################################################################

.PHONY: fmt test test-coverage lint lint-code lint-fix
fmt: ## Run go fmt against code.
	go fmt ./...

test:
	mkdir -p $(go_test_cache) $(go_mod_cache)
	$(go_run_env) go test ./...

test-coverage:
	mkdir -p $(go_test_cache) $(go_mod_cache) $(DIR)/.build
	$(go_run_env) go test -coverprofile="$(coverage_profile)" ./...
	go tool cover -html="$(coverage_profile)"

lint: lint-code

lint-code: $(golangci_lint_bin) | go-check
	$(E)mkdir -p $(go_test_cache) $(go_mod_cache) $(golangci_lint_local_cache)
	$(E)PATH="$(PATH):$(go_bin_dir)" GOLANGCI_LINT_CACHE="$(golangci_lint_local_cache)" $(go_run_env) $(golangci_lint_bin) run

lint-fix: $(golangci_lint_bin) | go-check
	$(E)mkdir -p $(go_test_cache) $(go_mod_cache) $(golangci_lint_local_cache)
	$(E)PATH="$(PATH):$(go_bin_dir)" GOLANGCI_LINT_CACHE="$(golangci_lint_local_cache)" $(go_run_env) $(golangci_lint_bin) run --fix
