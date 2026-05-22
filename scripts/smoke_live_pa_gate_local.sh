#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

port_file="$(mktemp)"
python3 - "$port_file" <<'PY' &
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import sys
import time


class Handler(BaseHTTPRequestHandler):
    def _json(self, status, payload):
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        if length:
            self.rfile.read(length)
        if self.path != "/invoices":
            self._json(404, {"code": "not_found", "message": "not found"})
            return
        if not self.headers.get("Authorization"):
            self._json(401, {"code": "missing_auth", "message": "missing authorization"})
            return
        self._json(
            200,
            {
                "pa_ref": f"pa-local-{int(time.time() * 1000)}",
                "status": "SUBMITTED",
                "accepted_at": "2026-05-22T00:00:00Z",
            },
        )

    def do_GET(self):
        if self.path.startswith("/invoices/") and self.path.endswith("/status"):
            pa_ref = self.path.split("/")[2]
            self._json(
                200,
                {
                    "pa_ref": pa_ref,
                    "status": "SUBMITTED",
                    "occurred_at": "2026-05-22T00:00:00Z",
                },
            )
            return
        self._json(404, {"code": "not_found", "message": "not found"})

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
	echo "local PA mock did not start" >&2
	exit 1
fi

base_url="http://127.0.0.1:$(cat "$port_file")"
ONEFACTURE_CHORUS_BASE_URL="$base_url" \
	ONEFACTURE_CHORUS_ACCESS_TOKEN="local-token" \
	ONEFACTURE_DOCAPOSTE_BASE_URL="$base_url" \
	ONEFACTURE_DOCAPOSTE_API_TOKEN="local-token" \
	ONEFACTURE_PENNYLANE_BASE_URL="$base_url" \
	ONEFACTURE_PENNYLANE_API_TOKEN="local-token" \
	bash scripts/verify_external_acceptance.sh live-pa
