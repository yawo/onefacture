#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

stamp="2099-12-31-collector-smoke"
bundle="docs/operations/evidence/${stamp}-external-acceptance"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir" "$bundle"' EXIT
rm -rf "$bundle"

mkdir -p "$tmpdir/bin"
cat >"$tmpdir/bin/make" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
	verify-live-pa)
		printf 'Bearer abcdefghijklmnopqrstuvwxyz123456\n'
		printf 'PASS\n'
		printf 'ok  \tgithub.com/yawo/onefacture/internal/adapters/live\t0.123s\n'
		;;
	verify-public-sandbox)
		if [[ "${ONEFACTURE_FAKE_FAIL_PUBLIC_SANDBOX:-}" == "true" ]]; then
			printf 'Sandbox smoke failed after HTTP 500\n'
			exit 1
		fi
		printf 'Sandbox smoke test passed\n'
		;;
	verify-sdk-registries)
		printf 'sk-abcdefghijklmnopqrstuvwxyz123456\n'
		printf 'PyPI onefacture install ok\n'
		printf 'npm @onefacture/sdk install ok\n'
		;;
	verify-kms-broker)
		printf 'KMS active key ok: redacted-key-id\n'
		;;
	verify-outcome-metrics)
		printf 'ofx_live_abcdefghijklmnopqrstuvwxyz123456\n'
		printf 'outcome metric ok: retried=2 accepted_after_retry=1 success_rate=0.5\n'
		;;
	verify-external)
		printf 'Bearer abcdefghijklmnopqrstuvwxyz123456\n'
		printf 'sk-abcdefghijklmnopqrstuvwxyz123456\n'
		printf 'ofx_prod_abcdefghijklmnopqrstuvwxyz123456\n'
		printf 'PASS\n'
		printf 'ok  \tgithub.com/yawo/onefacture/internal/adapters/live\t0.123s\n'
		printf 'Sandbox smoke test passed\n'
		printf 'PyPI onefacture install ok\n'
		printf 'npm @onefacture/sdk install ok\n'
		printf 'KMS active key ok: redacted-key-id\n'
		printf 'outcome metric ok: retried=2 accepted_after_retry=1 success_rate=0.5\n'
		;;
	*)
		echo "unexpected fake make target: ${1:-}" >&2
		exit 2
		;;
esac
EOF
chmod +x "$tmpdir/bin/make"

bad_links_stamp="2099-12-31-collector-bad-links-smoke"
bad_links_bundle="docs/operations/evidence/${bad_links_stamp}-external-acceptance"
rm -rf "$bad_links_bundle"
if PATH="$tmpdir/bin:$PATH" \
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
	ONEFACTURE_EVIDENCE_OPERATOR="collector-smoke" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="collector-smoke-env" \
	ONEFACTURE_EVIDENCE_LINKS="redacted" \
	bash scripts/collect_external_acceptance_evidence.sh "$bad_links_stamp" >"$tmpdir/bad-links.out" 2>&1; then
	echo "expected collector preflight to reject evidence links without URL" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_LINKS: must include at least one evidence URL" "$tmpdir/bad-links.out"; then
	echo "collector bad-links preflight did not include URL requirement" >&2
	exit 1
fi
if [[ -e "$bad_links_bundle" ]]; then
	echo "collector created evidence bundle despite bad links preflight" >&2
	exit 1
fi

bad_operator_stamp="2099-12-31-collector-bad-operator-smoke"
bad_operator_bundle="docs/operations/evidence/${bad_operator_stamp}-external-acceptance"
rm -rf "$bad_operator_bundle"
if PATH="$tmpdir/bin:$PATH" \
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
	ONEFACTURE_EVIDENCE_OPERATOR="unknown" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="collector-smoke-env" \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	bash scripts/collect_external_acceptance_evidence.sh "$bad_operator_stamp" >"$tmpdir/bad-operator.out" 2>&1; then
	echo "expected collector preflight to reject unknown operator" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_OPERATOR must name the actual reviewer or automation identity" "$tmpdir/bad-operator.out"; then
	echo "collector bad-operator preflight did not include operator requirement" >&2
	exit 1
fi
if [[ -e "$bad_operator_bundle" ]]; then
	echo "collector created evidence bundle despite bad operator preflight" >&2
	exit 1
fi

bad_environment_stamp="2099-12-31-collector-bad-environment-smoke"
bad_environment_bundle="docs/operations/evidence/${bad_environment_stamp}-external-acceptance"
rm -rf "$bad_environment_bundle"
if PATH="$tmpdir/bin:$PATH" \
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
	ONEFACTURE_EVIDENCE_OPERATOR="collector-smoke" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="unknown" \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	bash scripts/collect_external_acceptance_evidence.sh "$bad_environment_stamp" >"$tmpdir/bad-environment.out" 2>&1; then
	echo "expected collector preflight to reject unknown environment" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_EVIDENCE_ENVIRONMENT must name the external acceptance target" "$tmpdir/bad-environment.out"; then
	echo "collector bad-environment preflight did not include environment requirement" >&2
	exit 1
fi
if [[ -e "$bad_environment_bundle" ]]; then
	echo "collector created evidence bundle despite bad environment preflight" >&2
	exit 1
fi

PATH="$tmpdir/bin:$PATH" \
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
	ONEFACTURE_EVIDENCE_OPERATOR="collector-smoke" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="collector-smoke-env" \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	bash scripts/collect_external_acceptance_evidence.sh "$stamp" >/tmp/onefacture-collector-smoke.out

bash scripts/verify_external_evidence_bundle.sh "$bundle" >/dev/null

if grep -R "Paste redacted output" "$bundle"; then
	echo "collector left scaffold placeholder text in evidence bundle" >&2
	exit 1
fi

if grep -RE '(Bearer [A-Za-z0-9._~-]{12,}|sk-[A-Za-z0-9]{12,}|ofx_(live|prod)_[A-Za-z0-9._~-]{8,})' "$bundle"; then
	echo "collector failed to redact secret-like output" >&2
	exit 1
fi

if ! grep -RFq "Bearer [REDACTED]" "$bundle"; then
	echo "collector smoke did not exercise bearer redaction" >&2
	exit 1
fi

failed_stamp="2099-12-31-collector-failure-smoke"
failed_bundle="docs/operations/evidence/${failed_stamp}-external-acceptance"
rm -rf "$failed_bundle"
trap 'rm -rf "$tmpdir" "$bundle" "$bad_links_bundle" "$bad_operator_bundle" "$bad_environment_bundle" "$failed_bundle"' EXIT

if PATH="$tmpdir/bin:$PATH" \
	ONEFACTURE_FAKE_FAIL_PUBLIC_SANDBOX="true" \
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
	ONEFACTURE_EVIDENCE_OPERATOR="collector-smoke" \
	ONEFACTURE_EVIDENCE_ENVIRONMENT="collector-smoke-env" \
	ONEFACTURE_EVIDENCE_LINKS="https://github.com/yawo/onefacture/actions/runs/123456789" \
	bash scripts/collect_external_acceptance_evidence.sh "$failed_stamp" >"$tmpdir/failed.out" 2>"$tmpdir/failed.err"; then
	echo "expected collector to fail when a gate fails" >&2
	exit 1
fi

if ! grep -Fq -- "- make verify-public-sandbox: FAIL" "$failed_bundle/summary.md"; then
	echo "collector failure summary did not mark public sandbox as FAIL" >&2
	exit 1
fi
if ! grep -Fq "Sandbox smoke failed after HTTP 500" "$failed_bundle/public-sandbox.log"; then
	echo "collector failure bundle did not preserve failing gate log" >&2
	exit 1
fi

echo "external evidence collector smoke passed"
