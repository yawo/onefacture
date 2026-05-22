#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

stamp="${1:-$(date -u +%F)}"
bundle="docs/operations/evidence/${stamp}-external-acceptance"

if [[ -e "$bundle" ]]; then
	echo "evidence bundle already exists: $bundle" >&2
	exit 1
fi

mkdir -p "$bundle"

for gate in live-pa public-sandbox sdk-registries kms-broker outcome-metrics all; do
	command="make verify-${gate}"
	if [[ "$gate" == "all" ]]; then
		command="make verify-external"
	fi
	cat >"$bundle/${gate}.log" <<EOF
# Paste redacted output for: ${command}
EOF
done

cat >"$bundle/summary.md" <<'EOF'
# External acceptance summary

Commit SHA:
Branch:
Operator:
Timestamp:
Environment:

Commands:
- make verify-live-pa:
- make verify-public-sandbox:
- make verify-sdk-registries:
- make verify-kms-broker:
- make verify-outcome-metrics:
- make verify-external:

Reruns:
Links:
EOF

echo "$bundle"
