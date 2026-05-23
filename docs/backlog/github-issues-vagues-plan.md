# Plan d'execution - Backlog GitHub 3 vagues

Date: 2026-05-23

Objectif: planifier, implementer et reviewer chaque issue de `docs/backlog/github-issues-vagues.md`.

Ce plan est le point d'ancrage avant implementation. Les preuves finales sont dans `docs/backlog/github-issues-vagues-review.md`, `docs/backlog/github-issues-vagues-completion-audit.md` et le manifest executable `docs/backlog/github-issues-vagues-acceptance.json`.

## Strategie

- Vague 1: fermer les risques marche et fiabilite avec adaptateurs configurables, idempotence, retry/circuit breaker, DLQ, annuaire et routage tenant.
- Vague 2: rendre l'experience developpeur observable et consommable via SDKs, CLI, trace IDs, timeline, inspector webhook, erreurs enrichies et exemples.
- Vague 3: ajouter les differenciateurs entreprise: bulk validation, analytics conformite, assistant rejet, BYOK/KMS, securite webhook avancee et multi-juridiction.
- Separateur local/externe: tout ce qui depend de credentials PA, publication publique, sandbox publique, KMS cloud ou donnees produit est livre avec gate executable mais reste marque externe tant que la preuve live manque.

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
  25. Génération PDF/A-3 wire-complete + délégation sidecar (PDF/A-3 + Factur-X embedding)
 26. Charts Helm de production + observabilité minimale (Prometheus + Grafana + OTel)
 27. Publication SDK automatisée sur GitHub Releases
 28. Adaptateurs supplémentaires (Cegid, Qonto) + extension multi-juridiction

 ## Metadata source couverte

| # | Vague | Labels |
|---|---|---|
| 1 | 1 | adapter, priority:p0, wave:1 |
| 2 | 1 | adapter, priority:p0, wave:1 |
| 3 | 1 | adapter, priority:p0, wave:1 |
| 4 | 1 | api, reliability, priority:p0, wave:1 |
| 5 | 1 | reliability, worker, priority:p0, wave:1 |
| 6 | 1 | infra, reliability, priority:p0, wave:1 |
| 7 | 1 | api, directory, priority:p1, wave:1 |
| 8 | 1 | api, multitenancy, priority:p1, wave:1 |
| 9 | 1 | dx, infra, priority:p0, wave:1 |
| 10 | 1 | dx, docs, priority:p1, wave:1 |
| 11 | 2 | dx, sdk, priority:p1, wave:2 |
| 12 | 2 | dx, sdk, priority:p1, wave:2 |
| 13 | 2 | dx, tooling, priority:p2, wave:2 |
| 14 | 2 | api, observability, priority:p1, wave:2 |
| 15 | 2 | api, observability, priority:p1, wave:2 |
| 16 | 2 | dx, webhooks, priority:p2, wave:2 |
| 17 | 2 | api, ux, priority:p1, wave:2 |
| 18 | 2 | docs, dx, priority:p2, wave:2 |
| 19 | 3 | validation, enterprise, priority:p2, wave:3 |
| 20 | 3 | analytics, enterprise, priority:p2, wave:3 |
| 21 | 3 | automation, ux, priority:p2, wave:3 |
| 22 | 3 | security, compliance, priority:p1, wave:3 |
| 23 | 3 | security, webhooks, priority:p2, wave:3 |
| 24 | 3 | architecture, future, priority:p3, wave:3 |
| 25 | 4 | core, facturx, priority:p0, wave:4 |
| 26 | 4 | infra, observability, priority:p1, wave:4 |
| 27 | 4 | dx, ci, sdk, priority:p1, wave:4 |
| 28 | 4 | adapter, architecture, priority:p2, wave:4 |

## Plan par issue

| # | Plan | Gate |
|---|---|---|
| 1 | Adapter Chorus Pro PISTE configurable, OAuth2 client credentials, submit/status/webhook, mapping erreurs, test live strict. | `make verify-live-pa` |
| 2 | Adapter Docaposte sandbox avec submit/status/webhook et normalisation statuts. | `make verify-live-pa` |
| 3 | Adapter Pennylane sandbox avec auth bearer, submit/status/webhook et mapping erreurs. | `make verify-live-pa` |
| 4 | Persister les cles d'idempotence par organisation et appliquer replay/conflit sur create + submit. | `make verify-local` |
| 5 | Envelopper les adaptateurs avec retry exponentiel, jitter et circuit breaker par PA. | `make verify-local` |
| 6 | Ajouter DLQ soumission et webhooks avec endpoints inspection/replay. | `make verify-local` |
| 7 | Introduire resolver annuaire cache TTL + fallback et test P95 cache. | `make verify-local` |
| 8 | Lire les overrides tenant depuis settings organisation et tracer la decision lifecycle. | `make verify-local` |
| 9 | Fournir profil Helm sandbox, credentials immediats, PA mockees et smoke test public. | `make verify-public-sandbox` |
| 10 | Documenter quickstart 5 minutes avec payload, Postman et smoke webhook. | `make verify-public-sandbox` |
| 11 | Packager SDK Python, workflow publication et verifier installabilite locale/publique. | `make verify-sdk`, `make verify-sdk-registries` |
| 12 | Packager SDK TypeScript, workflow npm et verifier installabilite tarball/publique. | `make verify-sdk`, `make verify-sdk-registries` |
| 13 | Ajouter `onefacture doctor` pour cle API, reachability et payload minimal. | `make verify-local` |
| 14 | Injecter `X-Request-ID` dans reponses et logs. | `make verify-local` |
| 15 | Exposer timeline facture avec transitions, erreurs, retries et latences. | `make verify-local` |
| 16 | Ajouter UI inspector webhook avec tentatives, payloads et replay. | `make verify-local` |
| 17 | Enrichir RFC7807 avec `remediation_hint`, `docs_url`, `retryable`. | `make verify-local` |
| 18 | Ajouter scenarios avoir/correction/rejet et snippets SDK. | `make verify-local` |
| 19 | Ajouter endpoint bulk pre-validation avec rapport agrege et CSV erreurs. | `make verify-local` |
| 20 | Calculer score conformite tenant et dashboard tendances mensuelles. | `make verify-local` |
| 21 | Proposer patch JSON de rejet et mesurer taux de resoumission reussie. | `make verify-outcome-metrics` |
| 22 | Chiffrer raw artifacts via AES-GCM, BYOK/KMS provider, rotation et runbook. | `make verify-kms-broker` |
| 23 | Ajouter mTLS optionnel et IP allowlist par endpoint webhook. | `make verify-local` |
| 24 | Extraire profils/regles pays dans un registry multi-juridiction. | `make verify-local` |
| 25 | Émettre conteneur PDF minimal valide + wiring sidecar pour vrai PDF/A-3 + Factur-X embedding. | `make verify-local` |
| 26 | Étendre Helm avec values-prod, ServiceMonitor et dashboards Grafana de base. | `make verify-local` |
| 27 | Rendre le workflow sdk-publish déclenché par release tag (v*). | `make verify-local` |
| 28 | Ajouter adapters Cegid/Qonto et profils ViDA/PEPPOL dans le registry. | `make verify-local` |

## Ordre d'execution

1. Implementer les fondations partagees: idempotence, reliability wrapper, storage DLQ, request ID et problem details.
2. Brancher les surfaces API/UI/CLI: routes, dashboards, doctor, inspectors et OpenAPI.
3. Ajouter les adaptateurs PA et gates live sans masquer l'absence de credentials.
4. Ajouter SDKs, docs, runbooks et workflows de publication/smoke.
5. Auditer chaque issue avec un manifest executable et separer les preuves locales des preuves externes.
