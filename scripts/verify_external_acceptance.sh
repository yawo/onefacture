#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mode="${1:-all}"

require_env() {
	local name="$1"
	if [[ -z "${!name:-}" ]]; then
		echo "missing required env: ${name}" >&2
		return 1
	fi
}

verify_live_pa() {
	require_env ONEFACTURE_CHORUS_BASE_URL
	require_env ONEFACTURE_CHORUS_ACCESS_TOKEN
	require_env ONEFACTURE_DOCAPOSTE_BASE_URL
	require_env ONEFACTURE_DOCAPOSTE_API_TOKEN
	require_env ONEFACTURE_PENNYLANE_BASE_URL
	require_env ONEFACTURE_PENNYLANE_API_TOKEN
	ONEFACTURE_REQUIRE_LIVE_PA=true go test -tags=live_pa ./internal/adapters/live -count=1 -v
}

verify_public_sandbox() {
	require_env ONEFACTURE_SANDBOX_URL
	bash scripts/smoke_public_sandbox.sh
}

verify_sdk_registries() {
	tmpdir="$(mktemp -d)"
	trap 'rm -rf "$tmpdir"' RETURN
	failed=0

	if python -m venv "$tmpdir/py" &&
		PIP_CONFIG_FILE=/dev/null "$tmpdir/py/bin/python" -m pip install --upgrade pip >/dev/null &&
		PIP_CONFIG_FILE=/dev/null "$tmpdir/py/bin/python" -m pip install onefacture >/dev/null &&
		"$tmpdir/py/bin/python" - <<'PY'
from onefacture import Client
Client("ofx_test")
print("PyPI onefacture install ok")
PY
	then
		:
	else
		echo "PyPI onefacture install failed" >&2
		failed=1
	fi

	mkdir -p "$tmpdir/npm"
	if (
		cd "$tmpdir/npm" &&
			npm init -y >/dev/null &&
			npm install @onefacture/sdk >/dev/null &&
			node --input-type=module - <<'JS'
import { OnefactureClient } from "@onefacture/sdk";
new OnefactureClient({ apiKey: "ofx_test" });
console.log("npm @onefacture/sdk install ok");
JS
	)
	then
		:
	else
		echo "npm @onefacture/sdk install failed" >&2
		failed=1
	fi

	return "$failed"
}

verify_kms_broker() {
	require_env ONEFACTURE_KMS_URL
	tmpfile="$(mktemp)"
	trap 'rm -f "$tmpfile"' RETURN
	curl_args=(-fsS)
	if [[ -n "${ONEFACTURE_KMS_TOKEN:-}" ]]; then
		curl_args+=(-H "Authorization: Bearer ${ONEFACTURE_KMS_TOKEN}")
	fi
	curl "${curl_args[@]}" "${ONEFACTURE_KMS_URL%/}/keys/active" >"$tmpfile"
	python - "$tmpfile" <<'PY'
import base64
import binascii
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    payload = json.load(f)
key_id = payload.get("key_id")
raw_key = payload.get("key")
if not key_id:
    raise SystemExit("KMS response missing key_id")
if not raw_key:
    raise SystemExit("KMS response missing key")
try:
    key = bytes.fromhex(raw_key)
except ValueError:
    try:
        key = base64.b64decode(raw_key, validate=True)
    except binascii.Error as exc:
        raise SystemExit(f"KMS key is not hex or base64: {exc}") from exc
if len(key) != 32:
    raise SystemExit(f"KMS key must decode to 32 bytes, got {len(key)}")
print(f"KMS active key ok: {key_id}")
PY
}

verify_outcome_metrics() {
	require_env ONEFACTURE_PROD_API_URL
	require_env ONEFACTURE_PROD_API_KEY
	require_env ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE
	tmpfile="$(mktemp)"
	trap 'rm -f "$tmpfile"' RETURN
	curl -fsS "${ONEFACTURE_PROD_API_URL%/}/v1/analytics/rejection-retry-success-rate" \
		-H "X-API-Key: ${ONEFACTURE_PROD_API_KEY}" >"$tmpfile"
	python - "$tmpfile" "${ONEFACTURE_MIN_RETRIED_INVOICES:-1}" "${ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE}" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    payload = json.load(f)
min_retried = int(sys.argv[2])
baseline_rate = float(sys.argv[3])
if payload.get("metric") != "rejection_retry_success_rate":
    raise SystemExit("unexpected metric name")
retried = int(payload.get("retried_invoices", 0))
accepted = int(payload.get("accepted_after_retry", 0))
rate = float(payload.get("success_rate", 0))
if retried < min_retried:
    raise SystemExit(f"retried_invoices {retried} below required minimum {min_retried}")
if accepted < 0 or accepted > retried:
    raise SystemExit("accepted_after_retry outside valid range")
if rate < 0 or rate > 1:
    raise SystemExit("success_rate outside [0,1]")
if baseline_rate < 0 or baseline_rate > 1:
    raise SystemExit("baseline success rate outside [0,1]")
if rate <= baseline_rate:
    raise SystemExit(f"success_rate {rate} did not improve over baseline {baseline_rate}")
print(f"outcome metric ok: retried={retried} accepted_after_retry={accepted} success_rate={rate} baseline_success_rate={baseline_rate}")
PY
}

case "$mode" in
	live-pa)
		verify_live_pa
		;;
	public-sandbox)
		verify_public_sandbox
		;;
	sdk-registries)
		verify_sdk_registries
		;;
	kms-broker)
		verify_kms_broker
		;;
	outcome-metrics)
		verify_outcome_metrics
		;;
	all)
		verify_live_pa
		verify_public_sandbox
		verify_sdk_registries
		verify_kms_broker
		verify_outcome_metrics
		;;
	*)
		echo "usage: $0 [all|live-pa|public-sandbox|sdk-registries|kms-broker|outcome-metrics]" >&2
		exit 2
		;;
esac
