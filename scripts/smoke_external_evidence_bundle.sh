#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

make_bundle() {
	local bundle="$1"
	local commit_sha
	commit_sha="$(git rev-parse HEAD)"
	mkdir -p "$bundle"
	cat >"$bundle/live-pa.log" <<'EOF'
=== RUN   TestLivePAAdapters
--- PASS: TestLivePAAdapters (0.01s)
PASS
ok  	github.com/yawo/onefacture/internal/adapters/live	0.123s
EOF
	cat >"$bundle/public-sandbox.log" <<'EOF'
Sandbox smoke test passed
EOF
	cat >"$bundle/sdk-registries.log" <<'EOF'
PyPI onefacture install ok
npm @onefacture/sdk install ok
EOF
	cat >"$bundle/kms-broker.log" <<'EOF'
KMS active key ok: redacted-key-id
EOF
	cat >"$bundle/outcome-metrics.log" <<'EOF'
outcome metric ok: retried=2 accepted_after_retry=1 success_rate=0.5
EOF
	cat >"$bundle/all.log" <<'EOF'
PASS
ok  	github.com/yawo/onefacture/internal/adapters/live	0.123s
Sandbox smoke test passed
PyPI onefacture install ok
npm @onefacture/sdk install ok
KMS active key ok: redacted-key-id
outcome metric ok: retried=2 accepted_after_retry=1 success_rate=0.5
EOF
	cat >"$bundle/summary.md" <<EOF
# External acceptance summary

Commit SHA: ${commit_sha}
Branch: main
Operator: local-smoke
Timestamp: 2026-05-22T00:00:00Z
Environment: redacted-live-targets

Commands:
- make verify-live-pa: PASS
- make verify-public-sandbox: PASS
- make verify-sdk-registries: PASS
- make verify-kms-broker: PASS
- make verify-outcome-metrics: PASS
- make verify-external: PASS

Reruns: none
Links: https://github.com/yawo/onefacture/actions/runs/123456789
EOF
}

valid_bundle="$tmpdir/valid"
make_bundle "$valid_bundle"
bash scripts/verify_external_evidence_bundle.sh "$valid_bundle"

secret_bundle="$tmpdir/secret"
make_bundle "$secret_bundle"
printf "Bearer abcdefghijklmnop\n" >>"$secret_bundle/live-pa.log"
if bash scripts/verify_external_evidence_bundle.sh "$secret_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject unredacted secret" >&2
	exit 1
fi

missing_bundle="$tmpdir/missing"
make_bundle "$missing_bundle"
rm "$missing_bundle/kms-broker.log"
if bash scripts/verify_external_evidence_bundle.sh "$missing_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject missing log" >&2
	exit 1
fi

marker_bundle="$tmpdir/missing-marker"
make_bundle "$marker_bundle"
printf "HTTP 200 without terminal success marker\n" >"$marker_bundle/public-sandbox.log"
if bash scripts/verify_external_evidence_bundle.sh "$marker_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject missing success marker" >&2
	exit 1
fi

wrong_commit_bundle="$tmpdir/wrong-commit"
make_bundle "$wrong_commit_bundle"
sed -i.bak 's/^Commit SHA:.*/Commit SHA: deadbeef/' "$wrong_commit_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$wrong_commit_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject mismatched commit" >&2
	exit 1
fi

failed_summary_bundle="$tmpdir/failed-summary"
make_bundle "$failed_summary_bundle"
sed -i.bak 's/make verify-kms-broker: PASS/make verify-kms-broker: FAIL/' "$failed_summary_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$failed_summary_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject non-PASS summary command" >&2
	exit 1
fi

bad_timestamp_bundle="$tmpdir/bad-timestamp"
make_bundle "$bad_timestamp_bundle"
sed -i.bak 's/^Timestamp:.*/Timestamp: 2026-05-22 00:00:00/' "$bad_timestamp_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$bad_timestamp_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject malformed summary timestamp" >&2
	exit 1
fi

invalid_timestamp_bundle="$tmpdir/invalid-timestamp"
make_bundle "$invalid_timestamp_bundle"
sed -i.bak 's/^Timestamp:.*/Timestamp: 2026-99-99T99:99:99Z/' "$invalid_timestamp_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$invalid_timestamp_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject invalid summary timestamp" >&2
	exit 1
fi

missing_reruns_bundle="$tmpdir/missing-reruns"
make_bundle "$missing_reruns_bundle"
sed -i.bak '/^Reruns:/d' "$missing_reruns_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$missing_reruns_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject missing reruns summary field" >&2
	exit 1
fi

empty_links_bundle="$tmpdir/empty-links"
make_bundle "$empty_links_bundle"
sed -i.bak 's/^Links:.*/Links:/' "$empty_links_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$empty_links_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject empty links summary field" >&2
	exit 1
fi

placeholder_links_bundle="$tmpdir/placeholder-links"
make_bundle "$placeholder_links_bundle"
sed -i.bak 's/^Links:.*/Links: redacted/' "$placeholder_links_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$placeholder_links_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject summary links without URL" >&2
	exit 1
fi

placeholder_url_bundle="$tmpdir/placeholder-url"
make_bundle "$placeholder_url_bundle"
sed -i.bak 's#^Links:.*#Links: https://example.invalid/onefacture/external-acceptance#' "$placeholder_url_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$placeholder_url_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject placeholder evidence URL" >&2
	exit 1
fi

placeholder_reruns_bundle="$tmpdir/placeholder-reruns"
make_bundle "$placeholder_reruns_bundle"
sed -i.bak 's/^Reruns:.*/Reruns: TODO/' "$placeholder_reruns_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$placeholder_reruns_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject placeholder reruns summary field" >&2
	exit 1
fi

unknown_operator_bundle="$tmpdir/unknown-operator"
make_bundle "$unknown_operator_bundle"
sed -i.bak 's/^Operator:.*/Operator: unknown/' "$unknown_operator_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$unknown_operator_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject unknown operator summary field" >&2
	exit 1
fi

unknown_environment_bundle="$tmpdir/unknown-environment"
make_bundle "$unknown_environment_bundle"
sed -i.bak 's/^Environment:.*/Environment: unknown/' "$unknown_environment_bundle/summary.md"
if bash scripts/verify_external_evidence_bundle.sh "$unknown_environment_bundle" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject unknown environment summary field" >&2
	exit 1
fi

scaffold_path="$(bash scripts/create_external_evidence_bundle.sh "2099-12-31-evidence-smoke")"
trap 'rm -rf "$tmpdir" "$scaffold_path"' EXIT
if bash scripts/verify_external_evidence_bundle.sh "$scaffold_path" >/dev/null 2>&1; then
	echo "expected evidence verifier to reject untouched scaffold" >&2
	exit 1
fi

echo "external evidence bundle smoke passed"
