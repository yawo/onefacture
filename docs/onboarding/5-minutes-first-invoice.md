# 5 minutes to first invoice

## Prerequis

- URL sandbox: `https://sandbox.onefacture.io`
- Cle API de test: `ofx_...`
- Binaire CLI: `onefacture`
- Collection Postman: `docs/onboarding/onefacture.postman_collection.json`

## 1. Diagnostic

```bash
curl -X POST "https://sandbox.onefacture.io/v1/sandbox/credentials" \
  -H "Content-Type: application/json" \
  -d '{"name":"Quickstart sandbox"}'
```

```bash
ONEFACTURE_BASE_URL=https://sandbox.onefacture.io \
ONEFACTURE_API_KEY=ofx_test \
onefacture doctor
```

## 2. Creation et emission mockee

```bash
curl -X POST "https://sandbox.onefacture.io/v1/invoices?submit=true" \
  -H "X-API-Key: ofx_test" \
  -H "Idempotency-Key: first-invoice-001" \
  -H "Content-Type: application/json" \
  -d @docs/examples/commercial-invoice.json
```

## 3. Webhook bout en bout

```bash
curl -X POST "https://sandbox.onefacture.io/v1/webhooks" \
  -H "X-API-Key: ${ONEFACTURE_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/webhooks/onefacture","secret":"replace-with-32-character-secret","events":["invoice.accepted","invoice.rejected"]}'
```

## 4. Verification

```bash
curl -H "X-API-Key: ofx_test" \
  "https://sandbox.onefacture.io/v1/invoices/{invoice_id}/timeline"
```

```bash
curl -H "X-API-Key: ${ONEFACTURE_API_KEY}" \
  "https://sandbox.onefacture.io/v1/webhooks/deliveries?limit=10"
```

Le statut attendu sur sandbox mockee est `SUBMITTED`.
