#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

bash -n \
	scripts/smoke_public_sandbox_local.sh \
	scripts/smoke_live_pa_gate_local.sh \
	scripts/smoke_kms_gate_local.sh \
	scripts/smoke_outcome_metrics_gate_local.sh \
	scripts/verify_sdk_release_artifacts.sh \
	scripts/verify_external_acceptance.sh

bash scripts/smoke_public_sandbox_local.sh
bash scripts/smoke_live_pa_gate_local.sh
bash scripts/smoke_kms_gate_local.sh
bash scripts/smoke_outcome_metrics_gate_local.sh
bash scripts/verify_sdk_release_artifacts.sh

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
if bash scripts/verify_external_acceptance.sh not-a-gate >"$tmpdir/invalid.out" 2>"$tmpdir/invalid.err"; then
	echo "expected verify_external_acceptance.sh to reject invalid gate" >&2
	exit 1
fi
if ! grep -Fq "usage: scripts/verify_external_acceptance.sh [all|live-pa|public-sandbox|sdk-registries|kms-broker|outcome-metrics]" "$tmpdir/invalid.err"; then
	echo "invalid gate output did not include usage" >&2
	exit 1
fi

fakebin="$tmpdir/fakebin"
mkdir -p "$fakebin"
cat >"$fakebin/python" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "-m" && "${2:-}" == "venv" ]]; then
	mkdir -p "$3/bin"
	cat >"$3/bin/python" <<'PYSH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "-m" && "${2:-}" == "pip" && "${3:-}" == "install" && "${4:-}" == "--upgrade" ]]; then
	exit 0
fi
if [[ "${1:-}" == "-m" && "${2:-}" == "pip" && "${3:-}" == "install" && "${4:-}" == "onefacture" ]]; then
	echo "fake PyPI missing onefacture" >&2
	exit 1
fi
exit 0
PYSH
	chmod +x "$3/bin/python"
	exit 0
fi
exit 0
SH
cat >"$fakebin/npm" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "init" ]]; then
	exit 0
fi
if [[ "${1:-}" == "install" && "${2:-}" == "@onefacture/sdk" ]]; then
	echo "fake npm missing @onefacture/sdk" >&2
	exit 1
fi
exit 0
SH
chmod +x "$fakebin/python" "$fakebin/npm"
if PATH="$fakebin:$PATH" bash scripts/verify_external_acceptance.sh sdk-registries >"$tmpdir/sdk.out" 2>"$tmpdir/sdk.err"; then
	echo "expected sdk-registries smoke to fail with fake missing packages" >&2
	exit 1
fi
if ! grep -Fq "PyPI onefacture install failed" "$tmpdir/sdk.err"; then
	echo "sdk-registries smoke did not report PyPI failure" >&2
	exit 1
fi
if ! grep -Fq "npm @onefacture/sdk install failed" "$tmpdir/sdk.err"; then
	echo "sdk-registries smoke did not report npm failure" >&2
	exit 1
fi
