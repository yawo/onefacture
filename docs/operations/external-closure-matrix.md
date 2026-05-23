# External closure matrix

Date: 2026-05-23

This matrix is the issue-by-issue handoff for the acceptance criteria that cannot be closed by local implementation alone. Use it after `make verify-local` is green.

| Issue | Gate | External prerequisite | Required evidence | Completion update |
|---|---|---|---|---|
| 1. Intégration Chorus Pro PISTE sandbox (round-trip complet) | `make verify-live-pa` | Chorus PISTE sandbox base URL and access token. | `live-pa.log` includes the Chorus subtest, submit success, non-empty PA ref, status retrieval, rejection mapping and retry path. | Mark issue 1 `covered_external` with reviewed evidence after bundle review. |
| 2. Intégration Docaposte sandbox (submit/status/webhook) | `make verify-live-pa` | Docaposte sandbox base URL and API token. | `live-pa.log` includes the Docaposte subtest, submit/status success, webhook decode path and normalized status mapping. | Mark issue 2 `covered_external` with reviewed evidence after bundle review. |
| 3. Intégration Pennylane sandbox (submit/status/webhook) | `make verify-live-pa` | Pennylane sandbox base URL and API token. | `live-pa.log` includes the Pennylane subtest, secure auth path, submit/status success and error/state mapping. | Mark issue 3 `covered_external` with reviewed evidence after bundle review. |
| 9. Sandbox publique onefacture (multi-tenant, PA mockées) | `make verify-public-sandbox` | Public sandbox deployment URL reachable by an external developer. | `public-sandbox.log` includes `/healthz`, immediate sandbox credential creation, invoice submit and timeline checks from the public URL. | Mark issue 9 `covered_external` with reviewed evidence after bundle review. |
| 10. Onboarding “5 minutes to first invoice” | `make verify-public-sandbox` | Fresh account or clean sandbox credential on the public deployment. | `public-sandbox.log` plus evidence links show the quickstart was executed from a clean credential path in under 10 minutes, including webhook setup. | Mark issue 10 `covered_external` with reviewed evidence after bundle review. |
| 11. Publication SDK Python sur PyPI | `make verify-sdk-registries` | Published public PyPI package `onefacture`. | `sdk-registries.log` includes a fresh `pip install onefacture`, import smoke and `PyPI onefacture install ok`; evidence links include the PyPI package page. | Mark issue 11 `covered_external` with reviewed evidence after bundle review. |
| 12. Publication SDK TypeScript sur npm | `make verify-sdk-registries` | Published public npm package `@onefacture/sdk`. | `sdk-registries.log` includes a fresh `npm install @onefacture/sdk`, Node import smoke and `npm @onefacture/sdk install ok`; evidence links include the npm package page. | Mark issue 12 `covered_external` with reviewed evidence after bundle review. |
| 21. Assistant de correction automatique des rejets | `make verify-outcome-metrics` | Production or representative product analytics API with retry outcome data and `ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE`. | `outcome-metrics.log` includes `retried_invoices >= ONEFACTURE_MIN_RETRIED_INVOICES`, `success_rate > ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE`, accepted-after-retry count, success rate, baseline rate and `outcome metric ok`. | Mark issue 21 `covered_external` with reviewed evidence after bundle review. |
| 22. Chiffrement at-rest BYOK/KMS | `make verify-kms-broker` | Deployed KMS broker with active-key endpoint and rotation/audit record. | `kms-broker.log` includes active key id, decoded 32-byte key validation and `KMS active key ok`; `summary.md` links the rotation/audit record. | Mark issue 22 `covered_external` with reviewed evidence after bundle review. |

Closure sequence:

1. Fill real values from `docs/operations/external-acceptance.env.example`.
2. Run `make check-external-env`.
3. Run `make collect-external-evidence STAMP=YYYY-MM-DD`.
4. Run `make review-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance`.
5. Update `docs/backlog/github-issues-vagues-acceptance.json`, `docs/backlog/github-issues-vagues-review.md` and `docs/backlog/github-issues-vagues-completion-audit.md` using the review helper output.
6. Run `make audit-backlog-completion BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance`.
