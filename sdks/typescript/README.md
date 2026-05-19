# onefacture — TypeScript SDK

Generated from the OpenAPI spec at `internal/gateway/openapi/spec.yaml`.

```bash
# Once published:
npm install @onefacture/sdk
```

```ts
import { OnefactureClient } from "@onefacture/sdk";

const client = new OnefactureClient({ apiKey: "ofx_...", baseUrl: "https://api.onefacture.io" });
const invoice = await client.invoices.create({
  profile: "EN16931",
  typeCode: "380",
  number: "INV-0001",
  currency: "EUR",
  issueDate: "2026-03-01",
  seller: { name: "Acme", siren: "732829320", address: { /* … */ } },
  buyer:  { name: "Globex", siren: "552120222", address: { /* … */ } },
  lines: [{ description: "Consulting", quantity: 10, unitCode: "HUR", unitPrice: 150, taxRate: 20, taxCategory: "S" }],
});
```

Status: scaffolding only. Generation pipeline (`openapi-typescript-codegen`) and
npm publish workflow ship with v1.
