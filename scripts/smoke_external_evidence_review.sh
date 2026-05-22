#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
bundle="$tmpdir/valid-evidence"
commit_sha="$(git rev-parse HEAD)"
mkdir -p "$bundle"

cat >"$bundle/live-pa.log" <<'EOF'
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

out="$tmpdir/review.out"
ruby scripts/review_external_evidence_bundle.rb "$bundle" >"$out"

if ! grep -Fq "External evidence review checklist" "$out"; then
	echo "review output missing checklist heading" >&2
	exit 1
fi
if ! grep -Fq "#01 Intégration Chorus Pro PISTE sandbox" "$out"; then
	echo "review output missing issue mapping" >&2
	exit 1
fi
if ! grep -Fq "Evidence links: https://github.com/yawo/onefacture/actions/runs/123456789" "$out"; then
	echo "review output missing evidence links" >&2
	exit 1
fi
if ! grep -Fq "Evidence branch: main" "$out"; then
	echo "review output missing evidence branch" >&2
	exit 1
fi
if ! grep -Fq "Evidence environment: redacted-live-targets" "$out"; then
	echo "review output missing evidence environment" >&2
	exit 1
fi
if ! grep -Fq "Evidence reruns: none" "$out"; then
	echo "review output missing evidence reruns" >&2
	exit 1
fi
if ! grep -Fq "github-issues-vagues-acceptance.json" "$out"; then
	echo "review output missing update instruction" >&2
	exit 1
fi
if ! grep -Fq '"status": "covered_external"' "$out"; then
	echo "review output missing covered_external template" >&2
	exit 1
fi
if ! grep -Fq '"reviewed_evidence"' "$out"; then
	echo "review output missing reviewed_evidence template" >&2
	exit 1
fi
if ! grep -Fq "\"commit_sha\": \"${commit_sha}\"" "$out"; then
	echo "review output missing summary commit" >&2
	exit 1
fi
if ! grep -Fq '"reviewed_at": "2026-05-22T00:00:00Z"' "$out"; then
	echo "review output missing summary timestamp" >&2
	exit 1
fi
if ! grep -Fq '"reviewed_by": "local-smoke"' "$out"; then
	echo "review output missing summary operator" >&2
	exit 1
fi
if ! grep -Fq "Review document markers after human review:" "$out"; then
	echo "review output missing review marker section" >&2
	exit 1
fi
if ! grep -Fq "1. Intégration Chorus Pro PISTE sandbox (round-trip complet): covered_external" "$out"; then
	echo "review output missing per-issue review marker" >&2
	exit 1
fi
if ! grep -Fq "Completion audit status rows after human review:" "$out"; then
	echo "review output missing audit marker section" >&2
	exit 1
fi
if ! grep -Fq "| 1 | covered_external |" "$out"; then
	echo "review output missing per-issue audit marker" >&2
	exit 1
fi
if ! grep -Fq "Final gate: make audit-backlog-completion" "$out"; then
	echo "review output missing final audit gate" >&2
	exit 1
fi

invalid_bundle="$tmpdir/invalid-evidence"
mkdir -p "$invalid_bundle"
printf "Paste redacted output\n" >"$invalid_bundle/live-pa.log"
if ruby scripts/review_external_evidence_bundle.rb "$invalid_bundle" >"$tmpdir/invalid.out" 2>"$tmpdir/invalid.err"; then
	echo "expected review helper to reject invalid evidence bundle" >&2
	exit 1
fi
if ! grep -Fq "scaffold placeholder" "$tmpdir/invalid.err" "$tmpdir/invalid.out"; then
	echo "invalid evidence rejection did not include verifier failure" >&2
	exit 1
fi

echo "external evidence review smoke passed"
