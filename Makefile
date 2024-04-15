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
	go_url = https://storage.googleapis.com/golang/go$(go_version).$(os1)-$(arch2).tar.gz
	exe=
endif
go_path := PATH="$(go_bin_dir):$(PATH)"

golangci_lint_version = v1.52.2
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
	$(E)GOBIN=$(golangci_lint_dir) $(go_path) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_lint_version)

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
	./sorter-web	

clean:
	go clean ./cmd/sorter

#############################################################################
# Code cleanliness
#############################################################################

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

lint: lint-code

lint-code: $(golangci_lint_bin) | go-check
	$(E)PATH="$(PATH):$(go_bin_dir)" $(golangci_lint_bin) run ./...
