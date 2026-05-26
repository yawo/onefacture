## Goal
- Complete the two selected micro-étapes (Chorus real PISTE adapter with mapping/normalize + shape transformers + integ test; sidecar PDF full lines + multi-page + deeper validation) and address review suggestions.

## Constraints & Preferences
- Work strictly in small parallel micro-étapes
- Follow Cegid/Qonto patterns (map*Error + NormalizeLifecycleStatus)
- Reuse sandbox.Client for HTTP layer
- No new comments added to code
- Restore via `git checkout --` on import/build errors
- Keep sidecar optional via env-var delegation

## Progress
### Done
- Implemented metrics histogram AdapterCallDuration and gauge InFlightInvoices, wired in reliability wrapper and status poller
- Added httptest integration tests for Cegid and Qonto normalization
- Extended sidecar: added TaxBreakdown model + TVA rendering in sidecar/pdf/main.py; removed [:5] limit + added showPage() multi-page + _draw_header helper; updated pdf.go to send full Lines + TaxBreakdown; enhanced pdf_sidecar_test.go with deeper asserts
- Started real Chorus: added mapChorusError (extended) + NormalizeLifecycleStatus in chorus.go; wired to Submit/GetStatus; added httptest integ test
- Fixed review suggestions: header redraw via _draw_header; extended error codes; improved Chorus test
- Fixed pre-existing webhooks imports, sidecar PDF %PDF assert, layout y-overlap
- Completed Chorus real PISTE shape handling: added StatusMethod + StatusBodyTemplate to sandbox.Client (POST support for /consulter/fournisseur); added UnmarshalJSON aliases on SubmitResult/LifecycleEvent (identifiantFactureCPP, statutFacture, statutCourantCode -> pa_ref/status); wired real paths + method + template in chorus.New when baseURL is production PISTE; added TestChorusIntegrationRealPISTEShapes covering PISTE JSON roundtrips + Normalize
- Added Chorus PDF multipart upload support in sandbox.Client.Submit (DEPOT_PDF_API mode): multipart/form-data with invoice JSON + file PDF when RawPDF present; added TestChorusSubmitWithPDFMultipart
- Added integration tests for Docaposte (TestDocaposteIntegrationSubmitAndGetStatus) and Pennylane (TestPennylaneIntegrationSubmitAndGetStatus) covering Submit/GetStatus roundtrips
- All go test (adapters + chorus + sandbox + race), go build, go vet green; verify-local subset passed

### In Progress
- (none)

### Blocked
- (none)

## Key Decisions
- Use alias UnmarshalJSON on shared result types (no reflection, minimal impact on other adapters)
- Enhance sandbox.Client with StatusMethod/StatusBodyTemplate (keeps thin chorus wrapper)
- Detect real PISTE via baseURL substring (env still wins for paths)
- Test real shapes via direct Client + httptest (covers decode + POST body + Normalize)

## Next Steps
- Emit remaining metrics (DLQ query accuracy, compliance gauge, more call sites)
- Add more adapter integration tests (Docaposte/Pennylane)
- Run full `make verify-local` when env ready + golangci-lint

## Critical Context
- Chorus PISTE real responses use "identifiantFactureCPP" (number or string), "statutFacture", "statutCourantCode"; GetStatus often POST JSON body; Submit for DEPOT_PDF_API uses multipart/form-data with file
- Sandbox Client Submit detects RawPDF for multipart; default JSON mode for other modes (SAISIE_API)
- Reportlab emits %PDF-1.3
- Review via /local-review-uncommitted addressed (header, errors, tests)
- Pre-existing comments untouched

## Relevant Files
- internal/adapters/types.go: UnmarshalJSON for SubmitResult + LifecycleEvent (PISTE key aliases)
- internal/adapters/sandbox/client.go: StatusMethod + StatusBodyTemplate fields + GetStatus POST/GET logic + multipart Submit for RawPDF
- internal/adapters/chorus/chorus.go: real PISTE wiring in New (paths, method, template for api.piste.gouv.fr)
- internal/adapters/chorus/chorus_test.go: TestChorusIntegrationRealPISTEShapes, TestChorusSubmitWithPDFMultipart
- internal/metrics/metrics.go, internal/reliability/adapter.go, internal/workers/status_poller.go (prior metrics)
- sidecar/pdf/main.py, internal/core/facturx/pdf.go + pdf_sidecar_test.go (prior sidecar)
- docs/backlog/github-issues-vagues.md: Vague 1 items 1/4/5 (Chorus/Idempotency/reliability) still reference
- scripts/verify_local_acceptance.sh (invoked for full gates)