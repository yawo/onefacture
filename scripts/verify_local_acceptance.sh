#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mapfile -t go_files < <(find . -path './.git' -prune -o -path './.lean-ctx' -prune -o -name '*.go' -print)
gofmt_out="$(gofmt -l "${go_files[@]}")"
if [[ -n "$gofmt_out" ]]; then
	echo "gofmt required:" >&2
	echo "$gofmt_out" >&2
	exit 1
fi

go test ./cmd/onefacture \
	./internal/adapters \
	./internal/adapters/mock \
	./internal/adapters/sandbox \
	./internal/adapters/chorus \
	./internal/adapters/docaposte \
	./internal/adapters/pennylane \
	./internal/adapters/registry \
	./internal/config \
	./internal/core/facturx \
	./internal/core/invoice \
	./internal/directory \
	./internal/events \
	./internal/gateway \
	./internal/jurisdiction \
	./internal/reliability \
	./internal/security \
	./internal/validation \
	./internal/gateway/routes \
	./internal/gateway/middleware \
	./internal/gateway/problem \
	./internal/gateway/openapi \
	./internal/webhooks \
	./internal/workers

go test -short ./internal/storage
go test ./internal/storage -run 'Test(InvoiceRepoEncryptsAndDecryptsArtifacts|InvoiceRepoLeavesArtifactsPlainWithoutEncryptor)'
go test ./internal/adapters/live -tags=live_pa -count=1
go test ./... -run '^$'

mapfile -t shell_scripts < <(find scripts -name '*.sh' -print | sort)
bash -n "${shell_scripts[@]}"
mapfile -t ruby_scripts < <(find scripts -name '*.rb' -print | sort)
for script in "${ruby_scripts[@]}"; do
	ruby -c "$script"
done
ruby scripts/verify_backlog_acceptance_manifest.rb
bash scripts/verify_external_gate_smokes.sh
bash scripts/smoke_external_evidence_bundle.sh
bash scripts/smoke_backlog_completion_audit.sh
bash scripts/smoke_external_evidence_collector.sh
bash scripts/smoke_external_acceptance_env.sh
bash scripts/smoke_external_evidence_review.sh
ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f) }' \
	.github/workflows/ci.yml \
	.github/workflows/external-acceptance.yml \
	.github/workflows/sandbox-smoke.yml \
	.github/workflows/sdk-publish.yml \
	deploy/helm/onefacture/values-sandbox.yaml

git diff --check
