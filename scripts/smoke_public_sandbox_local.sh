#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

port_file="$(mktemp)"
python3 - "$port_file" <<'PY' &
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import sys


class Handler(BaseHTTPRequestHandler):
    def _json(self, status, payload):
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(204)
            self.end_headers()
            return
        if self.path == "/v1/invoices/inv_local/timeline":
            self._json(200, {"timeline": [{"status": "SUBMITTED", "latency_ms": 12}]})
            return
        self._json(404, {"error": "not found"})

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        if length:
            self.rfile.read(length)
        if self.path == "/v1/sandbox/credentials":
            self._json(200, {"api_key": "ofx_local_sandbox"})
            return
        if self.path == "/v1/invoices?submit=true":
            if not self.headers.get("X-API-Key"):
                self._json(401, {"error": "missing api key"})
                return
            if not self.headers.get("Idempotency-Key"):
                self._json(400, {"error": "missing idempotency key"})
                return
            self._json(200, {"id": "inv_local"})
            return
        self._json(404, {"error": "not found"})

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
	echo "local public sandbox mock did not start" >&2
	exit 1
fi

ONEFACTURE_SANDBOX_URL="http://127.0.0.1:$(cat "$port_file")" bash scripts/smoke_public_sandbox.sh
