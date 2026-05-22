#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${ONEFACTURE_SANDBOX_URL:-https://sandbox.onefacture.io}"
BASE_URL="${BASE_URL%/}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "==> Checking sandbox reachability: ${BASE_URL}/healthz"
curl -fsS "${BASE_URL}/healthz" >/dev/null

echo "==> Creating sandbox credentials"
curl -fsS -X POST "${BASE_URL}/v1/sandbox/credentials" \
	-H "Content-Type: application/json" \
	-d '{"name":"onefacture smoke test"}' >"${tmpdir}/credentials.json"

API_KEY="$(
	python - "${tmpdir}/credentials.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    print(json.load(f)["api_key"])
PY
)"

INVOICE_IDEMPOTENCY_KEY="smoke-$(date +%s)"
python - "${ROOT_DIR}/docs/examples/commercial-invoice.json" "${tmpdir}/invoice.json" "${INVOICE_IDEMPOTENCY_KEY}" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    payload = json.load(f)
payload["num"] = f"SMOKE-{sys.argv[3]}"
with open(sys.argv[2], "w", encoding="utf-8") as f:
    json.dump(payload, f)
PY

echo "==> Creating and submitting first invoice"
curl -fsS -X POST "${BASE_URL}/v1/invoices?submit=true" \
	-H "X-API-Key: ${API_KEY}" \
	-H "Idempotency-Key: ${INVOICE_IDEMPOTENCY_KEY}" \
	-H "Content-Type: application/json" \
	-d @"${tmpdir}/invoice.json" >"${tmpdir}/invoice-res.json"

INVOICE_ID="$(
	python - "${tmpdir}/invoice-res.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    print(json.load(f)["id"])
PY
)"

echo "==> Checking invoice timeline"
curl -fsS "${BASE_URL}/v1/invoices/${INVOICE_ID}/timeline" \
	-H "X-API-Key: ${API_KEY}" >"${tmpdir}/timeline.json"

python - "${tmpdir}/timeline.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    timeline = json.load(f)
if "timeline" not in timeline:
    raise SystemExit("timeline response missing timeline")
print("Sandbox smoke test passed")
PY
