#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

port_file="$(mktemp)"
python3 - "$port_file" <<'PY' &
from http.server import BaseHTTPRequestHandler, HTTPServer
import sys


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(
            b'{"key_id":"local-test","key":"0000000000000000000000000000000000000000000000000000000000000000"}'
        )

    def log_message(self, *args):
        pass


server = HTTPServer(("127.0.0.1", 0), Handler)
with open(sys.argv[1], "w", encoding="utf-8") as handle:
    handle.write(str(server.server_port))
server.serve_forever()
PY
pid=$!
trap 'kill "$pid" 2>/dev/null || true; rm -f "$port_file"' EXIT

for _ in $(seq 1 50); do
	if [[ -s "$port_file" ]]; then
		break
	fi
	sleep 0.1
done

if [[ ! -s "$port_file" ]]; then
	echo "local KMS mock did not start" >&2
	exit 1
fi

ONEFACTURE_KMS_URL="http://127.0.0.1:$(cat "$port_file")" bash scripts/verify_external_acceptance.sh kms-broker
