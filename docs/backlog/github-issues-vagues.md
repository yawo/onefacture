# Backlog GitHub — 3 vagues (prêt à créer)

## Vague 1 — Must-have marché

### 1. Intégration Chorus Pro PISTE sandbox (round-trip complet)
**Labels**: `adapter`, `priority:p0`, `wave:1`
**Description**:
- Implémenter OAuth2 client credentials PISTE.
- Submit facture + récupération statut + mapping erreurs.
- Couvrir cas rejet et retry.
**Critères d'acceptation**:
- Round-trip sur sandbox Chorus validé end-to-end.
- Tests d'intégration automatisés.

### 2. Intégration Docaposte sandbox (submit/status/webhook)
**Labels**: `adapter`, `priority:p0`, `wave:1`
**Description**:
- Implémenter submit/status sur endpoint Docaposte.
- Normaliser les statuts vers le modèle onefacture.
**Critères d'acceptation**:
- Tests d'intégration verts sur sandbox.

### 3. Intégration Pennylane sandbox (submit/status/webhook)
**Labels**: `adapter`, `priority:p0`, `wave:1`
**Description**:
- Implémenter connecteur Pennylane avec auth sécurisée.
- Mapper erreurs et états.
**Critères d'acceptation**:
- Round-trip complet automatisé.

### 4. Idempotency-Key obligatoire sur POST /v1/invoices et /submit
**Labels**: `api`, `reliability`, `priority:p0`, `wave:1`
**Description**:
- Support header `Idempotency-Key`.
- Dédoublonnage persistant par organisation.
**Critères d'acceptation**:
- Rejeu même clé => même résultat, sans duplicat invoice.

### 5. Circuit breaker + retry policy pour soumission PA
**Labels**: `reliability`, `worker`, `priority:p0`, `wave:1`
**Description**:
- Ajouter circuit breaker par adaptateur.
- Retry exponentiel + jitter + limite max.
**Critères d'acceptation**:
- Dégradation contrôlée en cas PA indisponible.

### 6. DLQ pour soumissions et événements non délivrables
**Labels**: `infra`, `reliability`, `priority:p0`, `wave:1`
**Description**:
- Créer queue de dead-letter (Redis Streams).
- Endpoint d’inspection et replay.
**Critères d'acceptation**:
- Message en échec terminal disponible pour replay manuel.

### 7. Annuaire SIREN avec cache TTL + fallback provider
**Labels**: `api`, `directory`, `priority:p1`, `wave:1`
**Description**:
- Résolution PA par SIREN avec cache.
- Fallback si provider primaire down.
**Critères d'acceptation**:
- Latence lookup P95 < 100ms en cache.

### 8. Override de routage PA par organisation
**Labels**: `api`, `multitenancy`, `priority:p1`, `wave:1`
**Description**:
- Permettre une règle tenant: destination -> PA forcée.
**Critères d'acceptation**:
- Règles appliquées et auditables.

### 9. Sandbox publique onefacture (multi-tenant, PA mockées)
**Labels**: `dx`, `infra`, `priority:p0`, `wave:1`
**Description**:
- Déployer instance accessible publiquement.
- Génération de credentials de test immédiats.
**Critères d'acceptation**:
- Un développeur externe exécute quickstart en < 10 min.

### 10. Onboarding “5 minutes to first invoice”
**Labels**: `dx`, `docs`, `priority:p1`, `wave:1`
**Description**:
- Tutoriel copy/paste + collection Postman.
- Exemple webhook de bout en bout.
**Critères d'acceptation**:
- Parcours vérifié sur compte vierge.

## Vague 2 — Wow Dev Experience

### 11. Publication SDK Python sur PyPI
**Labels**: `dx`, `sdk`, `priority:p1`, `wave:2`
**Description**:
- Pipeline génération + publication versionnée.
**Critères d'acceptation**:
- `pip install onefacture` fonctionnel.

### 12. Publication SDK TypeScript sur npm
**Labels**: `dx`, `sdk`, `priority:p1`, `wave:2`
**Description**:
- Pipeline génération + publication npm.
**Critères d'acceptation**:
- `npm install @onefacture/sdk` fonctionnel.

### 13. CLI onefacture doctor
**Labels**: `dx`, `tooling`, `priority:p2`, `wave:2`
**Description**:
- Vérifier clé API, reachability, schéma payload minimal.
**Critères d'acceptation**:
- Rapport diagnostic clair en sortie terminal.

### 14. Trace ID dans toutes les réponses API
**Labels**: `api`, `observability`, `priority:p1`, `wave:2`
**Description**:
- Injecter `X-Request-ID` dans logs + réponses.
**Critères d'acceptation**:
- Corrélation requête/logs de bout en bout.

### 15. Endpoint timeline facture
**Labels**: `api`, `observability`, `priority:p1`, `wave:2`
**Description**:
- Exposer transitions, erreurs, retries, latences.
**Critères d'acceptation**:
- Timeline complète pour toute facture non terminale.

### 16. Webhook inspector UI
**Labels**: `dx`, `webhooks`, `priority:p2`, `wave:2`
**Description**:
- Vue des tentatives, codes HTTP, payloads, replay.
**Critères d'acceptation**:
- Replay one-click d’un delivery échoué.

### 17. Erreurs enrichies RFC 7807 (hint/docs/retryable)
**Labels**: `api`, `ux`, `priority:p1`, `wave:2`
**Description**:
- Champs `remediation_hint`, `docs_url`, `retryable`.
**Critères d'acceptation**:
- 90% des erreurs top disposent d’un hint exploitable.

### 18. Pack d’exemples métier (avoir, correction, rejet)
**Labels**: `docs`, `dx`, `priority:p2`, `wave:2`
**Description**:
- Exemples JSON + snippets SDKs.
**Critères d'acceptation**:
- Scénarios couverts dans docs interactives.

## Vague 3 — Différenciation / Moat

### 19. Pré-validation bulk avant émission
**Labels**: `validation`, `enterprise`, `priority:p2`, `wave:3`
**Description**:
- Endpoint bulk pour analyser des lots de factures.
**Critères d'acceptation**:
- Rapport agrégé + export CSV erreurs.

### 20. Score qualité de conformité par tenant
**Labels**: `analytics`, `enterprise`, `priority:p2`, `wave:3`
**Description**:
- Calcul score hebdo (rejets, erreurs, correction speed).
**Critères d'acceptation**:
- Dashboard score + tendances mensuelles.

### 21. Assistant de correction automatique des rejets
**Labels**: `automation`, `ux`, `priority:p2`, `wave:3`
**Description**:
- Proposer patch JSON prêt à resoumettre.
**Critères d'acceptation**:
- Taux de re-soumission réussie amélioré.

### 22. Chiffrement at-rest BYOK/KMS
**Labels**: `security`, `compliance`, `priority:p1`, `wave:3`
**Description**:
- Intégration KMS + rotation clés + runbooks.
**Critères d'acceptation**:
- Données sensibles chiffrées et auditables.

### 23. mTLS optionnel + IP allowlist par webhook endpoint
**Labels**: `security`, `webhooks`, `priority:p2`, `wave:3`
**Description**:
- Sécurisation avancée des webhooks sortants.
**Critères d'acceptation**:
- Handshake mTLS validé et logs d’accès enrichis.

### 24. Framework multi-juridiction (PEPPOL/ViDA ready)
**Labels**: `architecture`, `future`, `priority:p3`, `wave:3`
**Description**:
- Abstraire règles pays/profils vers modules.
**Critères d'acceptation**:
- Ajout d’un nouveau profil pays sans toucher au core API.

## Vague 4 — Production Readiness & Conformité Factur-X Complète

### 25. Génération PDF/A-3 wire-complete + délégation sidecar (PDF/A-3 + Factur-X embedding)
**Labels**: `core`, `facturx`, `priority:p0`, `wave:4`
**Description**:
- Émettre un conteneur PDF minimal valide (%PDF-1.7 + métadonnées) afin que le pipeline d’émission soit end-to-end testable.
- Déléguer la génération réelle PDF/A-3 + attachment du XML CII + layout visuel au sidecar Python quand `ONEFACTURE_PDF_SIDECAR_URL` est défini.
**Critères d'acceptation**:
- `go test ./internal/core/facturx -run PackagePDFA3` passe.
- Le PDF généré commence par `%PDF-1.7` et contient les métadonnées attendues (invoice_number, profile, cii_xml_size).
- Le endpoint d’émission peut renvoyer ce conteneur sans erreur.
- Quand le sidecar est configuré, le vrai PDF/A-3 + XML embarqué est produit.

### 26. Charts Helm de production + observabilité minimale (Prometheus + Grafana + OTel)
**Labels**: `infra`, `observability`, `priority:p1`, `wave:4`
**Description**:
- Ajouter `values-prod.yaml` (HPA, PDB, resources, tracing) et `templates/prometheusrule.yaml` avec alertes réelles (DLQ, taux d'échec PA, latence).
- Ajouter un dashboard Grafana de base et configurer ServiceMonitor.
**Critères d'acceptation**:
- `helm template -f deploy/helm/onefacture/values-prod.yaml` produit des manifests valides incluant ServiceMonitor et PrometheusRule.
- Les alertes et le dashboard sont présents dans le chart.

### 27. Publication SDK automatisée sur GitHub Releases
**Labels**: `dx`, `ci`, `sdk`, `priority:p1`, `wave:4`
**Description**:
- Modifier le workflow `sdk-publish.yml` pour se déclencher automatiquement sur `release: published` (tags `v*`).
- Publier les deux SDKs et attacher les artefacts buildés (wheel + tarball) à la GitHub Release.
**Critères d'acceptation**:
- Un tag `v*` déclenche le workflow, publie les SDKs et attache les artefacts à la Release.

### 28. Adaptateurs supplémentaires (Cegid, Qonto) + extension multi-juridiction
**Labels**: `adapter`, `architecture`, `priority:p2`, `wave:4`
**Description**:
- Ajouter les packages complets `internal/adapters/cegid` et `internal/adapters/qonto` (même pattern que Pennylane).
- Enregistrer les adapters dans le Registry par défaut.
- Ajouter le profil `EU-ViDA` dans le jurisdiction registry et mettre à jour les tests.
**Critères d'acceptation**:
- `Registry.Names()` inclut "cegid" et "qonto".
- `go test ./internal/jurisdiction` passe avec le nouveau profil ViDA.
