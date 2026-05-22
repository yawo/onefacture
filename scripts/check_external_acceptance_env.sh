#!/usr/bin/env bash
set -euo pipefail

mode="${1:-all}"

required_env=()
case "$mode" in
	all)
		required_env=(
			ONEFACTURE_CHORUS_BASE_URL
			ONEFACTURE_CHORUS_ACCESS_TOKEN
			ONEFACTURE_DOCAPOSTE_BASE_URL
			ONEFACTURE_DOCAPOSTE_API_TOKEN
			ONEFACTURE_PENNYLANE_BASE_URL
			ONEFACTURE_PENNYLANE_API_TOKEN
			ONEFACTURE_SANDBOX_URL
					ONEFACTURE_KMS_URL
					ONEFACTURE_PROD_API_URL
					ONEFACTURE_PROD_API_KEY
					ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE
					ONEFACTURE_EVIDENCE_LINKS
					ONEFACTURE_EVIDENCE_OPERATOR
					ONEFACTURE_EVIDENCE_ENVIRONMENT
			)
		;;
	live-pa)
		required_env=(
			ONEFACTURE_CHORUS_BASE_URL
			ONEFACTURE_CHORUS_ACCESS_TOKEN
			ONEFACTURE_DOCAPOSTE_BASE_URL
			ONEFACTURE_DOCAPOSTE_API_TOKEN
			ONEFACTURE_PENNYLANE_BASE_URL
			ONEFACTURE_PENNYLANE_API_TOKEN
		)
		;;
	public-sandbox)
		required_env=(ONEFACTURE_SANDBOX_URL)
		;;
	sdk-registries)
		required_env=()
		;;
	kms-broker)
		required_env=(ONEFACTURE_KMS_URL)
		;;
	outcome-metrics)
			required_env=(
				ONEFACTURE_PROD_API_URL
				ONEFACTURE_PROD_API_KEY
				ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE
			)
		;;
	*)
		echo "usage: $0 [all|live-pa|public-sandbox|sdk-registries|kms-broker|outcome-metrics]" >&2
		exit 2
		;;
esac

optional_env=(
	ONEFACTURE_KMS_TOKEN
	ONEFACTURE_MIN_RETRIED_INVOICES
	ONEFACTURE_EVIDENCE_OPERATOR
	ONEFACTURE_EVIDENCE_ENVIRONMENT
)

missing=0
echo "External acceptance required environment for ${mode}:"
for name in "${required_env[@]}"; do
	if [[ -n "${!name:-}" ]]; then
		echo "- ${name}: set"
	else
		echo "- ${name}: missing"
		missing=1
	fi
done

if [[ "$mode" == "all" && -n "${ONEFACTURE_EVIDENCE_LINKS:-}" && ! "${ONEFACTURE_EVIDENCE_LINKS}" =~ https?:// ]]; then
	echo "- ONEFACTURE_EVIDENCE_LINKS: must include at least one evidence URL" >&2
	missing=1
fi
if [[ "$mode" == "all" && "${ONEFACTURE_EVIDENCE_LINKS:-}" =~ (example\.(invalid|com|org)|localhost|127\.0\.0\.1) ]]; then
	echo "- ONEFACTURE_EVIDENCE_LINKS: must not use placeholder or localhost URLs" >&2
	missing=1
fi
operator="${ONEFACTURE_EVIDENCE_OPERATOR:-unknown}"
if [[ "$mode" == "all" && "$operator" =~ ^(TODO|TBD|placeholder|reviewer|unknown)$ ]]; then
	echo "- ONEFACTURE_EVIDENCE_OPERATOR: must name the actual reviewer or automation identity" >&2
	missing=1
fi
environment="${ONEFACTURE_EVIDENCE_ENVIRONMENT:-unknown}"
if [[ "$mode" == "all" && "$environment" =~ ^(TODO|TBD|placeholder|unknown)$ ]]; then
	echo "- ONEFACTURE_EVIDENCE_ENVIRONMENT: must name the external acceptance target" >&2
	missing=1
fi

echo "External acceptance optional environment:"
for name in "${optional_env[@]}"; do
	if [[ -n "${!name:-}" ]]; then
		echo "- ${name}: set"
	else
		echo "- ${name}: unset"
	fi
done

if [[ "$missing" -ne 0 ]]; then
	echo "external acceptance environment is incomplete" >&2
	exit 1
fi

echo "external acceptance environment ok"
