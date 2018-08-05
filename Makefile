Uname := $(shell uname -s)
uname := $(shell uname -s | tr '[:upper:]' '[:lower:]')

ifeq ($(Uname),Linux)
else ifeq ($(Uname),Darwin)
else
$(error This Makefile only supports Linux and OSX build agents)
endif

ifneq ($(shell uname -m),x86_64)
$(error This Makefile only supports x86_64 build agents)
endif

verify: lint test
lint: lint-vendor lint-proto lint-go
clean: clean-vendor clean-build

.PHONY: test
test: vendor
	go test -race -cover ./...

.PHONY: clean-build
clean-build:
	rm -rf build

# -------------------------------
# Lint Go code with GolangCI-Lint
# -------------------------------

golangci_lint_version := 1.9.3
golangci_lint_dir := build/golangci-lint/$(golangci_lint_version)
golangci_lint := $(golangci_lint_dir)/golangci-lint

$(golangci_lint):
	mkdir -p $(golangci_lint_dir)
	curl -s -L https://github.com/golangci/golangci-lint/releases/download/v$(golangci_lint_version)/golangci-lint-$(golangci_lint_version)-$(uname)-amd64.tar.gz -o $(golangci_lint_dir)/archive.tar.gz
	tar xzf $(golangci_lint_dir)/archive.tar.gz -C $(golangci_lint_dir) --strip 1

.PHONY: lint-go
lint-go: $(golangci_lint) vendor
	$(golangci_lint) run --enable-all ./...

# -------------------------------------
# Download vendor dependencies with dep
# -------------------------------------

dep_version := 0.5.0
dep_dir := build/dep/$(dep_version)
dep := $(dep_dir)/dep

$(dep):
	mkdir -p $(dep_dir)
	curl -s -L https://github.com/golang/dep/releases/download/v$(dep_version)/dep-$(uname)-amd64 -o $(dep_dir)/dep
	chmod +x $(dep_dir)/dep

vendor: $(dep) Gopkg.toml Gopkg.lock
	$(dep) ensure -v

lint-vendor: vendor
	$(dep) check

.PHONY: clean-vendor
clean-vendor:
	rm -rf vendor

# ----------------------------------
# Generate gRPC stubs with prototool
# ----------------------------------

prototool_version := 0.6.0
prototool_dir := build/prototool/$(prototool_version)
prototool := $(prototool_dir)/bin/prototool

$(GOPATH)/bin/protoc-gen-go:
	go get github.com/golang/protobuf/protoc-gen-go

$(prototool):
	mkdir -p $(prototool_dir)
	curl -s -L https://github.com/uber/prototool/releases/download/v$(prototool_version)/prototool-$(Uname)-x86_64.tar.gz -o $(prototool_dir)/archive.tar.gz
	tar xzf $(prototool_dir)/archive.tar.gz -C $(prototool_dir) --strip 1
	chmod +x $(prototool)

.PHONY: proto
proto: $(prototool) $(GOPATH)/bin/protoc-gen-go
	$(prototool) gen

.PHONY: lint-proto
lint-proto: $(prototool) proto
	$(prototool) lint

.PHONY: clean-proto
clean-proto:
	find . -name '*.pb.go' -exec rm {} \+
