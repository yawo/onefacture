.PHONY: help build test test-unit test-integration lint lint-helm lint-docker fmt tidy clean dev run docker docker-build sidecar sidecar-dev migrate-up migrate-down openapi

GO            ?= go
GOLANGCI_LINT ?= golangci-lint
BIN_DIR       := bin
APP_NAME      := onefacture-api
PKG           := ./...
COVER_OUT     := coverage.out

DB_URL ?= postgres://onefacture:onefacture@localhost:5432/onefacture?sslmode=disable

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

build: ## Compile API binary
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/api

test: test-unit ## Run all tests (alias for test-unit)

test-unit: ## Run unit tests with coverage
	$(GO) test -race -covermode=atomic -coverprofile=$(COVER_OUT) $(PKG)
	$(GO) tool cover -func=$(COVER_OUT) | tail -n 1

test-integration: ## Run integration tests (require docker)
	$(GO) test -race -tags=integration -timeout=10m $(PKG)

lint: ## Run linters available in the current environment
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run --timeout=5m; \
	else \
		echo "Skipping Go lint: $(GOLANGCI_LINT) is not installed in this environment"; \
	fi
	@$(MAKE) lint-helm
	@$(MAKE) lint-docker

lint-helm: ## Lint Helm chart when helm is installed
	@if command -v helm >/dev/null 2>&1; then \
		helm lint deploy/helm/onefacture; \
	else \
		echo "Skipping Helm lint: helm is not installed in this environment"; \
	fi

lint-docker: ## Validate Docker tooling when docker is installed
	@if command -v docker >/dev/null 2>&1; then \
		docker --version >/dev/null; \
	else \
		echo "Skipping Docker checks: docker is not installed in this environment"; \
	fi

fmt: ## Format Go code
	$(GO) fmt $(PKG)
	$(GO) vet $(PKG)

tidy: ## Tidy go.mod
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(COVER_OUT) coverage.html

run: ## Run the API locally
	$(GO) run ./cmd/api

dev: ## Start full dev stack (postgres, redis, sidecar, api)
	docker compose -f deploy/docker-compose.yml up --build

docker-build: ## Build the API docker image
	docker build -t onefacture/api:dev -f deploy/Dockerfile .

sidecar: ## Build the Python sidecar image
	docker build -t onefacture/sidecar:dev -f sidecar/Dockerfile sidecar

sidecar-dev: ## Run the Python sidecar locally
	cd sidecar && python -m venv .venv && . .venv/bin/activate && pip install -r requirements.txt && uvicorn app.main:app --reload --port 8081

migrate-up: ## Apply DB migrations
	migrate -path internal/storage/migrations -database "$(DB_URL)" up

migrate-down: ## Rollback last migration
	migrate -path internal/storage/migrations -database "$(DB_URL)" down 1

openapi: ## Regenerate OpenAPI spec
	$(GO) run ./cmd/genopenapi > docs/openapi.json
