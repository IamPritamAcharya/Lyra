.DEFAULT_GOAL := build
SHELL := /bin/bash

CACHE_DIR ?= $(CURDIR)/.cache
BIN_DIR := $(CURDIR)/bin
BINARY := $(BIN_DIR)/lyra
export GOCACHE := $(CACHE_DIR)/go-build
export GOLANGCI_LINT_CACHE := $(CACHE_DIR)/golangci-lint

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/lyra

run: build
	$(BINARY) serve

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

test test-unit:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run

vuln:
	govulncheck ./...

test-integration:
	go test -tags=integration ./...

verify: fmt vet test

infra-up:
	docker compose up -d postgres valkey minio

infra-down:
	docker compose down

infra-wait:
	@for attempt in $$(seq 1 30); do \
		if docker compose exec -T postgres pg_isready -U lyra -d lyra >/dev/null 2>&1 \
			&& docker compose exec -T valkey valkey-cli ping >/dev/null 2>&1 \
			&& curl -fsS http://localhost:9000/minio/health/live >/dev/null; then \
			echo "local infrastructure is ready"; exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "local infrastructure did not become ready within 30 seconds" >&2; exit 1

dev:
	@test -f .env || { echo "missing .env; copy .env.example first" >&2; exit 1; }
	@test -d web/node_modules || { echo "missing web dependencies; run: cd web && npm install" >&2; exit 1; }
	@set -euo pipefail; \
	api_pid=""; worker_pid=""; web_pid=""; interrupted=0; \
	cleanup() { \
		result=$$?; trap - EXIT INT TERM; \
		if [ "$$interrupted" -eq 1 ]; then result=0; fi; \
		echo "stopping Lyra development services..."; \
		for pid in "$$api_pid" "$$worker_pid" "$$web_pid"; do \
			if [ -n "$$pid" ]; then kill -TERM -- "-$$pid" 2>/dev/null || true; fi; \
		done; \
		for pid in "$$api_pid" "$$worker_pid" "$$web_pid"; do \
			if [ -n "$$pid" ]; then wait "$$pid" 2>/dev/null || true; fi; \
		done; \
		docker compose down; \
		exit "$$result"; \
	}; \
	trap cleanup EXIT; \
	trap 'interrupted=1; exit 0' INT TERM; \
	docker compose up -d postgres valkey minio; \
	for attempt in $$(seq 1 30); do \
		if docker compose exec -T postgres pg_isready -U lyra -d lyra >/dev/null 2>&1 \
			&& docker compose exec -T valkey valkey-cli ping >/dev/null 2>&1 \
			&& curl -fsS http://localhost:9000/minio/health/live >/dev/null; then break; fi; \
		if [ "$$attempt" -eq 30 ]; then echo "local infrastructure did not become ready within 30 seconds" >&2; exit 1; fi; \
		sleep 1; \
	done; \
	echo "local infrastructure is ready"; \
	set -a; source ./.env; set +a; \
	mkdir -p $(BIN_DIR); \
	go build -o $(BINARY) ./cmd/lyra; \
	$(BINARY) migrate; \
	setsid $(BINARY) serve & api_pid=$$!; \
	setsid $(BINARY) worker & worker_pid=$$!; \
	(cd web && exec setsid npm run dev) & web_pid=$$!; \
	wait -n $$api_pid $$worker_pid $$web_pid

db-migrate: build
	$(BINARY) migrate

eval: build
	$(BINARY) eval --manifest ./testdata/manifests/eval.json

benchmark: build
	$(BINARY) benchmark --synthetic-tracks=1000

docker-build:
	docker build -t lyra:local .

db-reset:
	@echo "$@ is not available until its corresponding implementation phase is complete" >&2; exit 1
