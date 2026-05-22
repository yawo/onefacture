#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir" sdks/typescript/node_modules sdks/typescript/dist sdks/python/dist' EXIT

python - <<'PY'
from pathlib import Path
import tomllib

data = tomllib.loads(Path("sdks/python/pyproject.toml").read_text())
project = data["project"]
for field in ("name", "version", "description", "readme", "requires-python", "dependencies"):
    if not project.get(field):
        raise SystemExit(f"missing Python project field: {field}")
PY

python -m compileall -q sdks/python/src
python -m venv "$tmpdir/py"
PIP_CONFIG_FILE=/dev/null "$tmpdir/py/bin/python" -m pip install --upgrade pip >/dev/null
PIP_CONFIG_FILE=/dev/null "$tmpdir/py/bin/python" -m pip install ./sdks/python >/dev/null
"$tmpdir/py/bin/python" - <<'PY'
from onefacture import Client

client = Client("ofx_test", base_url="https://example.test/")
assert client.base_url == "https://example.test"
print("python sdk import ok")
PY

npm --prefix sdks/typescript ci
npm --prefix sdks/typescript run build
package_json="$(cd sdks/typescript && npm pack --json)"
package_file="$(PACKAGE_JSON="$package_json" python - <<'PY'
import json
import os

print(json.loads(os.environ["PACKAGE_JSON"])[0]["filename"])
PY
)"
mkdir -p "$tmpdir/ts"
(
	cd "$tmpdir/ts"
	npm init -y >/dev/null
	npm install "$ROOT/sdks/typescript/$package_file" >/dev/null
	node --input-type=module - <<'JS'
import { OnefactureClient } from "@onefacture/sdk";

const client = new OnefactureClient({ apiKey: "ofx_test", baseUrl: "https://example.test/" });
if (!client) throw new Error("client construction failed");
console.log("typescript sdk import ok");
JS
)
rm -f "sdks/typescript/$package_file"
