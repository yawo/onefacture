# Exemples metier onefacture

Ces payloads couvrent les cas demandes dans la vague 2: avoir, correction et rejet avec resoumission.

## Avoir

```json
{
  "profile": "EN16931",
  "type_code": "381",
  "number": "AV-2026-0001",
  "currency": "EUR",
  "issue_date": "2026-05-22T00:00:00Z",
  "seller": {
    "name": "Acme SAS",
    "siren": "732829320",
    "address": { "line1": "1 rue Cler", "postal_code": "75007", "city": "Paris", "country_code": "FR" }
  },
  "buyer": {
    "name": "Globex SAS",
    "siren": "552120222",
    "address": { "line1": "2 avenue Foch", "postal_code": "75116", "city": "Paris", "country_code": "FR" }
  },
  "lines": [
    { "description": "Avoir commercial", "quantity": 1, "unit_code": "C62", "unit_price": 250, "tax_rate": 20, "tax_category": "S" }
  ],
  "notes": [{ "subject": "credit_note", "content": "Avoir lie a la facture INV-2026-0007." }]
}
```

## Facture corrective

```json
{
  "profile": "EN16931",
  "type_code": "384",
  "number": "COR-2026-0001",
  "currency": "EUR",
  "issue_date": "2026-05-22T00:00:00Z",
  "buyer_ref": "INV-2026-0007",
  "seller": {
    "name": "Acme SAS",
    "siren": "732829320",
    "address": { "line1": "1 rue Cler", "postal_code": "75007", "city": "Paris", "country_code": "FR" }
  },
  "buyer": {
    "name": "Globex SAS",
    "siren": "552120222",
    "address": { "line1": "2 avenue Foch", "postal_code": "75116", "city": "Paris", "country_code": "FR" }
  },
  "lines": [
    { "description": "Correction prix unitaire", "quantity": 4, "unit_code": "HUR", "unit_price": 125, "tax_rate": 20, "tax_category": "S" }
  ],
  "notes": [{ "subject": "correction", "content": "Corrige le montant HT de la facture initiale." }]
}
```

## Rejet puis resoumission

```bash
curl -X POST "http://localhost:8080/v1/invoices/{invoice_id}/retry" \
  -H "X-API-Key: $ONEFACTURE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"resolution_hint":"SIREN acheteur corrige dans ERP"}'
```

## Snippet Python

```python
import requests

resp = requests.post(
    "http://localhost:8080/v1/invoices?submit=true",
    headers={
        "X-API-Key": "of_test",
        "Idempotency-Key": "invoice-INV-2026-0001",
        "Content-Type": "application/json",
    },
    json={...},
    timeout=30,
)
resp.raise_for_status()
print(resp.json()["id"])
```

## Snippet TypeScript

```ts
const response = await fetch("http://localhost:8080/v1/invoices?submit=true", {
  method: "POST",
  headers: {
    "X-API-Key": "of_test",
    "Idempotency-Key": "invoice-INV-2026-0001",
    "Content-Type": "application/json",
  },
  body: JSON.stringify(invoice),
});

if (!response.ok) throw new Error(await response.text());
console.log(await response.json());
```
