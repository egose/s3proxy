SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help
.PHONY: help build build-all build-single build-archive check-toolchain test-asdf format fmt vet test test-race cover clean \
        docker-build docker-run run sandbox-up sandbox-down sandbox-destroy \
        sandbox-reset sandbox-logs sandbox-logs-follow sandbox-ps validate \
        test-integration test-integration-race sandbox-integration-up \
        sandbox-integration-down

# --- Project --------------------------------------------------------------

BINARY      := s3proxy
MAIN_PKG    := ./cmd/s3proxy
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LD_FLAGS    := -s -w -X main.version=$(VERSION)
DIST_DIR    := dist
PREFIX      := s3proxy
GO_BUILD_FLAGS := -trimpath -buildvcs=false
TAR_FLAGS   := --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner

# --- Cross-compile matrix -------------------------------------------------

OS_ARCH_PAIRS := \
    linux:amd64 \
    linux:arm64 \
    linux:386 \
    linux:arm \
    windows:amd64 \
    windows:386 \
    darwin:amd64 \
    darwin:arm64 \
    freebsd:amd64 \
    freebsd:arm64 \
    openbsd:amd64 \
    openbsd:arm64 \
    netbsd:amd64

# --- Docker / Compose -----------------------------------------------------

COMPOSE      := docker-compose --env-file .env -f ./sandbox/docker-compose.yml
UP_FLAGS     := up --build --remove-orphans
DOWN_FLAGS   := down
DESTROY_FLAGS := down --volumes --rmi all --remove-orphans
LOGS_FLAGS   := logs --tail=50

DAEMON ?= false
ifeq ($(DAEMON),true)
	UP_FLAGS += -d
endif

# --- Help -----------------------------------------------------------------

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Local Go build -------------------------------------------------------

build: ## Build the s3proxy binary into dist/
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags "$(LD_FLAGS)" -o "$(DIST_DIR)/$(BINARY)" $(MAIN_PKG)
	@echo "built $(DIST_DIR)/$(BINARY) (version $(VERSION))"

build-single: ## Build for a single OS:ARCH pair (OS_ARCH=linux:amd64)
	@set -euo pipefail; \
	OS_ARCH='$(OS_ARCH)'; \
	case "$$OS_ARCH" in *:*:*) echo "invalid OS_ARCH: $$OS_ARCH" >&2; exit 2;; *:*) ;; *) echo "OS_ARCH must be os:arch" >&2; exit 2;; esac; \
	OS="$${OS_ARCH%%:*}"; \
	ARCH="$${OS_ARCH#*:}"; \
	case " $(OS_ARCH_PAIRS) " in *" $$OS_ARCH "*) ;; *) echo "unsupported OS_ARCH: $$OS_ARCH" >&2; exit 2;; esac; \
	echo "Building for OS=$$OS and ARCH=$$ARCH"; \
	DIR="$(DIST_DIR)/$$OS-$$ARCH"; \
	mkdir -p "$$DIR"; \
	EXT=""; [ "$$OS" != "windows" ] || EXT=.exe; \
	tmp="$$(mktemp "$$DIR/.$(BINARY).XXXXXX")"; trap 'rm -f "$$tmp"' EXIT; \
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH \
	  go build $(GO_BUILD_FLAGS) -ldflags "$(LD_FLAGS)" \
	  -o "$$tmp" $(MAIN_PKG); \
	chmod +x "$$tmp"; mv -f "$$tmp" "$$DIR/$(BINARY)$$EXT"; trap - EXIT

build-all: ## Cross-compile for all OS/arch pairs in OS_ARCH_PAIRS
	@set -euo pipefail; \
	stage="$$(mktemp -d "$(CURDIR)/.dist.XXXXXX")"; trap 'rm -rf "$$stage"' EXIT; \
	for pair in $(OS_ARCH_PAIRS); do $(MAKE) --no-print-directory build-single OS_ARCH="$$pair" DIST_DIR="$$stage"; done; \
	rm -rf "$(DIST_DIR)"; mv "$$stage" "$(DIST_DIR)"; trap - EXIT

build-archive: ## Tar each cross-compiled dist/<os>-<arch>/ dir into a release archive
	@set -euo pipefail; \
	rm -f "$(DIST_DIR)"/*.tar.gz "$(DIST_DIR)/SHA256SUMS"; \
	for pair in $(OS_ARCH_PAIRS); do \
	  name="$${pair/:/-}"; d="$(DIST_DIR)/$$name"; \
	  ext=""; [ "$${pair%%:*}" != windows ] || ext=.exe; \
	  test -f "$$d/$(BINARY)$$ext" || { echo "missing $$d/$(BINARY)$$ext" >&2; exit 1; }; \
	  test "$$(find "$$d" -mindepth 1 -maxdepth 1 | wc -l)" -eq 1 || { echo "unexpected files in $$d" >&2; exit 1; }; \
	  archive="$(DIST_DIR)/$(PREFIX)-$$name.tar.gz"; \
	  tmp="$$archive.tmp"; tar $(TAR_FLAGS) -C "$$d" -cf - "$(BINARY)$$ext" | gzip -n >"$$tmp"; \
	  tar -tzf "$$tmp" | diff -u <(printf '%s\n' "$(BINARY)$$ext") - >/dev/null; \
	  mv "$$tmp" "$$archive"; \
	  echo "archived $$archive"; \
	done; \
	(cd "$(DIST_DIR)" && sha256sum $(PREFIX)-*.tar.gz >SHA256SUMS)

# --- Quality --------------------------------------------------------------

format fmt: ## Run gofmt -s on all Go sources
	@gofmt -w -s .

vet: ## Run go vet on all packages
	@go vet ./...

test: ## Run all unit tests
	@go test ./...

test-race: ## Run tests with the race detector
	@go test -race ./...

cover: ## Run tests with coverage report
	@mkdir -p $(DIST_DIR)
	@go test -coverprofile=$(DIST_DIR)/coverage.out ./...
	@go tool cover -func=$(DIST_DIR)/coverage.out | tail -1
	@echo "coverage profile: $(DIST_DIR)/coverage.out"

clean: ## Remove dist/ and coverage artifacts
	@rm -rf $(DIST_DIR)
	@echo "cleaned $(DIST_DIR)"

# --- Run / validate -------------------------------------------------------

run: ## Run the server locally (CONFIG=path/to/config.hcl)
	@if [ -z "$(CONFIG)" ]; then echo "usage: make run CONFIG=path/to/config.hcl"; exit 1; fi
	@config="$$(realpath -- "$(CONFIG)")" && go run $(MAIN_PKG) serve --config "$$config"

validate: ## Validate config without starting the server (CONFIG=path/to/config.hcl)
	@if [ -z "$(CONFIG)" ]; then echo "usage: make validate CONFIG=path/to/config.hcl"; exit 1; fi
	@config="$$(realpath -- "$(CONFIG)")" && go run $(MAIN_PKG) validate --config "$$config"

check-toolchain: ## Verify tool declarations and build inputs remain aligned
	@set -euo pipefail; \
	go_mod="$$(awk '$$1 == "go" { print $$2; exit }' go.mod)"; \
	go_tool="$$(awk '$$1 == "golang" { print $$2; exit }' .tool-versions)"; \
	pnpm_tool="$$(awk '$$1 == "pnpm" { print $$2; exit }' .tool-versions)"; \
	pnpm_pkg="$$(node -p "require('./package.json').packageManager.split('@')[1]")"; \
	docker_go="$$(awk -F'[:@]' '/^FROM golang:/ { print $$2; exit }' Dockerfile)"; \
	test "$${go_tool%.*}" = "$$go_mod" || { echo "Go drift: go.mod=$$go_mod .tool-versions=$$go_tool" >&2; exit 1; }; \
	test "$$go_tool" = "$$docker_go" || { echo "Go drift: .tool-versions=$$go_tool Dockerfile=$$docker_go" >&2; exit 1; }; \
	test "$$pnpm_tool" = "$$pnpm_pkg" || { echo "pnpm drift: .tool-versions=$$pnpm_tool package.json=$$pnpm_pkg" >&2; exit 1; }; \
	for tool in actionlint shellcheck; do grep -Eq "^$$tool [^ ]+$$" .tool-versions || { echo "missing $$tool version" >&2; exit 1; }; done; \
	grep -Eq '^FROM golang:[^ ]+@sha256:[0-9a-f]{64} AS builder$$' Dockerfile || { echo "Docker builder base is not digest-pinned" >&2; exit 1; }; \
	grep -Eq '^FROM [^ ]+@sha256:[0-9a-f]{64}$$' Dockerfile || { echo "Docker runtime base is not digest-pinned" >&2; exit 1; }; \
	grep -Eq '^        actionlint$$' .github/workflows/test.yml; \
	grep -Eq '^        shellcheck ' .github/workflows/test.yml; \
	echo "toolchain declarations aligned"

test-asdf: ## Test the public asdf plugin scripts
	@tests/asdf-plugin.sh

# --- Docker ---------------------------------------------------------------

docker-build: ## Build the container image as $(PREFIX):$(VERSION)
	@docker build \
	  --build-arg "VERSION=$(VERSION)" \
	  --build-arg "REVISION=$$(git rev-parse HEAD 2>/dev/null || printf unknown)" \
	  -t "$(PREFIX):$(VERSION)" \
	  -t $(PREFIX):latest \
	  -f Dockerfile .

docker-run: ## Run the container image with a mounted config (CONFIG=path/to/config.hcl)
	@if [ -z "$(CONFIG)" ]; then echo "usage: make docker-run CONFIG=path/to/config.hcl"; exit 1; fi
	@config="$$(realpath -- "$(CONFIG)")" && docker run --rm -p 127.0.0.1:8080:8080 -v "$$config:/etc/s3proxy/config.hcl:ro" \
	  --env-file .env "$(PREFIX):latest"

# --- Sandbox (docker compose) ---------------------------------------------

sandbox-up: ## Start sandbox stack via docker compose (DAEMON=true for detached)
	@$(COMPOSE) $(UP_FLAGS)

sandbox-down: ## Stop sandbox stack
	@$(COMPOSE) $(DOWN_FLAGS)

sandbox-destroy: ## Stop sandbox + remove containers, volumes, and images
	@$(COMPOSE) $(DESTROY_FLAGS)
	@docker image prune -f || true

sandbox-reset: sandbox-destroy sandbox-up ## Destroy then up

sandbox-logs: ## Show recent sandbox logs (tail=50)
	@$(COMPOSE) $(LOGS_FLAGS)

sandbox-logs-follow: ## Tail sandbox logs live
	@$(COMPOSE) logs -f

sandbox-ps: ## List running sandbox services
	@$(COMPOSE) ps

# --- Integration tests (docker-compose sandbox) ---------------------------
#
# Run the live sandbox stack, then start s3proxy locally pointed at
# `sandbox/integration-config.hcl`, and run the integration test suite.
# Requires the docker-compose sandbox to be up (sandbox-up) OR uses the
# convenience target `sandbox-integration-up` which starts the sandbox and
# s3proxy in one shot.
#
# Requires a populated `.env` file (see .env.example). The proxy listens on
# the host's :8082 (see `listener "http" "public" { address = ":8082" }` in
# sandbox/integration-config.hcl).

INTEGRATION_CONFIG := sandbox/integration-config.hcl
INTEGRATION_PROXY_PID := $(DIST_DIR)/s3proxy-integration.pid
INTEGRATION_PROXY_LOG := $(DIST_DIR)/s3proxy-integration.log

test-integration: ## Run integration tests against live sandbox stack
	@go test -tags integration -count=1 -v ./internal/integration/...

test-integration-race: ## Run integration tests with the race detector
	@go test -tags integration -race -count=1 -v ./internal/integration/...

sandbox-integration-up: ## Start sandbox + s3proxy pointed at integration config
	@scripts/run-integration.sh

sandbox-integration-down: ## Stop s3proxy (sandbox integration) + sandbox stack
	@if [ -f $(INTEGRATION_PROXY_PID) ]; then \
	  kill $$(cat $(INTEGRATION_PROXY_PID)) 2>/dev/null || true; \
	  rm -f $(INTEGRATION_PROXY_PID); \
	  echo "s3proxy stopped"; \
	fi
	@$(MAKE) sandbox-down
