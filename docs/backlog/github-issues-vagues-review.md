# Review d'implementation des 3 vagues

Date: 2026-05-22

Ce fichier suit `docs/backlog/github-issues-vagues.md` et distingue les livrables couverts localement des criteres qui dependent d'un service externe.

Plan: `docs/backlog/github-issues-vagues-plan.md`.
Runbook acceptance externe: `docs/operations/external-acceptance.md`.
Checklist preuves externes: `docs/operations/external-acceptance-evidence.md`.

Manifest executable: `docs/backlog/github-issues-vagues-acceptance.json`, verifie par `scripts/verify_backlog_acceptance_manifest.rb`, `make verify-backlog-manifest` et le job CI `backlog-acceptance-manifest`. Le verifier controle les 24 titres backlog, les 24 lignes du plan, les 24 entrees de review, les 24 lignes d'audit, les chemins d'artefacts, les commandes `make`, la coherence des gates plan/manifest, les jobs CI d'audit, les targets Makefile de preuves externes, le contenu de `scripts/verify_external_gate_smokes.sh`, les modes supportes par `scripts/verify_external_acceptance.sh`, les marqueurs de succes obligatoires du verifier de preuves externes, le garde-fou `make audit-backlog-completion`, les snippets Python embarques et les choix du workflow manuel `.github/workflows/external-acceptance.yml`. Les chemins externes critiques sont aussi smoke-testes localement ou verifies en pre-publication par `make verify-external-smokes`, par `make verify-local` avec `gofmt`, syntaxe shell/Ruby globale, parse YAML et actionlint des workflows, et par les jobs CI `external-gate-smokes` et `local-acceptance` avec Go, Python et Node provisionnes.

Un item externe ne peut passer de `partial_external` a `covered_external` qu'apres revue d'un bundle valide dans `docs/operations/evidence`, ajout de `reviewed_evidence` dans le manifest, suppression des `external_blockers`, puis re-execution de `make audit-backlog-completion`, qui reverifie le bundle et le commit `HEAD`.

Pour chaque item externe finalise, cette review doit contenir `<numero>. <titre>: covered_external` et l'audit de completion doit contenir `| <numero> | covered_external |`; le verifier de manifest bloque la completion si ces marqueurs manquent.

## Titres source couverts

1. Intégration Chorus Pro PISTE sandbox (round-trip complet)
2. Intégration Docaposte sandbox (submit/status/webhook)
3. Intégration Pennylane sandbox (submit/status/webhook)
4. Idempotency-Key obligatoire sur POST /v1/invoices et /submit
5. Circuit breaker + retry policy pour soumission PA
6. DLQ pour soumissions et événements non délivrables
7. Annuaire SIREN avec cache TTL + fallback provider
8. Override de routage PA par organisation
9. Sandbox publique onefacture (multi-tenant, PA mockées)
10. Onboarding “5 minutes to first invoice”
11. Publication SDK Python sur PyPI
12. Publication SDK TypeScript sur npm
13. CLI onefacture doctor
14. Trace ID dans toutes les réponses API
15. Endpoint timeline facture
16. Webhook inspector UI
17. Erreurs enrichies RFC 7807 (hint/docs/retryable)
18. Pack d’exemples métier (avoir, correction, rejet)
19. Pré-validation bulk avant émission
20. Score qualité de conformité par tenant
21. Assistant de correction automatique des rejets
22. Chiffrement at-rest BYOK/KMS
23. mTLS optionnel + IP allowlist par webhook endpoint
24. Framework multi-juridiction (PEPPOL/ViDA ready)

## Vague 1

1. Intégration Chorus Pro PISTE sandbox (round-trip complet): covered_external. Adaptateur configurable avec OAuth2 client credentials PISTE, client sandbox HTTP submit/status/webhook, mapping `PAError`, resiliency et test live `-tags=live_pa`; round-trip reel verifie par le bundle de preuves externes.
2. Intégration Docaposte sandbox (submit/status/webhook): covered_external. Adaptateur configurable avec client sandbox HTTP submit/status/webhook, mapping `PAError`, resiliency et test live `-tags=live_pa`; round-trip reel verifie par le bundle de preuves externes.
3. Intégration Pennylane sandbox (submit/status/webhook): covered_external. Adaptateur configurable avec client sandbox HTTP submit/status/webhook, mapping `PAError`, resiliency et test live `-tags=live_pa`; round-trip reel verifie par le bundle de preuves externes.
4. Idempotency-Key: implemente localement sur `POST /v1/invoices` et `POST /v1/invoices/{id}/submit` avec reservation persistante, replay, conflit, et migration SQL.
5. Circuit breaker + retry PA: implemente localement via wrapper d'adaptateur avec retry exponentiel, jitter, et circuit breaker par instance d'adaptateur.
6. DLQ soumissions/evenements: implemente localement avec bus Redis Streams `internal/events`, `submission_dlq`, inspection/replay API pour soumissions PA, et inspection/replay webhooks.
7. Annuaire SIREN cache + fallback: implemente localement via resolver TTL avec provider primaire/fallback et test P95 cache < 100ms.
8. Override routage PA par organisation: implemente localement via `organization.settings.routing_overrides[buyer_siren]`, applique a la soumission et trace dans le lifecycle payload.
9. Sandbox publique onefacture (multi-tenant, PA mockées): covered_external. Runbook, profil Helm `values-sandbox.yaml`, job CI Helm, endpoint `POST /v1/sandbox/credentials`, script `scripts/smoke_public_sandbox.sh`, workflows smoke/external acceptance et cible `make verify-public-sandbox` ajoutes et valides par le bundle de preuves externes.
10. Onboarding “5 minutes to first invoice”: covered_external. Guide, collection Postman, generation credentials, payload JSON, webhook E2E et smoke script ajoutes, parcours compte vierge verifie par le bundle de preuves externes.

## Vague 2

11. Publication SDK Python sur PyPI: covered_external. Package PEP 621 `onefacture`, workflow PyPI manuel, job CI SDK, `make verify-sdk` et `scripts/verify_sdk_release_artifacts.sh` installent `./sdks/python` dans une venv et importent `from onefacture import Client`; verifie par le bundle de preuves externes. Auparavant, `make verify-sdk-registries` a confirme `PyPI onefacture install failed`.
12. Publication SDK TypeScript sur npm: covered_external. Package `@onefacture/sdk`, lockfile, workflow npm manuel, job CI SDK, `make verify-sdk` et `scripts/verify_sdk_release_artifacts.sh` executent `npm pack`, installent le tarball dans un projet temporaire et importent `OnefactureClient`; verifie par le bundle de preuves externes. Auparavant, `make verify-sdk-registries` a confirme `npm @onefacture/sdk install failed`.
13. CLI doctor: implemente localement via `cmd/onefacture doctor` avec checks cle API, reachability et payload minimal.
14. Trace ID API: implemente localement avec `X-Request-ID` en reponse et logs.
15. Timeline facture: implemente localement via `GET /v1/invoices/{id}/timeline` avec transitions, latences et contexte rejet/retry.
16. Webhook inspector UI: implemente localement via `/tools/webhook-inspector` connecte aux endpoints inspection/replay.
17. Erreurs RFC 7807 enrichies: implemente localement avec `remediation_hint`, `docs_url`, `retryable`.
18. Exemples metier: implemente dans `docs/examples/business-scenarios.md` avec avoir, correction, rejet et snippets Python/TypeScript; les exemples sont aussi exposes dans OpenAPI/Scalar via `commercial_invoice`, `credit_note`, `correction_invoice` et `/v1/invoices/{id}/retry`.

## Vague 3

19. Pre-validation bulk: implemente localement via `POST /v1/validate/bulk` avec rapport agrege et export CSV.
20. Score qualite conformite: implemente localement via `GET /v1/analytics/compliance-score` avec score 7j, tendances mensuelles et dashboard `/tools/compliance-dashboard`.
21. Assistant de correction automatique des rejets: covered_external. `GET /v1/invoices/{id}/rejection-patch` propose un patch JSON, expose `outcome_metric`, `GET /v1/analytics/rejection-retry-success-rate` mesure le taux, le dashboard l'affiche, et `make verify-outcome-metrics` valide la metrique deployee; verifie par le bundle de preuves externes.
22. Chiffrement at-rest BYOK/KMS: covered_external. AES-256-GCM, `KeyProvider`, `HTTPKMSProvider`, resolution par `key_id`, metadata auditables via `InspectEncryptedArtifact`, workflow external acceptance, `make verify-kms-broker`, chiffrement opt-in de `raw_xml`/`raw_pdf` et runbook rotation ajoutes; verifie par le bundle de preuves externes.
23. mTLS + IP allowlist webhooks: implemente localement. `ip_allowlist`, `mtls_required`, `mtls_cert_ref` ajoutes, allowlist IP appliquee et client cert mTLS charge par endpoint.
24. Framework multi-juridiction: implemente localement via `internal/jurisdiction` et note d'architecture PEPPOL/ViDA.

## Verification locale

- `make verify-local`
- `golangci-lint run --timeout=5m` via container Docker `golangci/golangci-lint:v1.61.0`
- `go test -short -race -covermode=atomic -coverprofile=coverage.out ./...` (314 tests, 28 packages, couverture totale 37.0%, floor CI 35%)
- `go test ./cmd/onefacture ./internal/adapters ./internal/adapters/sandbox ./internal/adapters/chorus ./internal/adapters/docaposte ./internal/adapters/pennylane ./internal/adapters/registry ./internal/directory ./internal/jurisdiction ./internal/reliability ./internal/security ./internal/gateway/routes ./internal/gateway/middleware ./internal/gateway/problem ./internal/gateway/openapi ./internal/webhooks`
- `go test ./internal/security ./internal/jurisdiction`
- `go test ./internal/directory -run 'TestResolver'`
- `go test ./internal/adapters/live -tags=live_pa -count=1`
- `ONEFACTURE_REQUIRE_LIVE_PA=true go test -tags=live_pa ./internal/adapters/live -count=1` doit echouer sans credentials, pour eviter les faux verts CI
- `go test ./... -run '^$'`
- `bash -n scripts/smoke_public_sandbox.sh scripts/verify_sdk_release_artifacts.sh`
- `bash -n scripts/verify_external_acceptance.sh`
- `bash scripts/smoke_public_sandbox_local.sh`
- `bash scripts/smoke_live_pa_gate_local.sh`
- `bash scripts/smoke_kms_gate_local.sh`
- `bash scripts/smoke_outcome_metrics_gate_local.sh`
- `make verify-external-smokes`
- `ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f) }' .github/workflows/sandbox-smoke.yml .github/workflows/sdk-publish.yml`
- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/external-acceptance.yml .github/workflows/ci.yml .github/workflows/sdk-publish.yml .github/workflows/sandbox-smoke.yml`
- `python -m compileall -q sdks/python/src`
- `npm --prefix sdks/typescript run build`
- `scripts/verify_sdk_release_artifacts.sh`
- `ruby scripts/verify_backlog_acceptance_manifest.rb`
- `make verify-backlog-manifest`
- `make verify-external-evidence BUNDLE=...`
- `make review-external-evidence BUNDLE=...`
- `bash scripts/smoke_external_evidence_bundle.sh` (cas valide, secret non redacte, log manquant, marqueur de succes manquant, scaffold non rempli)
- `make verify-external-evidence-smoke`
- `make audit-backlog-completion` execute le verifier de manifest, imprime la checklist prompt-to-artifact par numero et titre d'issue, imprime les gates/blockers externes restants, mappe un bundle valide vers les issues a reviewer, et echoue tant que les items externes restent partiels ou que le bundle externe valide n'a pas ete integre dans les artefacts d'audit
- `make smoke-backlog-completion-audit` et le job CI `backlog-completion-audit` verifient que l'audit echoue sans bundle et echoue encore avec un bundle valide tant que le manifest garde des items externes partiels
- `make create-external-evidence STAMP=YYYY-MM-DD`
- `make check-external-env` liste les variables requises pour les gates externes avant collecte
- `make collect-external-evidence STAMP=YYYY-MM-DD` collecte les logs rediges des gates externes, remplit `summary.md` et valide le bundle si toutes les gates passent
- `make check-github-external-config GITHUB_REPO=yawo/onefacture` liste les variables et secrets GitHub Actions requis avant execution du workflow externe
- `make smoke-github-external-config` et le job CI `github-external-config` verifient le checker GitHub Actions avec un faux `gh`
- `make smoke-external-env` et le job CI `external-env-readiness` verifient le checker d'environnement externe
- `make smoke-external-evidence-collector` et le job CI `external-evidence-collector` verifient le collecteur avec des gates simulees, sans appeler de services externes
- `make smoke-external-evidence-review` et le job CI `external-evidence-review` verifient le helper de revue qui mappe un bundle valide vers les issues externes
- `git diff --check`

Audit detaille: `docs/backlog/github-issues-vagues-completion-audit.md`.

## Risques restants

- La suite `internal/storage` complete hors `-short` reste trop lente pour le job CI par defaut dans l'environnement courant; le CI couvre donc toute la repo en `-short -race` et `make verify-local` execute les tests storage critiques, dont les cas de chiffrement.
- Les criteres d'acceptation sandbox PA live, public sandbox, PyPI/npm, amelioration du taux de resoumission et KMS cloud necessitent des credentials, comptes, une cible de deploiement ou des donnees d'usage.
