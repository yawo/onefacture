# ISSUES.md

```markdown
# ISSUES.md — GitHub Issues pour onefacture

Chaque issue est formatée pour être créée directement via l'API GitHub.
Labels utilisés : `core`, `adapter`, `api`, `validation`, `security`,
`docs`, `test`, `dx`, `infra`, `ereporting`, `good first issue`

---

## EPIC 1 — Fondations & Modèles

### ISSUE-001 · Initialisation du projet
**Labels** : `infra`, `good first issue`
**Description** :
- Créer la structure de dossiers (`api/`, `adapters/`, `core/`, etc.)
- Configurer `pyproject.toml` (FastAPI, Pydantic v2, httpx, lxml, pypdf)
- Configurer pre-commit (ruff, mypy, black)
- Créer `.github/workflows/ci.yml` (lint + test + build Docker)
- Créer `docker-compose.yml` (api, postgres, redis)
**Critères d'acceptation** : `make dev` démarre l'API en local. CI passe.

---

### ISSUE-002 · Modèles Pydantic v2 core
**Labels** : `core`
**Description** :
Implémenter les modèles Pydantic v2 pour :
- `Invoice`, `InvoiceLine`, `Party` (seller/buyer), `Address`
- `LifecycleEvent`, `InvoiceStatus` (enum)
- `PASubmitResult`, `ValidationResult`, `ErrorDetail`
- `EReportingTransaction`, `EReportingPayment`
- `WebhookEndpoint`, `WebhookEvent`
**Critères d'acceptation** : sérialisation/désérialisation JSON sans perte. Tests unitaires 100%.

---

### ISSUE-003 · Validation Factur-X (pipeline complet)
**Labels** : `core`, `validation`
**Description** :
Implémenter `core/validation/facturx.py` :
1. Validation PDF/A-3 (pypdf)
2. Extraction du XML embarqué
3. Validation XSD EN 16931 (télécharger schémas FNFE-MPE)
4. Validation Schematron AFNOR XP Z12-012 (lxml)
5. Validation métier (SIREN, TVA, totaux, dates)
6. Retourner `ValidationResult` avec liste d'erreurs structurées
**Critères d'acceptation** : 20 factures de test (valides + invalides) toutes correctement classées.

---

### ISSUE-004 · Génération Factur-X (tous profils)
**Labels** : `core`
**Description** :
Implémenter `core/generation/facturx.py` :
- Profils : MINIMUM, BASIC_WL, BASIC, EN16931, EXTENDED
- Génération XML CII (CrossIndustryInvoice) depuis modèle `Invoice`
- Génération PDF/A-3 (layout sobre, logo paramétrable)
- Embarquement XML dans PDF (PDF/A-3 conforme)
- Support UBL 2.1 et CII en sortie alternative
**Critères d'acceptation** : fichiers générés passent la validation ISSUE-003 et l'outil FNFE-MPE.

---

### ISSUE-005 · Base de données & migrations
**Labels** : `infra`, `core`
**Description** :
- Modèle SQLAlchemy 2.0 (async) pour `invoices`, `lifecycle_events`,
  `organizations`, `pa_credentials`, `webhooks`, `audit_log`
- Migrations Alembic initiales
- Multi-tenancy : colonne `organization_id` sur toutes les tables
- Index : `(organization_id, status)`, `(organization_id, issue_date)`
**Critères d'acceptation** : migrations UP/DOWN idempotentes. Requêtes < 10ms sur 100k factures.

---

## EPIC 2 — API Gateway

### ISSUE-006 · Auth OAuth2 + API Key
**Labels** : `api`, `security`
**Description** :
- OAuth2 client_credentials flow (FastAPI SecurityScopes)
- API Key via header `X-API-Key`
- Middleware d'injection `organization_id` depuis token
- Rate limiting par API key (Redis token bucket)
- Endpoint `POST /v1/auth/token`
**Critères d'acceptation** : endpoints non authentifiés retournent 401. Rate limit retourne 429.

---

### ISSUE-007 · Endpoints CRUD Invoices
**Labels** : `api`
**Description** :
Implémenter tous les endpoints `/v1/invoices` :
- `POST /v1/invoices` : créer (DRAFT) + valider + optionnellement soumettre
- `GET /v1/invoices` : liste paginée (filtres : status, date, buyer_siren, pa_id)
- `GET /v1/invoices/{id}` : détail complet
- `PUT /v1/invoices/{id}` : modifier (DRAFT uniquement)
- `DELETE /v1/invoices/{id}` : annuler
- `GET /v1/invoices/{id}/download?format=pdf|xml` : téléchargement
**Critères d'acceptation** : OpenAPI spec générée et validée. Tests d'intégration complets.

---

### ISSUE-008 · Endpoints soumission & cycle de vie
**Labels** : `api`, `core`
**Description** :
- `POST /v1/invoices/{id}/submit` : soumettre à la PA configurée
- `POST /v1/invoices/{id}/cancel` : générer avoir d'annulation
- `GET /v1/invoices/{id}/lifecycle` : liste des événements horodatés
- Machine à états : DRAFT → SUBMITTED → ACCEPTED/REJECTED → PAID/CANCELLED
- Transitions invalides retournent 409 Conflict
**Critères d'acceptation** : toutes transitions testées. Impossible de régresser un état.

---

### ISSUE-009 · Endpoints Inbox (réception)
**Labels** : `api`
**Description** :
- `GET /v1/inbox` : factures reçues (filtres, pagination cursor-based)
- `POST /v1/inbox/{id}/acknowledge` : accusé de réception
- `POST /v1/inbox/{id}/approve` : approuver
- `POST /v1/inbox/{id}/reject` : rejeter (`body: { reason, code }`)
- Worker de polling PA → inbox (configurable par PA)
**Critères d'acceptation** : factures reçues depuis PA mockée apparaissent dans inbox < 30s.

---

### ISSUE-010 · Endpoint Validation & Conversion
**Labels** : `api`, `validation`, `dx`
**Description** :
- `POST /v1/validate` : upload multipart PDF/XML, retourne `ValidationResult`
- `POST /v1/convert` : convertir Factur-X ↔ UBL ↔ CII
- Réponse enrichie : score conformité, liste erreurs avec path XPath, suggestions
**Critères d'acceptation** : outil utilisable standalone par développeurs sans compte.

---

### ISSUE-011 · Annuaire PA (lookup SIREN)
**Labels** : `api`, `core`
**Description** :
- `GET /v1/directory/lookup?siren=XXX` : retourner PA(s) enregistrée(s) pour un SIREN
- Synchronisation avec l'annuaire PPF (DGFiP) via API ou scraping officiel
- Cache Redis (TTL 1h) + fallback DB locale
- Données : SIREN → PA_ID, adresse facturation électronique, date inscription
**Critères d'acceptation** : lookup < 100ms (cache hit). Données fraîches < 2h.

---

### ISSUE-012 · Webhooks
**Labels** : `api`, `dx`
**Description** :
- CRUD webhooks (`POST/GET/DELETE /v1/webhooks`)
- Événements : `invoice.submitted`, `invoice.accepted`, `invoice.rejected`,
  `invoice.paid`, `invoice.received`, `lifecycle.updated`
- Livraison avec signature HMAC-SHA256 (`X-Onefacture-Signature`)
- Retry avec backoff exponentiel (3 tentatives)
- Log des livraisons consultable via API
**Critères d'acceptation** : livraison < 5s post-événement. Signature vérifiable.

---

### ISSUE-013 · E-reporting
**Labels** : `api`, `ereporting`
**Description** :
- `POST /v1/ereporting/transactions` : données B2C / export hors UE
- `POST /v1/ereporting/payments` : données paiements
- Validation conformité DGFiP
- Transmission à la PA connectée
- Statuts de transmission retournés
**Critères d'acceptation** : format conforme spec DGFiP. Tests avec PA mockée.

---

## EPIC 3 — Adapters Plateformes Agréées

### ISSUE-014 · Interface PAAdapter + Mock
**Labels** : `adapter`, `test`
**Description** :
- Définir ABC `PAAdapter` (cf. AGENTS.md §5)
- Implémenter `MockPAAdapter` pour les tests
- `PAAdapterFactory` : instancier adapter selon `pa_id`
- Configuration par PA : base URL, credentials, timeout, retry
**Critères d'acceptation** : tous les tests EPIC 2 passent avec MockPAAdapter.

---

### ISSUE-015 · Adapter Chorus Pro / PPF
**Labels** : `adapter`
**Description** :
- Implémenter `adapters/chorus_pro/adapter.py`
- Auth : OAuth2 (PISTE API DGFiP)
- Submit facture (dépôt flux)
- Récupération statuts
- Réception factures (flux entrants)
- E-reporting
**Critères d'acceptation** : tests d'intégration sur sandbox Chorus Pro.

---

### ISSUE-016 · Adapter Super PDP
**Labels** : `adapter`, `good first issue`
**Description** :
- Implémenter `adapters/super_pdp/adapter.py`
- Documenter l'API Super PDP (REST, simple) [web:6]
- Submit, statuts, réception
**Critères d'acceptation** : round-trip complet sur sandbox Super PDP.

---

### ISSUE-017 · Adapter B2Brouter
**Labels** : `adapter`
**Description** :
- Implémenter `adapters/b2brouter/adapter.py`
- API REST B2Brouter (PA agréée, multi-format) [web:15]
- Submit (Factur-X, UBL, CII), statuts, réception
**Critères d'acceptation** : round-trip complet sandbox B2Brouter.

---

### ISSUE-018 · Adapter Sage PA
**Labels** : `adapter`
**Description** :
- Implémenter `adapters/sage/adapter.py`
- Documenter l'API Sage PA (OAuth2, REST)
- Submit, statuts, réception, e-reporting
**Critères d'acceptation** : tests d'intégration sandbox Sage.

---

### ISSUE-019 · Adapter Axway
**Labels** : `adapter`
**Description** :
- Implémenter `adapters/axway/adapter.py`
- API Axway (B2Bi / MFT Gateway)
- Submit, statuts, réception
**Critères d'acceptation** : tests d'intégration sandbox Axway.

---

### ISSUE-020 · Adapter Pagero
**Labels** : `adapter`
**Description** :
- Implémenter `adapters/pagero/adapter.py`
- API Pagero REST
- Submit, statuts, réception
**Critères d'acceptation** : tests d'intégration sandbox Pagero.

---

### ISSUE-021 · Adapter générique XP Z12-013
**Labels** : `adapter`, `core`
**Description** :
- Implémenter `adapters/generic_z12013/adapter.py`
- Basé strictement sur la norme AFNOR XP Z12-013 (API standard)
- Auto-découverte endpoint via annuaire
- Utilisé pour toute PA qui publie une API Z12-013 conforme
**Critères d'acceptation** : compatible avec au moins 2 PA ayant publié leur conformité Z12-013.

---

## EPIC 4 — Developer Experience

### ISSUE-022 · SDK Python
**Labels** : `dx`
**Description** :
- Générer SDK Python depuis OpenAPI spec (openapi-python-client)
- Ajouter helpers haut niveau : `onefacture.send()`, `onefacture.receive()`
- Publier sur PyPI (`pip install onefacture`)
- README avec quickstart 5 lignes
**Critères d'acceptation** : quickstart fonctionnel en < 10 min depuis zéro.

---

### ISSUE-023 · SDK TypeScript
**Labels** : `dx`
**Description** :
- Générer SDK TypeScript depuis OpenAPI spec (openapi-typescript-codegen)
- Types stricts, support ESM + CJS
- Publier sur npm (`npm install onefacture`)
**Critères d'acceptation** : types corrects, tests Vitest passent.

---

### ISSUE-024 · Sandbox publique & Playground
**Labels** : `dx`, `docs`
**Description** :
- Déployer instance sandbox publique (PA toutes mockées)
- Credentials de test disponibles sans inscription
- Interface Swagger UI / Scalar interactive
- Exemples de requêtes préchargés (Factur-X MINIMUM, EN16931, e-reporting)
**Critères d'acceptation** : développeur peut tester l'API entière sans compte réel PA.

---

### ISSUE-025 · Documentation complète
**Labels** : `docs`
**Description** :
- Guide "Démarrage rapide" (< 5 min, 1 facture émise)
- Guide intégration ERP (Odoo, Sage, custom)
- Référence API complète (OpenAPI Redoc)
- Guide adapters : comment contribuer un nouvel adapter PA
- FAQ : cas d'usage AFNOR XP Z12-014 couverts
- Changelog automatique (conventional commits)
**Critères d'acceptation** : revue par 2 développeurs externes sans aide.

---

## EPIC 5 — Sécurité & Compliance

### ISSUE-026 · Audit trail immuable
**Labels** : `security`, `core`
**Description** :
- Table `audit_log` : chaque action (émission, statut, accès) horodatée et signée
- Impossible de modifier un log (trigger DB + hash chaîné)
- Endpoint `GET /v1/audit?invoice_id=...`
- Rétention configurable (défaut 10 ans — obligation fiscale)
**Critères d'acceptation** : altération d'un log détectée < 100ms.

---

### ISSUE-027 · Conformité RGPD
**Labels** : `security`
**Description** :
- `GET /v1/gdpr/export` : export de toutes les données d'une organisation (JSON)
- `DELETE /v1/gdpr/purge` : suppression RGPD (sauf obligations fiscales)
- Chiffrement des données sensibles at-rest (AES-256)
- Politique de rétention automatique
**Critères d'acceptation** : DPO peut exercer droits RGPD via API.

---

## EPIC 6 — Infra & Opérations

### ISSUE-028 · Docker + Helm chart
**Labels** : `infra`
**Description** :
- `docker-compose.yml` complet (api, worker, postgres, redis)
- Helm chart Kubernetes (`helm install onefacture ./chart`)
- Health endpoints : `GET /health` et `GET /ready`
- Métriques Prometheus (`/metrics`)
**Critères d'acceptation** : déploiement Kubernetes en < 5 min.

---

### ISSUE-029 · Tests de charge
**Labels** : `infra`, `test`
**Description** :
- Scénario Locust : 1000 factures/min en émission, 500/min en réception
- Objectif : p99 < 500ms, 0 erreur à charge nominale
- Rapport automatique dans CI (PR label `load-test`)
**Critères d'acceptation** : SLO atteints sur infra standard (4 CPU, 8 GB RAM).

---

### ISSUE-030 · Monitoring & Alertes
**Labels** : `infra`
**Description** :
- Dashboard Grafana : taux d'erreur par PA, latence, volume factures
- Alertes : PA indisponible > 2 min, taux d'erreur > 1%, queue > 1000 messages
- Tracing distribué (OpenTelemetry)
**Critères d'acceptation** : alerte PA down reçue < 3 min après panne simulée.
```

