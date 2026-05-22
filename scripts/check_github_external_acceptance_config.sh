#!/usr/bin/env bash
set -euo pipefail

repo="${1:-}"
gh_args=()
if [[ -n "$repo" ]]; then
	gh_args=(--repo "$repo")
fi

if ! command -v gh >/dev/null 2>&1; then
	echo "gh CLI is required to inspect GitHub Actions configuration" >&2
	exit 2
fi

if ! gh auth status >/dev/null 2>&1; then
	echo "gh CLI is not authenticated" >&2
	exit 2
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

gh variable list "${gh_args[@]}" --json name --jq '.[].name' >"$tmpdir/variables"
gh secret list "${gh_args[@]}" --json name --jq '.[].name' >"$tmpdir/secrets"

required_variables=(
	ONEFACTURE_CHORUS_BASE_URL
	ONEFACTURE_DOCAPOSTE_BASE_URL
	ONEFACTURE_PENNYLANE_BASE_URL
	ONEFACTURE_SANDBOX_URL
	ONEFACTURE_KMS_URL
	ONEFACTURE_PROD_API_URL
	ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE
	ONEFACTURE_EVIDENCE_LINKS
	ONEFACTURE_EVIDENCE_ENVIRONMENT
)

required_secrets=(
	ONEFACTURE_CHORUS_ACCESS_TOKEN
	ONEFACTURE_DOCAPOSTE_API_TOKEN
	ONEFACTURE_PENNYLANE_API_TOKEN
	ONEFACTURE_PROD_API_KEY
	NPM_TOKEN
)

missing=0
echo "GitHub Actions required variables:"
for name in "${required_variables[@]}"; do
	if grep -Fxq "$name" "$tmpdir/variables"; then
		echo "- ${name}: set"
	else
		echo "- ${name}: missing"
		missing=1
	fi
done

echo "GitHub Actions required secrets:"
for name in "${required_secrets[@]}"; do
	if grep -Fxq "$name" "$tmpdir/secrets"; then
		echo "- ${name}: set"
	else
		echo "- ${name}: missing"
		missing=1
	fi
done

echo "GitHub Actions optional secrets:"
if grep -Fxq "ONEFACTURE_KMS_TOKEN" "$tmpdir/secrets"; then
	echo "- ONEFACTURE_KMS_TOKEN: set"
else
	echo "- ONEFACTURE_KMS_TOKEN: unset"
fi

if [[ "$missing" -ne 0 ]]; then
	echo "GitHub Actions external acceptance configuration is incomplete" >&2
	exit 1
fi

echo "GitHub Actions external acceptance configuration ok"
