SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help
.PHONY: help build build-all build-single build-archive format fmt vet test test-race cover clean \
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

COMPOSE      := docker compose --env-file .env -f ./sandbox/docker-compose.yml
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
	CGO_ENABLED=0 go build -ldflags "$(LD_FLAGS)" -o $(DIST_DIR)/$(BINARY) $(MAIN_PKG)
	@echo "built $(DIST_DIR)/$(BINARY) (version $(VERSION))"

build-single: ## Build for a single OS:ARCH pair (OS_ARCH=linux:amd64)
	@set -e; \
	OS_ARCH=$(OS_ARCH); \
	OS=$$(echo $$OS_ARCH | cut -d: -f1); \
	ARCH=$$(echo $$OS_ARCH | cut -d: -f2); \
	echo "Building for OS=$$OS and ARCH=$$ARCH"; \
	DIR="$(DIST_DIR)/$$OS-$$ARCH"; \
	mkdir -p $$DIR; \
	EXT=$$(if [ "$$OS" = "windows" ]; then echo ".exe"; else echo ""; fi); \
	CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH \
	  go build -ldflags "$(LD_FLAGS) -X main.version=$(VERSION)/$$OS-$$ARCH" \
	  -o $$DIR/$(BINARY)$$EXT $(MAIN_PKG)

build-all: ## Cross-compile for all OS/arch pairs in OS_ARCH_PAIRS
	@$(foreach pair,$(OS_ARCH_PAIRS),$(MAKE) build-single OS_ARCH=$(pair);)

build-archive: ## Tar each cross-compiled dist/<os>-<arch>/ dir into a release archive
	@set -e; \
	for d in $(DIST_DIR)/*-*/; do \
	  [ -d "$$d" ] || continue; \
	  name=$$(basename "$$d"); \
	  archive="$(DIST_DIR)/$(PREFIX)-$$name.tar.gz"; \
	  (cd "$$d" && tar -czf "$$archive" .); \
	  echo "archived $$archive"; \
	done

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
	@go test -coverprofile=$(DIST_DIR)/coverage.out ./...
	@go tool cover -func=$(DIST_DIR)/coverage.out | tail -1
	@echo "coverage profile: $(DIST_DIR)/coverage.out"

clean: ## Remove dist/ and coverage artifacts
	@rm -rf $(DIST_DIR)
	@echo "cleaned $(DIST_DIR)"

# --- Run / validate -------------------------------------------------------

run: ## Run the server locally (CONFIG=path/to/config.hcl)
	@if [ -z "$(CONFIG)" ]; then echo "usage: make run CONFIG=path/to/config.hcl"; exit 1; fi
	@go run $(MAIN_PKG) serve --config $(CONFIG)

validate: ## Validate config without starting the server (CONFIG=path/to/config.hcl)
	@if [ -z "$(CONFIG)" ]; then echo "usage: make validate CONFIG=path/to/config.hcl"; exit 1; fi
	@go run $(MAIN_PKG) validate --config $(CONFIG)

# --- Docker ---------------------------------------------------------------

docker-build: ## Build the container image as $(PREFIX):$(VERSION)
	@docker build \
	  --build-arg VERSION=$(VERSION) \
	  -t $(PREFIX):$(VERSION) \
	  -t $(PREFIX):latest \
	  -f Dockerfile .

docker-run: ## Run the container image with a mounted config (CONFIG=path/to/config.hcl)
	@if [ -z "$(CONFIG)" ]; then echo "usage: make docker-run CONFIG=path/to/config.hcl"; exit 1; fi
	@docker run --rm -p 8080:8080 -v $(PWD)/$(CONFIG):/etc/s3proxy/config.hcl:ro \
	  --env-file .env \
	  $(PREFIX):latest

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
	@test -f .env || { echo "missing .env (see .env.example)"; exit 1; }
	@mkdir -p $(DIST_DIR)
	@echo "-> starting sandbox stack (detached)"
	@DAEMON=true $(MAKE) sandbox-up
	@echo "-> building s3proxy"
	@$(MAKE) build
	@echo "-> starting s3proxy with $(INTEGRATION_CONFIG) (env from .env)"
	@set -euo pipefail; \
	  set -a; . ./.env; set +a; \
	  ./$(DIST_DIR)/$(BINARY) serve --config $(INTEGRATION_CONFIG) >$(INTEGRATION_PROXY_LOG) 2>&1 & \
	  p=$$!; \
	  echo $$p > $(INTEGRATION_PROXY_PID); \
	  echo "s3proxy pid: $$p  log: $(INTEGRATION_PROXY_LOG)"; \
	  sleep 1; \
	  if ! kill -0 $$p 2>/dev/null; then \
	    echo "s3proxy exited immediately, log:"; \
	    cat $(INTEGRATION_PROXY_LOG); \
	    exit 1; \
	  fi
	@echo "-> running integration tests"
	@set -euo pipefail; \
	  rc=0; \
	  set -a; . ./.env; set +a; \
	  go test -tags integration -count=1 -v ./internal/integration/... || rc=$$?; \
	  echo "-> stopping s3proxy"; \
	  kill $$(cat $(INTEGRATION_PROXY_PID)) 2>/dev/null || true; \
	  rm -f $(INTEGRATION_PROXY_PID); \
	  $(MAKE) sandbox-down; \
	  exit $$rc

sandbox-integration-down: ## Stop s3proxy (sandbox integration) + sandbox stack
	@if [ -f $(INTEGRATION_PROXY_PID) ]; then \
	  kill $$(cat $(INTEGRATION_PROXY_PID)) 2>/dev/null || true; \
	  rm -f $(INTEGRATION_PROXY_PID); \
	  echo "s3proxy stopped"; \
	fi
	@$(MAKE) sandbox-down
