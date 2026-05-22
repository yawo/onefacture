#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

stamp="${1:-$(date -u +%F)}"
bundle="docs/operations/evidence/${stamp}-external-acceptance"
operator="${ONEFACTURE_EVIDENCE_OPERATOR:-unknown}"
environment="${ONEFACTURE_EVIDENCE_ENVIRONMENT:-unknown}"
links="${ONEFACTURE_EVIDENCE_LINKS:-redacted}"

if [[ -e "$bundle" ]]; then
	echo "evidence bundle already exists: $bundle" >&2
	exit 1
fi

if [[ "$operator" =~ ^(TODO|TBD|placeholder|reviewer|unknown)$ ]]; then
	echo "ONEFACTURE_EVIDENCE_OPERATOR must name the actual reviewer or automation identity" >&2
	exit 1
fi
if [[ "$environment" =~ ^(TODO|TBD|placeholder|unknown)$ ]]; then
	echo "ONEFACTURE_EVIDENCE_ENVIRONMENT must name the external acceptance target" >&2
	exit 1
fi

bash scripts/check_external_acceptance_env.sh

mkdir -p "$bundle"

redact_stream() {
	sed -E \
		-e 's/(Bearer )[A-Za-z0-9._~-]{12,}/\1[REDACTED]/g' \
		-e 's/sk-[A-Za-z0-9]{12,}/sk-[REDACTED]/g' \
		-e 's/ofx_(live|prod)_[A-Za-z0-9._~-]{8,}/ofx_\1_[REDACTED]/g'
}

declare -A commands=(
	["live-pa"]="make verify-live-pa"
	["public-sandbox"]="make verify-public-sandbox"
	["sdk-registries"]="make verify-sdk-registries"
	["kms-broker"]="make verify-kms-broker"
	["outcome-metrics"]="make verify-outcome-metrics"
	["all"]="make verify-external"
)

gates=(live-pa public-sandbox sdk-registries kms-broker outcome-metrics all)
declare -A results=()
failed=0

for gate in "${gates[@]}"; do
	log="$bundle/${gate}.log"
	echo "==> ${commands[$gate]}"
	set +e
	${commands[$gate]} 2>&1 | redact_stream | tee "$log"
	status="${PIPESTATUS[0]}"
	set -e
	if [[ "$status" -eq 0 ]]; then
		results["$gate"]="PASS"
	else
		results["$gate"]="FAIL"
		failed=1
	fi
done

cat >"$bundle/summary.md" <<EOF
# External acceptance summary

Commit SHA: $(git rev-parse HEAD)
Branch: $(git rev-parse --abbrev-ref HEAD)
Operator: ${operator}
Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Environment: ${environment}

Commands:
- make verify-live-pa: ${results[live-pa]}
- make verify-public-sandbox: ${results[public-sandbox]}
- make verify-sdk-registries: ${results[sdk-registries]}
- make verify-kms-broker: ${results[kms-broker]}
- make verify-outcome-metrics: ${results[outcome-metrics]}
- make verify-external: ${results[all]}

Reruns: document any reruns here before review.
Links: ${links}
EOF

if [[ "$failed" -ne 0 ]]; then
	echo "external evidence collection failed; inspect bundle: $bundle" >&2
	exit 1
fi

bash scripts/verify_external_evidence_bundle.sh "$bundle"
echo "$bundle"
