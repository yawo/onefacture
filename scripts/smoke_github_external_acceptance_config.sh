#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cat >"$tmpdir/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "$1" in
	auth)
		[[ "${2:-}" == "status" ]] || exit 2
		exit 0
		;;
	variable)
		[[ "${2:-}" == "list" ]] || exit 2
		if [[ "${ONEFACTURE_FAKE_GH_COMPLETE:-}" == "1" ]]; then
			cat <<'VARS'
ONEFACTURE_CHORUS_BASE_URL
ONEFACTURE_DOCAPOSTE_BASE_URL
ONEFACTURE_PENNYLANE_BASE_URL
ONEFACTURE_SANDBOX_URL
ONEFACTURE_KMS_URL
ONEFACTURE_PROD_API_URL
ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE
ONEFACTURE_EVIDENCE_LINKS
ONEFACTURE_EVIDENCE_ENVIRONMENT
VARS
		fi
		;;
	secret)
		[[ "${2:-}" == "list" ]] || exit 2
		if [[ "${ONEFACTURE_FAKE_GH_COMPLETE:-}" == "1" ]]; then
			cat <<'SECRETS'
ONEFACTURE_CHORUS_ACCESS_TOKEN
ONEFACTURE_DOCAPOSTE_API_TOKEN
ONEFACTURE_PENNYLANE_API_TOKEN
ONEFACTURE_PROD_API_KEY
NPM_TOKEN
ONEFACTURE_KMS_TOKEN
SECRETS
		fi
		;;
	api)
		echo "fake-gh"
		;;
	*)
		echo "unexpected fake gh command: $*" >&2
		exit 2
		;;
esac
EOF
chmod +x "$tmpdir/gh"

missing_out="$tmpdir/missing.out"
if PATH="$tmpdir:$PATH" bash scripts/check_github_external_acceptance_config.sh yawo/onefacture >"$missing_out" 2>&1; then
	echo "expected GitHub config check to fail with missing vars/secrets" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_CHORUS_BASE_URL: missing" "$missing_out"; then
	echo "missing GitHub config output did not include required variable" >&2
	exit 1
fi
if ! grep -Fq "NPM_TOKEN: missing" "$missing_out"; then
	echo "missing GitHub config output did not include required secret" >&2
	exit 1
fi

complete_out="$tmpdir/complete.out"
PATH="$tmpdir:$PATH" ONEFACTURE_FAKE_GH_COMPLETE=1 \
	bash scripts/check_github_external_acceptance_config.sh yawo/onefacture >"$complete_out"
if ! grep -Fq "GitHub Actions external acceptance configuration ok" "$complete_out"; then
	echo "complete GitHub config output did not include success marker" >&2
	exit 1
fi
if ! grep -Fq "ONEFACTURE_KMS_TOKEN: set" "$complete_out"; then
	echo "complete GitHub config output did not include optional KMS token status" >&2
	exit 1
fi

echo "GitHub external acceptance config smoke passed"
