.DEFAULT_GOAL := build
SHELL := /bin/bash

CACHE_DIR ?= $(CURDIR)/.cache
export GOCACHE := $(CACHE_DIR)/go-build
export GOLANGCI_LINT_CACHE := $(CACHE_DIR)/golangci-lint

build:
	go build ./cmd/lyra

run: build
	./lyra serve

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

dev: infra-up infra-wait
	@test -f .env || { echo "missing .env; copy .env.example first" >&2; exit 1; }
	@test -d web/node_modules || { echo "missing web dependencies; run: cd web && npm install" >&2; exit 1; }
	@set -euo pipefail; \
	set -a; source ./.env; set +a; \
	go run ./cmd/lyra migrate; \
	go run ./cmd/lyra serve & api_pid=$$!; \
	go run ./cmd/lyra worker & worker_pid=$$!; \
	(cd web && npm run dev -- --host 127.0.0.1) & web_pid=$$!; \
	cleanup() { kill $$api_pid $$worker_pid $$web_pid 2>/dev/null || true; wait $$api_pid $$worker_pid $$web_pid 2>/dev/null || true; }; \
	trap cleanup EXIT INT TERM; \
	wait -n $$api_pid $$worker_pid $$web_pid

db-migrate: build
	./lyra migrate

eval: build
	./lyra eval --manifest ./testdata/manifests/eval.json

benchmark: build
	./lyra benchmark --synthetic-tracks=1000

docker-build:
	docker build -t lyra:local .

db-reset:
	@echo "$@ is not available until its corresponding implementation phase is complete" >&2; exit 1
