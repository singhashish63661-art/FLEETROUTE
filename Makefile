# ── GPS Fleet Management Platform — Makefile ─────────────────────────────────
# Works on Windows (PowerShell), Linux, and macOS.
# On Windows: run targets as   make <target>
# On Linux/macOS: same commands work natively.

SERVICES := ingestion-service stream-processor api-service websocket-service

# DB connection used for local migrate target (override via env or .env)
TIMESCALE_DSN ?= postgres://gpsgo:gpsgo@localhost:5432/gpsgo?sslmode=disable

.PHONY: all build test lint tidy docker-up docker-down \
        migrate-up migrate-down migrate-create \
        frontend-install frontend-dev frontend-build \
        gen-keys clean help

all: build

## ── Help ──────────────────────────────────────────────────────────────────────
help:
	@echo "Available targets:"
	@echo "  docker-up        Start all infrastructure (DB, Redis, NATS, monitoring)"
	@echo "  docker-infra     Start only DB + Redis + NATS (skip app services)"
	@echo "  docker-down      Stop and remove all containers + volumes"
	@echo "  migrate-up       Run all pending DB migrations (via Docker)"
	@echo "  migrate-down     Roll back the last migration (via Docker)"
	@echo "  frontend-install npm install in frontend/"
	@echo "  frontend-dev     Start Vite dev server"
	@echo "  gen-keys         Generate JWT RSA-2048 keypair into secrets/"
	@echo "  build            go build all services into bin/"
	@echo "  test             go test -race all modules"
	@echo "  tidy             go mod tidy all modules"
	@echo "  clean            Remove bin/ and frontend/dist/"

## ── Docker ────────────────────────────────────────────────────────────────────

# Start the full stack (infrastructure only — no app services that need Go build)
docker-up:
	docker compose up -d timescaledb redis nats prometheus grafana

# Wait for DB to be healthy then run migrations
migrate-up: docker-up
	@echo "Waiting for TimescaleDB to be ready..."
	docker compose run --rm migrate
	@echo "Migrations complete."

migrate-down:
	docker run --rm \
		-v "$(CURDIR)/migrations:/migrations" \
		migrate/migrate:v4.17.0 \
		-path=/migrations \
		-database="$(TIMESCALE_DSN)" \
		down 1

migrate-create:
	docker run --rm \
		-v "$(CURDIR)/migrations:/migrations" \
		migrate/migrate:v4.17.0 \
		create -ext sql -dir /migrations -seq $(NAME)

# Start infra only (skips app services that require Go build)
docker-infra:
	docker compose up -d timescaledb redis nats

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f $(SERVICE)

## ── Go ────────────────────────────────────────────────────────────────────────

# On Windows without bash, run these in individual PowerShell sessions per module.
# On Linux/macOS these loop automatically.
build:
	@echo "Building all services..."
	cd pkg              && go build ./...
	cd protocols        && go build ./...
	cd ingestion-service  && go build -o ../bin/ingestion-service   ./cmd/...
	cd stream-processor   && go build -o ../bin/stream-processor    ./cmd/...
	cd api-service        && go build -o ../bin/api-service         ./cmd/...
	cd websocket-service  && go build -o ../bin/websocket-service   ./cmd/...

test:
	cd pkg              && go test -race -count=1 ./...
	cd protocols        && go test -race -count=1 ./...
	cd ingestion-service  && go test -race -count=1 ./...
	cd stream-processor   && go test -race -count=1 ./...
	cd api-service        && go test -race -count=1 ./...

tidy:
	cd pkg              && go mod tidy
	cd protocols        && go mod tidy
	cd ingestion-service  && go mod tidy
	cd stream-processor   && go mod tidy
	cd api-service        && go mod tidy
	cd websocket-service  && go mod tidy

## ── Frontend ──────────────────────────────────────────────────────────────────
frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

## ── Keys ──────────────────────────────────────────────────────────────────────
# Requires OpenSSL to be installed (comes with Git for Windows).
gen-keys:
	go build -o bin/gen_keys scripts/gen_keys.go
	bin/gen_keys
	@echo "JWT keys written to secrets/"

## ── Cleanup ───────────────────────────────────────────────────────────────────
clean:
	@if exist bin  rmdir /s /q bin
	@if exist frontend\dist  rmdir /s /q frontend\dist
