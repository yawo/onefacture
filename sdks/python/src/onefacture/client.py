from __future__ import annotations

import uuid
from typing import Any

import httpx


class Client:
    def __init__(self, api_key: str, base_url: str = "https://api.onefacture.io") -> None:
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.http = httpx.Client(timeout=30)

    def create_invoice(self, invoice: dict[str, Any], submit: bool = False, idempotency_key: str | None = None) -> dict[str, Any]:
        headers = self._headers(idempotency_key)
        response = self.http.post(f"{self.base_url}/v1/invoices", params={"submit": str(submit).lower()}, json=invoice, headers=headers)
        response.raise_for_status()
        return response.json()

    def retry_invoice(self, invoice_id: str, resolution_hint: str = "") -> dict[str, Any]:
        response = self.http.post(
            f"{self.base_url}/v1/invoices/{invoice_id}/retry",
            json={"resolution_hint": resolution_hint},
            headers=self._headers(),
        )
        response.raise_for_status()
        return response.json()

    def _headers(self, idempotency_key: str | None = None) -> dict[str, str]:
        headers = {"X-API-Key": self.api_key, "Content-Type": "application/json"}
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        else:
            headers["Idempotency-Key"] = str(uuid.uuid4())
        return headers
