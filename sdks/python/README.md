# onefacture — Python SDK

Generated from the OpenAPI spec at `internal/gateway/openapi/spec.yaml`.

```bash
# Once published:
pip install onefacture
```

```python
from onefacture import Client

client = Client(api_key="ofx_...", base_url="https://api.onefacture.io")
invoice = client.invoices.create({
    "profile": "EN16931",
    "type_code": "380",
    "number": "INV-0001",
    "currency": "EUR",
    "issue_date": "2026-03-01",
    "seller": {"name": "Acme", "siren": "732829320", "address": {...}},
    "buyer":  {"name": "Globex", "siren": "552120222", "address": {...}},
    "lines": [{"description": "Consulting", "quantity": 10, "unit_code": "HUR", "unit_price": 150, "tax_rate": 20, "tax_category": "S"}],
})
```

Status: scaffolding only. Code is generated from the OpenAPI spec via
`openapi-python-client`; the publish workflow lives in `.github/workflows/sdk-publish.yml`
(to be added when the spec is frozen for v1).
