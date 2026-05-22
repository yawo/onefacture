#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

missing_out="$tmpdir/missing.out"
if bash scripts/check_external_acceptance_env.sh >"$missing_out" 2>&1; then
	echo "expected external env check to fail with missing vars" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_CHORUS_BASE_URL: missing" "$missing_out"; then
	echo "missing env output did not include required variable status" >&2
	exit 1
fi

complete_out="$tmpdir/complete.out"
env \
	ONEFACTURE_CHORUS_BASE_URL="https://chorus.example.test" \
	ONEFACTURE_CHORUS_ACCESS_TOKEN="redacted" \
	ONEFACTURE_DOCAPOSTE_BASE_URL="https://docaposte.example.test" \
	ONEFACTURE_DOCAPOSTE_API_TOKEN="redacted" \
	ONEFACTURE_PENNYLANE_BASE_URL="https://pennylane.example.test" \
	ONEFACTURE_PENNYLANE_API_TOKEN="redacted" \
	ONEFACTURE_SANDBOX_URL="https://sandbox.example.test" \
	ONEFACTURE_KMS_URL="https://kms.example.test" \
	ONEFACTURE_PROD_API_URL="https://api.example.test" \
	ONEFACTURE_PROD_API_KEY="redacted" \
	ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE=0.4 \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	ONEFACTURE_EVIDENCE_OPERATOR="env-smoke" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="external-env-smoke" \
	bash scripts/check_external_acceptance_env.sh >"$complete_out"

if ! grep -Fq "external acceptance environment ok" "$complete_out"; then
	echo "complete env output did not include success marker" >&2
	exit 1
fi

bad_links_out="$tmpdir/bad-links.out"
if env \
	ONEFACTURE_CHORUS_BASE_URL="https://chorus.example.test" \
	ONEFACTURE_CHORUS_ACCESS_TOKEN="redacted" \
	ONEFACTURE_DOCAPOSTE_BASE_URL="https://docaposte.example.test" \
	ONEFACTURE_DOCAPOSTE_API_TOKEN="redacted" \
	ONEFACTURE_PENNYLANE_BASE_URL="https://pennylane.example.test" \
	ONEFACTURE_PENNYLANE_API_TOKEN="redacted" \
	ONEFACTURE_SANDBOX_URL="https://sandbox.example.test" \
	ONEFACTURE_KMS_URL="https://kms.example.test" \
	ONEFACTURE_PROD_API_URL="https://api.example.test" \
	ONEFACTURE_PROD_API_KEY="redacted" \
	ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE=0.4 \
	ONEFACTURE_EVIDENCE_LINKS="redacted" \
	bash scripts/check_external_acceptance_env.sh >"$bad_links_out" 2>&1; then
	echo "expected external env check to reject evidence links without URL" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_LINKS: must include at least one evidence URL" "$bad_links_out"; then
	echo "bad links env output did not include URL requirement" >&2
	exit 1
fi

placeholder_links_out="$tmpdir/placeholder-links.out"
if env \
	ONEFACTURE_CHORUS_BASE_URL="https://chorus.example.test" \
	ONEFACTURE_CHORUS_ACCESS_TOKEN="redacted" \
	ONEFACTURE_DOCAPOSTE_BASE_URL="https://docaposte.example.test" \
	ONEFACTURE_DOCAPOSTE_API_TOKEN="redacted" \
	ONEFACTURE_PENNYLANE_BASE_URL="https://pennylane.example.test" \
	ONEFACTURE_PENNYLANE_API_TOKEN="redacted" \
	ONEFACTURE_SANDBOX_URL="https://sandbox.example.test" \
	ONEFACTURE_KMS_URL="https://kms.example.test" \
	ONEFACTURE_PROD_API_URL="https://api.example.test" \
	ONEFACTURE_PROD_API_KEY="redacted" \
	ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE=0.4 \
	ONEFACTURE_EVIDENCE_LINKS="https://example.invalid/onefacture/external-acceptance" \
	bash scripts/check_external_acceptance_env.sh >"$placeholder_links_out" 2>&1; then
	echo "expected external env check to reject placeholder evidence URL" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_LINKS: must not use placeholder or localhost URLs" "$placeholder_links_out"; then
	echo "placeholder links env output did not include placeholder URL requirement" >&2
	exit 1
fi

bad_operator_out="$tmpdir/bad-operator.out"
if env \
	ONEFACTURE_CHORUS_BASE_URL="https://chorus.example.test" \
	ONEFACTURE_CHORUS_ACCESS_TOKEN="redacted" \
	ONEFACTURE_DOCAPOSTE_BASE_URL="https://docaposte.example.test" \
	ONEFACTURE_DOCAPOSTE_API_TOKEN="redacted" \
	ONEFACTURE_PENNYLANE_BASE_URL="https://pennylane.example.test" \
	ONEFACTURE_PENNYLANE_API_TOKEN="redacted" \
	ONEFACTURE_SANDBOX_URL="https://sandbox.example.test" \
	ONEFACTURE_KMS_URL="https://kms.example.test" \
	ONEFACTURE_PROD_API_URL="https://api.example.test" \
	ONEFACTURE_PROD_API_KEY="redacted" \
	ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE=0.4 \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	ONEFACTURE_EVIDENCE_OPERATOR="unknown" \
	bash scripts/check_external_acceptance_env.sh >"$bad_operator_out" 2>&1; then
	echo "expected external env check to reject unknown evidence operator" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_OPERATOR: must name the actual reviewer or automation identity" "$bad_operator_out"; then
	echo "bad operator env output did not include operator requirement" >&2
	exit 1
fi

bad_environment_out="$tmpdir/bad-environment.out"
if env \
	ONEFACTURE_CHORUS_BASE_URL="https://chorus.example.test" \
	ONEFACTURE_CHORUS_ACCESS_TOKEN="redacted" \
	ONEFACTURE_DOCAPOSTE_BASE_URL="https://docaposte.example.test" \
	ONEFACTURE_DOCAPOSTE_API_TOKEN="redacted" \
	ONEFACTURE_PENNYLANE_BASE_URL="https://pennylane.example.test" \
	ONEFACTURE_PENNYLANE_API_TOKEN="redacted" \
	ONEFACTURE_SANDBOX_URL="https://sandbox.example.test" \
	ONEFACTURE_KMS_URL="https://kms.example.test" \
	ONEFACTURE_PROD_API_URL="https://api.example.test" \
	ONEFACTURE_PROD_API_KEY="redacted" \
	ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE=0.4 \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	ONEFACTURE_EVIDENCE_OPERATOR="env-smoke" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="unknown" \
	bash scripts/check_external_acceptance_env.sh >"$bad_environment_out" 2>&1; then
	echo "expected external env check to reject unknown evidence environment" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_ENVIRONMENT: must name the external acceptance target" "$bad_environment_out"; then
	echo "bad environment env output did not include environment requirement" >&2
	exit 1
fi

public_out="$tmpdir/public.out"
env ONEFACTURE_SANDBOX_URL="https://sandbox.example.test" \
	bash scripts/check_external_acceptance_env.sh public-sandbox >"$public_out"
if grep -Fq "ONEFACTURE_CHORUS_BASE_URL: missing" "$public_out"; then
	echo "public-sandbox env check required unrelated live PA variables" >&2
	exit 1
fi
if ! grep -Fq "External acceptance required environment for public-sandbox" "$public_out"; then
	echo "public-sandbox env output did not include mode-specific heading" >&2
	exit 1
fi

echo "external acceptance env smoke passed"
