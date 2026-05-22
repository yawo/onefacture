# Audit de completion - Backlog GitHub 3 vagues

Date: 2026-05-22

Objectif audite: planifier, implementer et reviewer chaque issue de `docs/backlog/github-issues-vagues.md`.

Verdict: non complet au sens strict des criteres d'acceptation externes. Les livrables locaux sont presents pour les 24 items, mais les items PA sandbox live, sandbox publique, publications PyPI/npm et broker KMS cloud demandent encore credentials, hosting ou comptes externes.

Plan: `docs/backlog/github-issues-vagues-plan.md`.
Checklist preuves externes: `docs/operations/external-acceptance-evidence.md`.
Template env externe: `docs/operations/external-acceptance.env.example`.
Matrice fermeture externe: `docs/operations/external-closure-matrix.md`.
Manifest executable: `docs/backlog/github-issues-vagues-acceptance.json`, verifie par `scripts/verify_backlog_acceptance_manifest.rb`, `make verify-backlog-manifest` et le job CI `backlog-acceptance-manifest`. Le verifier controle les 24 titres backlog, les 24 lignes du plan, les 24 entrees de review, les 24 lignes d'audit, les chemins d'artefacts, les commandes `make`, la coherence des gates plan/manifest, les jobs CI d'audit, les targets Makefile de preuves externes, le contenu de `scripts/verify_external_gate_smokes.sh`, les modes supportes par `scripts/verify_external_acceptance.sh`, les marqueurs de succes obligatoires du verifier de preuves externes, le garde-fou `make audit-backlog-completion`, les snippets Python embarques et les choix du workflow manuel `.github/workflows/external-acceptance.yml`. Les chemins externes critiques sont aussi smoke-testes localement ou verifies en pre-publication par `make verify-external-smokes`, par `make verify-local` avec `gofmt`, syntaxe shell/Ruby globale et parse YAML des workflows, et par les jobs CI `external-gate-smokes` et `local-acceptance` avec Go, Python et Node provisionnes.

Le statut final pour une preuve externe revue est `covered_external`. Il exige `reviewed_evidence.bundle`, `reviewed_evidence.commit_sha`, `reviewed_evidence.reviewed_at`, `reviewed_evidence.reviewed_by`, aucun `external_blockers`, et une revalidation du bundle par `make audit-backlog-completion` contre le `HEAD` courant.

Pour chaque item externe finalise, la review doit contenir `<numero>. <titre>: covered_external` et cet audit doit contenir `| <numero> | covered_external |`; `scripts/verify_backlog_acceptance_manifest.rb` bloque la completion si ces marqueurs par issue manquent.

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

## Checklist prompt-to-artifact

| # | Critere explicite | Artefacts inspectes | Verification actuelle | Statut |
|---|---|---|---|---|
| 1 | Chorus Pro PISTE OAuth2, submit, status, erreurs, rejet/retry, round-trip sandbox | `internal/adapters/chorus`, `internal/adapters/sandbox`, `internal/adapters/live`, `.github/workflows/sandbox-smoke.yml`, `.github/workflows/external-acceptance.yml`, `docs/operations/external-acceptance.md`, `make verify-live-pa` | OAuth2 `client_credentials` configurable et teste; `TestClientSubmitAndStatusRoundTrip`, `TestClientUsesOAuthClientCredentials`, `TestClientMapsPAErrorResponse` et `TestClientWebhookDecode` couvrent submit/status/webhook et erreurs `PAError`; test live `-tags=live_pa`; CI live stricte avec `ONEFACTURE_REQUIRE_LIVE_PA=true`; round-trip reel bloque sans credentials PISTE | Partiel externe |
| 2 | Docaposte submit/status/webhook, statuts normalises, tests sandbox | `internal/adapters/docaposte`, `internal/adapters/sandbox`, `internal/adapters/live` | Client sandbox configurable par env, bearer auth, statuts core et erreurs `PAError`; `TestNewConfiguresSandboxClientFromEnv` et les tests sandbox client couvrent la config locale; test live strict possible via `ONEFACTURE_REQUIRE_LIVE_PA=true`; credentials requis | Partiel externe |
| 3 | Pennylane auth securisee, erreurs/etats, round-trip automatise | `internal/adapters/pennylane`, `internal/adapters/sandbox`, `internal/adapters/live` | Client sandbox configurable par env, bearer auth, statuts core et erreurs `PAError`; `TestNewConfiguresSandboxClientFromEnv` et les tests sandbox client couvrent la config locale; test live strict possible via `ONEFACTURE_REQUIRE_LIVE_PA=true`; credentials requis | Partiel externe |
| 4 | `Idempotency-Key` obligatoire sur creation invoice et submit, dedoublonnage persistant | `internal/storage/idempotency.go`, `internal/storage/migrations/0001_init.up.sql`, `internal/gateway/routes/routes.go`, OpenAPI, `internal/gateway/routes/handlers_test.go` | API reserve/replay/conflit; migration table `idempotency_keys`; `TestIdempotencyKeyIsRequired` couvre l'obligation du header; tests routes/storage inclus dans suite cible | Couvert localement |
| 5 | Circuit breaker + retry exponentiel/jitter par PA | `internal/reliability`, `internal/adapters/registry/registry.go`, `internal/reliability/adapter_test.go` | Wrapper de registry applique retry/circuit breaker; `TestAdapterRetriesSubmitUntilSuccess` et `TestAdapterOpensCircuitAfterFailures` couvrent retry puis circuit ouvert | Couvert localement |
| 6 | DLQ soumissions/evenements, inspection et replay manuel | `internal/events/bus.go`, `internal/storage/submissions.go`, `internal/storage/webhooks.go`, `internal/gateway/routes/routes.go` | Bus Redis Streams pour evenements; APIs DLQ soumission et webhook inspection/replay; migration `submission_dlq` | Couvert localement |
| 7 | Annuaire SIREN cache TTL + fallback provider, P95 cache <100ms | `internal/directory`, `DirectoryLookup` | Resolver TTL/fallback teste; `TestResolverCachedLookupP95Under100ms` mesure le chemin cache en memoire sous 100ms P95 | Couvert localement |
| 8 | Override routage PA par organisation, applique et auditable | `resolvePAID`, `storage.Organization.Settings`, lifecycle payload, `internal/gateway/routes/handlers_test.go` | `TestResolvePAIDUsesBuyerOverride` couvre `routing_overrides[buyer_siren]`; submit trace l'override dans payload lifecycle | Couvert localement |
| 9 | Sandbox publique multi-tenant, PA mockees, credentials immediats, quickstart <10 min | `POST /v1/sandbox/credentials`, `docs/sandbox/public-sandbox.md`, `deploy/helm/onefacture/values-sandbox.yaml`, CI `helm-sandbox`, `scripts/smoke_public_sandbox.sh`, `.github/workflows/sandbox-smoke.yml`, `.github/workflows/external-acceptance.yml`, `docs/operations/external-acceptance.md`, `make verify-public-sandbox` | Generation credentials + profil Helm sandbox + CI render + smoke test deploy-ready; aucune URL publique verifiee | Partiel externe |
| 10 | Onboarding 5 minutes, copy/paste, Postman, webhook E2E, compte vierge verifie | `docs/onboarding/5-minutes-first-invoice.md`, `docs/onboarding/onefacture.postman_collection.json`, `docs/examples`, smoke script | Parcours documente avec collection Postman et webhook E2E; compte vierge non teste sur sandbox publique | Partiel externe |
| 11 | SDK Python publie PyPI, `pip install onefacture` fonctionnel | `sdks/python`, `.github/workflows/sdk-publish.yml`, `.github/workflows/external-acceptance.yml`, CI `sdk-artifacts`, `make verify-sdk`, `make verify-sdk-registries` | Package PEP 621 `onefacture` + workflow PyPI; verifier pre-publish installe `./sdks/python` dans une venv et importe `from onefacture import Client`; `make verify-sdk-registries` tente `pip install onefacture` et a confirme le 2026-05-22 `PyPI onefacture install failed`; publication PyPI requise | Partiel externe |
| 12 | SDK TypeScript publie npm, `npm install @onefacture/sdk` fonctionnel | `sdks/typescript`, lockfile, `.github/workflows/sdk-publish.yml`, `.github/workflows/external-acceptance.yml`, CI `sdk-artifacts`, `make verify-sdk`, `make verify-sdk-registries` | Package `@onefacture/sdk`; verifier pre-publish execute `npm pack`, installe le tarball dans un projet temporaire et importe `OnefactureClient`; `make verify-sdk-registries` tente `npm install @onefacture/sdk` et a confirme le 2026-05-22 `npm @onefacture/sdk install failed`; publication npm requise | Partiel externe |
| 13 | CLI `onefacture doctor`: cle API, reachability, schema payload minimal | `cmd/onefacture`, Makefile | Tests CLI doctor verts dans suite cible, dont `TestFormatDoctorReportShowsClearTerminalStatus` pour le rapport terminal | Couvert localement |
| 14 | Trace ID toutes reponses + logs | `internal/gateway/middleware/request_id.go`, `logging.go`, `server.go`, `internal/gateway/middleware/middleware_test.go` | Middleware expose `X-Request-ID`; `TestAccessLogIncludesRequestID` verifie la correlation logs via `request_id=` | Couvert localement |
| 15 | Endpoint timeline facture: transitions, erreurs, retries, latences | `InvoiceTimeline`, `buildTimeline`, `internal/gateway/routes/handlers_test.go` | Endpoint `GET /v1/invoices/{id}/timeline`; `TestBuildTimelineIncludesLatencyAndRejectionRetry` couvre latence, rejet et retry | Couvert localement |
| 16 | Webhook inspector UI avec tentatives, codes, payloads, replay one-click | `/tools/webhook-inspector`, webhook delivery APIs, `internal/gateway/routes/handlers_test.go` | UI HTML et endpoints inspection/replay; test rendu UI assert le bouton `Replay` et l'appel `/v1/webhooks/deliveries/{id}/replay` | Couvert localement |
| 17 | RFC7807 enrichi: hint, docs_url, retryable sur erreurs top | `internal/gateway/problem`, `internal/gateway/problem/problem_test.go` | Champs et defaults ajoutes; `TestTopErrorHelpersHaveActionableEnrichment` couvre les helpers d'erreurs top avec hint/docs/retryable | Couvert localement |
| 18 | Exemples avoir, correction, rejet + snippets SDK/docs interactives | `docs/examples/business-scenarios.md`, SDK READMEs, `internal/gateway/openapi/spec.yaml`, `internal/gateway/openapi/openapi_test.go` | Scenarios et snippets presents; OpenAPI/Scalar expose `commercial_invoice`, `credit_note`, `correction_invoice` et `/v1/invoices/{id}/retry`; test OpenAPI verifie ces marqueurs | Couvert localement |
| 19 | Pre-validation bulk, rapport agrege + CSV erreurs | `ValidateBulk`, OpenAPI, `internal/gateway/routes/handlers_test.go` | Endpoint JSON + export CSV; tests route `TestValidateBulkReturnsAggregateReport` et `TestValidateBulkExportsCSVErrors` | Couvert localement |
| 20 | Score conformite hebdo, dashboard score + tendances mensuelles | `ComplianceScore`, `/tools/compliance-dashboard`, `internal/gateway/routes/handlers_test.go` | API + dashboard HTML affichant score, tendances mensuelles et retry success rate; test rendu UI assert `monthly_trends` et `Tendances mensuelles` | Couvert localement |
| 21 | Assistant correction rejets, patch JSON pret a resoumettre, amelioration taux | `RejectionPatch`, `suggestRejectionPatch`, `GET /v1/analytics/rejection-retry-success-rate`, `/tools/compliance-dashboard`, `.github/workflows/external-acceptance.yml`, `make verify-outcome-metrics` | Endpoint patch JSON suggere avec `outcome_metric`; `TestSuggestRejectionPatchForSIREN` et `TestBuildRejectionRetrySuccessRate` couvrent patch + metrique locale; verifier externe controle la metrique et un volume minimal; amelioration taux non prouvable sans donnees prod | Partiel outcome externe |
| 22 | Chiffrement at-rest BYOK/KMS, rotation, runbooks, donnees chiffrees et auditables | `internal/security`, `HTTPKMSProvider`, `storage.InvoiceRepo`, `InspectEncryptedArtifact`, `docs/security/byok-kms-runbook.md`, `.github/workflows/external-acceptance.yml`, `make verify-kms-broker` | AES-GCM + provider HTTP KMS + rotation `key_id` testes par `TestEncryptorDecryptsOldEnvelopeAfterRotation` et `TestHTTPKMSProviderRoundTripAndRotation`; metadata `encrypted/key_id/field` inspectable sans dechiffrement via `InspectEncryptedArtifact`; verifier externe controle `/keys/active`; broker KMS cloud/audit prod externe | Partiel externe |
| 23 | mTLS optionnel + IP allowlist webhook, handshake et logs enrichis | `internal/storage/webhooks.go`, `internal/webhooks/deliverer.go`, `internal/webhooks/deliverer_test.go` | Champs config, allowlist IP et client cert mTLS charges; `TestClientForEndpointPerformsMTLSHandshake` valide un handshake mTLS local avec certificat client | Couvert localement |
| 24 | Framework multi-juridiction PEPPOL/ViDA, nouveau profil sans toucher core API | `internal/jurisdiction`, `docs/architecture/multi-jurisdiction.md`, `internal/jurisdiction/registry_test.go` | Registry juridiction/profils + `TestRegistryCanAddJurisdictionWithoutCoreAPIChange` | Couvert localement |

## Criteres d'acceptation source couverts

| # | Critere source | Statut |
|---|---|---|
| 1 | Round-trip sur sandbox Chorus validé end-to-end. Tests d'intégration automatisés. | Partiel externe |
| 2 | Tests d'intégration verts sur sandbox. | Partiel externe |
| 3 | Round-trip complet automatisé. | Partiel externe |
| 4 | Rejeu même clé => même résultat, sans duplicat invoice. | Couvert localement |
| 5 | Dégradation contrôlée en cas PA indisponible. | Couvert localement |
| 6 | Message en échec terminal disponible pour replay manuel. | Couvert localement |
| 7 | Latence lookup P95 < 100ms en cache. | Couvert localement |
| 8 | Règles appliquées et auditables. | Couvert localement |
| 9 | Un développeur externe exécute quickstart en < 10 min. | Partiel externe |
| 10 | Parcours vérifié sur compte vierge. | Partiel externe |
| 11 | `pip install onefacture` fonctionnel. | Partiel externe |
| 12 | `npm install @onefacture/sdk` fonctionnel. | Partiel externe |
| 13 | Rapport diagnostic clair en sortie terminal. | Couvert localement |
| 14 | Corrélation requête/logs de bout en bout. | Couvert localement |
| 15 | Timeline complète pour toute facture non terminale. | Couvert localement |
| 16 | Replay one-click d’un delivery échoué. | Couvert localement |
| 17 | 90% des erreurs top disposent d’un hint exploitable. | Couvert localement |
| 18 | Scénarios couverts dans docs interactives. | Couvert localement |
| 19 | Rapport agrégé + export CSV erreurs. | Couvert localement |
| 20 | Dashboard score + tendances mensuelles. | Couvert localement |
| 21 | Taux de re-soumission réussie amélioré. | Partiel outcome externe |
| 22 | Données sensibles chiffrées et auditables. | Partiel externe |
| 23 | Handshake mTLS validé et logs d’accès enrichis. | Couvert localement |
| 24 | Ajout d’un nouveau profil pays sans toucher au core API. | Couvert localement |

## Descriptions source couvertes

| # | Description source | Statut |
|---|---|---|
| 1 | Implémenter OAuth2 client credentials PISTE. | Partiel externe |
| 1 | Submit facture + récupération statut + mapping erreurs. | Partiel externe |
| 1 | Couvrir cas rejet et retry. | Partiel externe |
| 2 | Implémenter submit/status sur endpoint Docaposte. | Partiel externe |
| 2 | Normaliser les statuts vers le modèle onefacture. | Partiel externe |
| 3 | Implémenter connecteur Pennylane avec auth sécurisée. | Partiel externe |
| 3 | Mapper erreurs et états. | Partiel externe |
| 4 | Support header `Idempotency-Key`. | Couvert localement |
| 4 | Dédoublonnage persistant par organisation. | Couvert localement |
| 5 | Ajouter circuit breaker par adaptateur. | Couvert localement |
| 5 | Retry exponentiel + jitter + limite max. | Couvert localement |
| 6 | Créer queue de dead-letter (Redis Streams). | Couvert localement |
| 6 | Endpoint d’inspection et replay. | Couvert localement |
| 7 | Résolution PA par SIREN avec cache. | Couvert localement |
| 7 | Fallback si provider primaire down. | Couvert localement |
| 8 | Permettre une règle tenant: destination -> PA forcée. | Couvert localement |
| 9 | Déployer instance accessible publiquement. | Partiel externe |
| 9 | Génération de credentials de test immédiats. | Partiel externe |
| 10 | Tutoriel copy/paste + collection Postman. | Partiel externe |
| 10 | Exemple webhook de bout en bout. | Partiel externe |
| 11 | Pipeline génération + publication versionnée. | Partiel externe |
| 12 | Pipeline génération + publication npm. | Partiel externe |
| 13 | Vérifier clé API, reachability, schéma payload minimal. | Couvert localement |
| 14 | Injecter `X-Request-ID` dans logs + réponses. | Couvert localement |
| 15 | Exposer transitions, erreurs, retries, latences. | Couvert localement |
| 16 | Vue des tentatives, codes HTTP, payloads, replay. | Couvert localement |
| 17 | Champs `remediation_hint`, `docs_url`, `retryable`. | Couvert localement |
| 18 | Exemples JSON + snippets SDKs. | Couvert localement |
| 19 | Endpoint bulk pour analyser des lots de factures. | Couvert localement |
| 20 | Calcul score hebdo (rejets, erreurs, correction speed). | Couvert localement |
| 21 | Proposer patch JSON prêt à resoumettre. | Partiel outcome externe |
| 22 | Intégration KMS + rotation clés + runbooks. | Partiel externe |
| 23 | Sécurisation avancée des webhooks sortants. | Couvert localement |
| 24 | Abstraire règles pays/profils vers modules. | Couvert localement |

## Verification executee

- `make verify-local`
- `golangci-lint run --timeout=5m` via container Docker `golangci/golangci-lint:v1.61.0`
- `go test -short -race -covermode=atomic -coverprofile=coverage.out ./...` (314 tests, 28 packages, couverture totale 37.0%, floor CI 35%)
- `go test ./cmd/onefacture ./internal/adapters ./internal/adapters/mock ./internal/adapters/sandbox ./internal/adapters/chorus ./internal/adapters/docaposte ./internal/adapters/pennylane ./internal/adapters/registry ./internal/config ./internal/core/facturx ./internal/core/invoice ./internal/directory ./internal/events ./internal/gateway ./internal/jurisdiction ./internal/reliability ./internal/security ./internal/validation ./internal/gateway/routes ./internal/gateway/middleware ./internal/gateway/problem ./internal/gateway/openapi ./internal/webhooks ./internal/workers`
- `go test -short ./internal/storage`
- `go test ./internal/storage -run 'Test(InvoiceRepoEncryptsAndDecryptsArtifacts|InvoiceRepoLeavesArtifactsPlainWithoutEncryptor)'`
- `go test ./internal/directory -run 'TestResolver'`
- `go test ./internal/adapters/live -tags=live_pa -count=1`
- `ONEFACTURE_REQUIRE_LIVE_PA=true go test -tags=live_pa ./internal/adapters/live -count=1` verifie que les credentials manquants font echouer le mode strict
- `go test ./... -run '^$'`
- `bash -n scripts/smoke_public_sandbox.sh scripts/verify_sdk_release_artifacts.sh`
- `bash -n scripts/verify_external_acceptance.sh`
- `bash scripts/smoke_public_sandbox_local.sh`
- `bash scripts/smoke_live_pa_gate_local.sh`
- `bash scripts/smoke_kms_gate_local.sh`
- `bash scripts/smoke_outcome_metrics_gate_local.sh`
- `make verify-external-smokes`
- `scripts/verify_external_acceptance.sh live-pa` sans env verifie que le preflight echoue explicitement sur credentials manquants
- `scripts/verify_external_acceptance.sh kms-broker` sans env verifie que le preflight echoue explicitement sur KMS manquant
- `scripts/verify_external_acceptance.sh outcome-metrics` sans env verifie que le preflight echoue explicitement sur API prod manquante
- `ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f) }' .github/workflows/sandbox-smoke.yml .github/workflows/sdk-publish.yml`
- `ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_file(f) }' .github/workflows/ci.yml deploy/helm/onefacture/values-sandbox.yaml`
- `scripts/verify_sdk_release_artifacts.sh` via `make verify-external-smokes`
- `ruby scripts/verify_backlog_acceptance_manifest.rb`
- `make verify-backlog-manifest`
- `make verify-local` verifie aussi les smokes externes locaux et les checks SDK pre-publication via `make verify-external-smokes`
- `make verify-external-evidence BUNDLE=...` valide la structure et les champs obligatoires d'un bundle de preuves externes
- `make review-external-evidence BUNDLE=...` mappe un bundle valide vers les issues externes a reviewer
- `bash scripts/smoke_external_evidence_bundle.sh` verifie le cas valide, le rejet secret non redacte, le rejet log manquant, le rejet marqueur de succes manquant et le rejet scaffold non rempli
- `make verify-external-evidence-smoke` couvre le verifier de bundle en local et via le job CI `external-evidence-verifier`
- `make audit-backlog-completion` execute le verifier de manifest, imprime la checklist prompt-to-artifact par numero et titre d'issue, imprime les gates/blockers externes restants, mappe un bundle valide vers les issues a reviewer, et echoue tant que les items externes restent partiels ou que le bundle externe valide n'a pas ete integre dans les artefacts d'audit
- `make smoke-backlog-completion-audit` et le job CI `backlog-completion-audit` verifient que l'audit echoue sans bundle et echoue encore avec un bundle valide tant que le manifest garde des items externes partiels
- `make create-external-evidence STAMP=YYYY-MM-DD` cree le squelette de preuves externes attendu
- `make check-external-env` liste les variables requises pour les gates externes avant collecte
- `make collect-external-evidence STAMP=YYYY-MM-DD` collecte les logs rediges des gates externes, remplit `summary.md` et valide le bundle si toutes les gates passent
- `make smoke-external-env` et le job CI `external-env-readiness` verifient le checker d'environnement externe
- `make smoke-external-evidence-collector` et le job CI `external-evidence-collector` verifient le collecteur avec des gates simulees, sans appeler de services externes
- `make smoke-external-evidence-review` et le job CI `external-evidence-review` verifient le helper de revue qui mappe un bundle valide vers les issues externes
- `git diff --check`

## Manques restants

- Credentials PISTE, Docaposte et Pennylane pour valider `make verify-live-pa`; le round-trip submit/status est seulement smoke-teste localement par `scripts/smoke_live_pa_gate_local.sh`.
- URL sandbox publique et verification d'un compte vierge par `make verify-public-sandbox`; le flow quickstart est seulement smoke-teste localement par `scripts/smoke_public_sandbox_local.sh`.
- Publication reelle de `onefacture` sur PyPI et `@onefacture/sdk` sur npm, puis `make verify-sdk-registries`.
- Broker KMS cloud reel et audit de rotation en environnement de production, puis `make verify-kms-broker`.
- Donnees produit pour prouver l'amelioration effective du taux de resoumission apres assistant de correction, puis `make verify-outcome-metrics`.
