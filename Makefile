.PHONY: help build test test-unit test-integration lint lint-helm lint-docker fmt tidy clean dev run docker docker-build sidecar sidecar-dev migrate-up migrate-down openapi verify-local verify-backlog-manifest audit-backlog-completion smoke-backlog-completion-audit verify-external-smokes check-external-env smoke-external-env create-external-evidence collect-external-evidence smoke-external-evidence-collector review-external-evidence smoke-external-evidence-review verify-external-evidence verify-external-evidence-smoke verify-sdk verify-external verify-live-pa verify-public-sandbox verify-sdk-registries verify-kms-broker verify-outcome-metrics

GO            ?= go
GOLANGCI_LINT ?= golangci-lint
BIN_DIR       := bin
APP_NAME      := onefacture-api
PKG           := ./...
COVER_OUT     := coverage.out

DB_URL ?= postgres://onefacture:onefacture@localhost:5432/onefacture?sslmode=disable
GATE ?= all

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

build: ## Compile API binary
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/api
	$(GO) build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/onefacture ./cmd/onefacture

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

verify-local: ## Verify all local acceptance gates for the 3-wave backlog
	bash scripts/verify_local_acceptance.sh

verify-backlog-manifest: ## Verify backlog issue-to-artifact acceptance manifest
	ruby scripts/verify_backlog_acceptance_manifest.rb

audit-backlog-completion: ## Audit strict 3-wave backlog completion (BUNDLE=... optional)
	ruby scripts/audit_backlog_completion.rb

smoke-backlog-completion-audit: ## Verify the strict backlog completion audit fails closed
	bash scripts/smoke_backlog_completion_audit.sh

verify-external-smokes: ## Verify local smokes/pre-publication checks for external gates
	bash scripts/verify_external_gate_smokes.sh

check-external-env: ## Check required environment for external acceptance gates (GATE=all)
	bash scripts/check_external_acceptance_env.sh "$(GATE)"

smoke-external-env: ## Verify the external acceptance environment checker
	bash scripts/smoke_external_acceptance_env.sh

create-external-evidence: ## Create external acceptance evidence bundle scaffold (STAMP=YYYY-MM-DD)
	bash scripts/create_external_evidence_bundle.sh "$(STAMP)"

collect-external-evidence: ## Run external gates and collect a verified evidence bundle (STAMP=YYYY-MM-DD)
	bash scripts/collect_external_acceptance_evidence.sh "$(STAMP)"

smoke-external-evidence-collector: ## Verify the external evidence collector with fake gates
	bash scripts/smoke_external_evidence_collector.sh

review-external-evidence: ## Review a verified external evidence bundle against backlog issues (BUNDLE=...)
	@test -n "$(BUNDLE)" || (echo "BUNDLE is required" >&2; exit 2)
	ruby scripts/review_external_evidence_bundle.rb "$(BUNDLE)"

smoke-external-evidence-review: ## Verify the external evidence review helper
	bash scripts/smoke_external_evidence_review.sh

verify-external-evidence: ## Verify external acceptance evidence bundle (BUNDLE=...)
	@test -n "$(BUNDLE)" || (echo "BUNDLE is required" >&2; exit 2)
	bash scripts/verify_external_evidence_bundle.sh "$(BUNDLE)"

verify-external-evidence-smoke: ## Verify the external evidence bundle verifier
	bash scripts/smoke_external_evidence_bundle.sh

verify-sdk: ## Verify local SDK release artifacts install correctly
	bash scripts/verify_sdk_release_artifacts.sh

verify-external: ## Verify all external acceptance gates (requires live env)
	bash scripts/verify_external_acceptance.sh all

verify-live-pa: ## Verify live PA sandbox round-trips (requires credentials)
	bash scripts/verify_external_acceptance.sh live-pa

verify-public-sandbox: ## Verify deployed public sandbox quickstart
	bash scripts/verify_external_acceptance.sh public-sandbox

verify-sdk-registries: ## Verify published PyPI/npm SDK installs
	bash scripts/verify_external_acceptance.sh sdk-registries

verify-kms-broker: ## Verify deployed KMS broker active key endpoint
	bash scripts/verify_external_acceptance.sh kms-broker

verify-outcome-metrics: ## Verify deployed rejection retry outcome metric
	bash scripts/verify_external_acceptance.sh outcome-metrics
