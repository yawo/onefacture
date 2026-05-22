#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 docs/operations/evidence/YYYY-MM-DD-external-acceptance" >&2
	exit 2
fi

bundle="${1%/}"
if [[ ! -d "$bundle" ]]; then
	echo "evidence bundle directory not found: $bundle" >&2
	exit 1
fi

required_logs=(
	live-pa.log
	public-sandbox.log
	sdk-registries.log
	kms-broker.log
	outcome-metrics.log
	all.log
)

for log in "${required_logs[@]}"; do
	path="$bundle/$log"
	if [[ ! -s "$path" ]]; then
		echo "missing or empty evidence log: $path" >&2
		exit 1
	fi
	if grep -Fq "Paste redacted output" "$path"; then
		echo "evidence log still contains scaffold placeholder: $path" >&2
		exit 1
	fi
done

require_log_marker() {
	local log="$1"
	local marker="$2"
	local path="$bundle/$log"
	if ! grep -Eq "$marker" "$path"; then
		echo "evidence log missing success marker: $log" >&2
		exit 1
	fi
}

require_log_marker "live-pa.log" '(^PASS$|^ok[[:space:]]+[^[:space:]]*/internal/adapters/live[[:space:]])'
require_log_marker "public-sandbox.log" 'Sandbox smoke test passed'
require_log_marker "sdk-registries.log" 'PyPI onefacture install ok'
require_log_marker "sdk-registries.log" 'npm @onefacture/sdk install ok'
require_log_marker "kms-broker.log" 'KMS active key ok'
require_log_marker "outcome-metrics.log" 'outcome metric ok'
require_log_marker "all.log" '(^PASS$|^ok[[:space:]]+[^[:space:]]*/internal/adapters/live[[:space:]])'
require_log_marker "all.log" 'Sandbox smoke test passed'
require_log_marker "all.log" 'PyPI onefacture install ok'
require_log_marker "all.log" 'npm @onefacture/sdk install ok'
require_log_marker "all.log" 'KMS active key ok'
require_log_marker "all.log" 'outcome metric ok'

summary="$bundle/summary.md"
if [[ ! -s "$summary" ]]; then
	echo "missing or empty evidence summary: $summary" >&2
	exit 1
fi

required_summary_patterns=(
	"Commit SHA"
	"Branch"
	"Operator"
	"Timestamp"
	"Environment"
	"Reruns"
	"Links"
	"make verify-live-pa"
	"make verify-public-sandbox"
	"make verify-sdk-registries"
	"make verify-kms-broker"
	"make verify-outcome-metrics"
	"make verify-external"
)

for pattern in "${required_summary_patterns[@]}"; do
	if ! grep -Fq "$pattern" "$summary"; then
		echo "summary missing required field or command: $pattern" >&2
		exit 1
	fi
done

required_nonempty_fields=(
	"Commit SHA"
	"Branch"
	"Operator"
	"Timestamp"
	"Environment"
	"Reruns"
	"Links"
)

for field in "${required_nonempty_fields[@]}"; do
	if ! grep -Eq "^${field}:[[:space:]]*[^[:space:]]+" "$summary"; then
		echo "summary field is empty: $field" >&2
		exit 1
	fi
	if grep -Eiq "^${field}:[[:space:]]*(TODO|TBD|placeholder)([[:space:]]*$|[[:space:][:punct:]])" "$summary"; then
		echo "summary field still contains placeholder text: $field" >&2
		exit 1
	fi
done

if grep -Eiq '^Operator:[[:space:]]*(unknown|reviewer)([[:space:]]*$|[[:space:][:punct:]])' "$summary"; then
	echo "summary operator must name the actual reviewer or automation identity" >&2
	exit 1
fi
if grep -Eiq '^Environment:[[:space:]]*unknown([[:space:]]*$|[[:space:][:punct:]])' "$summary"; then
	echo "summary environment must name the external acceptance target" >&2
	exit 1
fi

summary_commit="$(awk -F': *' '/^Commit SHA:/ { print $2; exit }' "$summary")"
current_commit="$(git rev-parse HEAD)"
if [[ "$summary_commit" != "$current_commit" ]]; then
	echo "summary commit does not match current HEAD: $summary_commit != $current_commit" >&2
	exit 1
fi

summary_timestamp="$(awk '/^Timestamp:/ { sub(/^Timestamp:[[:space:]]*/, ""); print; exit }' "$summary")"
if ! [[ "$summary_timestamp" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]; then
	echo "summary timestamp must be ISO-8601 UTC: $summary_timestamp" >&2
	exit 1
fi
if ! date -u -d "$summary_timestamp" "+%Y-%m-%dT%H:%M:%SZ" >/dev/null 2>&1; then
	echo "summary timestamp is not a valid UTC instant: $summary_timestamp" >&2
	exit 1
fi

summary_links="$(awk '/^Links:/ { sub(/^Links:[[:space:]]*/, ""); print; exit }' "$summary")"
if ! [[ "$summary_links" =~ https?:// ]]; then
	echo "summary links must include at least one evidence URL" >&2
	exit 1
fi
if [[ "$summary_links" =~ (example\.(invalid|com|org)|localhost|127\.0\.0\.1) ]]; then
	echo "summary links must not use placeholder or localhost URLs" >&2
	exit 1
fi

required_pass_commands=(
	"make verify-live-pa"
	"make verify-public-sandbox"
	"make verify-sdk-registries"
	"make verify-kms-broker"
	"make verify-outcome-metrics"
	"make verify-external"
)

for command in "${required_pass_commands[@]}"; do
	if ! grep -Eq "${command}:[[:space:]]*PASS\\b" "$summary"; then
		echo "summary command is not marked PASS: $command" >&2
		exit 1
	fi
done

if grep -Eiq '(Bearer [A-Za-z0-9._~-]{12,}|sk-[A-Za-z0-9]{12,}|ofx_(live|prod)_[A-Za-z0-9._~-]{8,})' "$bundle"/*.log "$summary"; then
	echo "evidence bundle appears to contain an unredacted secret" >&2
	exit 1
fi

echo "external evidence bundle ok: $bundle"
