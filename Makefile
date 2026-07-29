.DEFAULT_GOAL := build

build:
	go build ./cmd/lyra

run: build
	./lyra serve --with-worker

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

dev:
	docker compose up --build

db-migrate db-reset eval benchmark docker-build:
	@echo "$@ is not available until its corresponding implementation phase is complete" >&2; exit 1
