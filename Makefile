# apps/budget/backend/.env feeds compose interpolation (AUTH_SECRET, TELEGRAM_BOT_TOKEN).
COMPOSE      := docker compose --env-file apps/budget/backend/.env -f infra/budget/compose/dev.yml
COMPOSE_TEST := docker compose -f infra/budget/compose/test.yml
COMPOSE_LEARNING := docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml
# Production stack (manual fallback — CI/CD is the primary path, see
# infra/budget/CI_CD.md). Images are pulled from the registry, never built here, so the
# .env must also carry BACKEND_IMAGE / FRONTEND_IMAGE (see infra/budget/.env.prod.example).
# --project-directory . makes the compose file's ./infra/edge/Caddyfile mount
# resolve from the repo root (it targets the on-server ~/budget_me layout).
COMPOSE_PROD := docker compose --env-file infra/budget/compose/.env --project-directory . -f infra/budget/compose/prod.yml

.PHONY: help up down restart logs ps psql reset
.PHONY: migrate.up migrate.down migrate.status migrate.reset
.PHONY: tidy front sqlc build vet test fmt lint
.PHONY: prod.pull prod.up prod.down prod.migrate prod.logs
.PHONY: learning.up learning.down learning.logs learning.migrate

# golangci-lint via the official pinned image (no host install, mirrors the
# sqlc target). Host Go caches are mounted so it reuses already-downloaded
# modules instead of re-fetching.
GOLANGCI_VERSION := v2.12.2
GOLANGCI := docker run --rm \
	-v $(CURDIR)/apps/budget/backend:/src -w /src \
	-v $(HOME)/go/pkg/mod:/go/pkg/mod \
	-v $(HOME)/.cache/golangci-lint:/root/.cache \
	golangci/golangci-lint:$(GOLANGCI_VERSION)

help:
	@echo "Backend stack (Docker):"
	@echo "  up              - start postgres, backend (with hot reload)"
	@echo "  down            - stop services"
	@echo "  restart         - restart backend container"
	@echo "  logs            - tail logs (Ctrl-C to stop)"
	@echo "  ps              - service status"
	@echo "  psql            - open psql shell inside postgres container"
	@echo "  reset           - down + remove volumes (DROPS DATA!)"
	@echo ""
	@echo "Migrations:"
	@echo "  migrate.up      - apply all pending migrations"
	@echo "  migrate.down    - roll back last migration"
	@echo "  migrate.status  - show migration status"
	@echo "  migrate.reset   - roll back ALL migrations"
	@echo ""
	@echo "Codegen:"
	@echo "  sqlc            - regenerate adapter/repository/postgre/queries from queries/*.sql"
	@echo ""
	@echo "Go checks (inside backend container):"
	@echo "  build           - go build ./..."
	@echo "  vet             - go vet ./..."
	@echo "  test            - go test ./... (unit + in-process API + arch)"
	@echo ""
	@echo "Lint / format (golangci-lint image):"
	@echo "  fmt             - auto-format (gofumpt + gci import grouping)"
	@echo "  lint            - golangci-lint run"
	@echo ""
	@echo "Other:"
	@echo "  tidy            - go mod tidy inside backend container"
	@echo "  front           - run frontend dev server (pnpm dev) on host"
	@echo "  learning.up     - build and start isolated learning dev stack"
	@echo "  learning.down   - stop learning dev stack"
	@echo "  learning.migrate - apply learning migrations"

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart backend

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

psql:
	$(COMPOSE) exec postgres psql -U budget_me -d budget_me

reset:
	$(COMPOSE) down -v

migrate.up:
	$(COMPOSE) exec backend go run ./cmd/migrate up

migrate.down:
	$(COMPOSE) exec backend go run ./cmd/migrate down

migrate.status:
	$(COMPOSE) exec backend go run ./cmd/migrate status

migrate.reset:
	$(COMPOSE) exec backend go run ./cmd/migrate reset

# ── e2e test stack (isolated from dev) ─────────────────────────────────
# `test.up` boots the stack on alt ports (5433 / 8081 / 9200), runs the
# migrations, and is what `cd apps/budget/frontend && pnpm test:e2e` expects to be up.
# `test.reset` is the "wipe everything and start clean" shortcut.

test.up:
	$(COMPOSE_TEST) up -d --build
	$(COMPOSE_TEST) exec backend go run ./cmd/migrate up

test.down:
	$(COMPOSE_TEST) down

test.reset:
	$(COMPOSE_TEST) down -v
	$(MAKE) test.up

test.logs:
	$(COMPOSE_TEST) logs -f

test.migrate:
	$(COMPOSE_TEST) exec backend go run ./cmd/migrate up

test.psql:
	$(COMPOSE_TEST) exec postgres psql -U budget_me -d budget_me_test

tidy:
	$(COMPOSE) exec backend go mod tidy

build:
	$(COMPOSE) exec backend go build ./...

vet:
	$(COMPOSE) exec backend go vet ./...

test:
	$(COMPOSE) exec backend go test ./...

sqlc:
	docker run --rm -v $(CURDIR)/apps/budget/backend:/src -w /src sqlc/sqlc:latest generate

fmt:
	$(GOLANGCI) golangci-lint fmt

lint:
	$(GOLANGCI) golangci-lint run

front:
	cd apps/budget/frontend && pnpm dev

learning.up:
	$(COMPOSE_LEARNING) up -d --build postgres
	$(COMPOSE_LEARNING) --profile tools run --rm migrate
	$(COMPOSE_LEARNING) up -d --build backend frontend

learning.down:
	$(COMPOSE_LEARNING) down

learning.logs:
	$(COMPOSE_LEARNING) logs -f

learning.migrate:
	$(COMPOSE_LEARNING) --profile tools run --rm migrate

# ── production stack (manual fallback; CI/CD is primary — see infra/budget/CI_CD.md) ──
# Requires a `docker login registry.gitlab.com` first and BACKEND_IMAGE /
# FRONTEND_IMAGE set in infra/budget/compose/.env — images come from the registry.
prod.pull:
	$(COMPOSE_PROD) pull

prod.up:
	$(COMPOSE_PROD) pull
	$(COMPOSE_PROD) up -d

prod.down:
	$(COMPOSE_PROD) down

prod.migrate:
	$(COMPOSE_PROD) --profile tools run --rm migrate

prod.logs:
	$(COMPOSE_PROD) logs -f
