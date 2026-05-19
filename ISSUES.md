# ISSUES.md — Backlog GitHub pour onefacture

Ce fichier est le backlog officiel. Chaque issue est conçue pour être créée sur GitHub avec ses labels et critères d'acceptation.
**Stack cible :** Go 1.23+ (Fiber/Chi), PostgreSQL, Python Sidecar (lxml), NATS/Redis.

---

## EPIC 0 — Fondations & Qualité

### ISSUE-001 · Initialisation du Monorepo Go
**Labels** : `infra`, `good first issue`
**Description** :
- Initialiser `go.mod` et la structure : `cmd/api`, `internal/core`, `internal/gateway`, `internal/adapters`, `pkg/`.
- Configurer `Makefile` (build, test, lint, dev).
- Créer `.editorconfig`, `.gitignore` et `LICENSE` (Apache 2.0).
- Configurer `golangci-lint` avec des règles strictes.
**Critères d'acceptation** : `make test` et `make lint` passent. `go run cmd/api/main.go` démarre un serveur minimal.

### ISSUE-002 · CI/CD GitHub Actions
**Labels** : `infra`
**Description** :
- Créer `.github/workflows/ci.yml`.
- Pipeline : Lint Go, Tests unitaires Go (avec coverage), Build image Docker.
- Pipeline : Lint Python (pour le sidecar).
**Critères d'acceptation** : Tout commit/PR déclenche la CI. Échec si < 80% coverage ou erreur lint.

---

## EPIC 1 — Core Domain & Validation

### ISSUE-003 · Modèle Invoice unifié (Factur-X / EN 16931)
**Labels** : `core`
**Description** :
- Définir les structs Go pour `Invoice`, `Party`, `Line`, `Totals`, `Status`.
- Support des profils : MINIMUM, BASIC, EN16931, EXTENDED.
- Tags JSON complets et validation `validator/v10`.
**Critères d'acceptation** : Sérialisation JSON robuste. Tests unitaires couvrant tous les profils.

### ISSUE-004 · Pipeline de Validation (avec Sidecar Python)
**Labels** : `core`, `validation`
**Description** :
- Implémenter le service de validation en Go.
- Créer un sidecar Python (FastAPI/lxml) pour la validation Schematron (AFNOR XP Z12-012) et XSD.
- Étapes : PDF/A-3 extraction -> XML XSD -> Schematron -> Business Rules.
- Mapper les erreurs au format RFC 7807.
**Critères d'acceptation** : Validation complète d'une facture Factur-X avec retour d'erreurs précis (XPath).

### ISSUE-005 · Génération Factur-X
**Labels** : `core`
**Description** :
- Génération du XML CII (CrossIndustryInvoice) depuis le modèle Go.
- Génération du PDF/A-3 et injection du XML (conforme Factur-X).
- Support des différents profils de conformité.
**Critères d'acceptation** : Les fichiers générés passent la validation ISSUE-004 et l'outil FNFE-MPE.

---

## EPIC 2 — Persistence & Multi-tenancy

### ISSUE-006 · PostgreSQL & Migrations
**Labels** : `infra`, `core`
**Description** :
- Configurer PostgreSQL avec `golang-migrate` ou `sqlc`.
- Tables : `organizations`, `api_keys`, `invoices`, `lifecycle_events`.
- Multi-tenancy : colonne `organization_id` obligatoire sur toutes les tables.
**Critères d'acceptation** : Migrations idempotentes. Isolation des données vérifiée par tests.

### ISSUE-007 · Audit Trail & Historique de Cycle de Vie
**Labels** : `core`, `security`
**Description** :
- Table `audit_log` immuable pour chaque action.
- Machine à états pour l'Invoice : `DRAFT` -> `SUBMITTED` -> `ACCEPTED`/`REJECTED`.
- Stockage des transitions avec horodatage et preuve d'intégrité.
**Critères d'acceptation** : Historique complet consultable pour chaque facture. Transitions invalides rejetées (409).

---

## EPIC 3 — API Gateway

### ISSUE-008 · Auth API Key & Middleware
**Labels** : `api`, `security`
**Description** :
- Authentification par header `X-API-Key` (stockage haché en DB).
- Middleware d'injection du `organization_id` dans le contexte.
- Rate limiting par organisation (Redis).
**Critères d'acceptation** : 401 si clé invalide. 429 si dépassement quota.

### ISSUE-009 · Endpoints CRUD Invoices
**Labels** : `api`
**Description** :
- `POST /v1/invoices` : Créer et valider.
- `GET /v1/invoices/{id}` : Détail et statut.
- `GET /v1/invoices` : Liste filtrée (status, date, buyer).
- `GET /v1/inbox` : Factures reçues.
**Critères d'acceptation** : Conforme aux specs OpenAPI 3.1 définies dans `AGENTS.md`.

### ISSUE-010 · Documentation Scalar & OpenAPI
**Labels** : `api`, `dx`
**Description** :
- Générer automatiquement `openapi.json` depuis le code Go.
- Intégrer Scalar UI sur `/docs`.
- Ajouter des exemples de payloads pour chaque profil Factur-X.
**Critères d'acceptation** : Documentation interactive complète et à jour.

---

## EPIC 4 — Adapters Plateformes Agréées (PA)

### ISSUE-011 · Interface PAAdapter & Registry
**Labels** : `adapter`, `core`
**Description** :
- Définir l'interface `PAAdapter` en Go (Submit, GetStatus, Webhook).
- Implémenter un `Registry` pour instancier dynamiquement l'adaptateur selon l'organisation.
- Créer un `MockPAAdapter` pour le développement local.
**Critères d'acceptation** : Injection de dépendance fonctionnelle. Tests d'intégration avec le Mock.

### ISSUE-012 · Adapter Chorus Pro / PPF
**Labels** : `adapter`
**Description** :
- Implémenter l'intégration avec l'API PISTE (OAuth2).
- Soumission de flux et récupération des statuts.
- Support du E-reporting.
**Critères d'acceptation** : Succès des tests sur la sandbox Chorus Pro.

### ISSUE-013 · Adapters Partenaires (Pennylane, Docaposte, Qonto)
**Labels** : `adapter`
**Description** :
- Implémenter les adaptateurs pour les PA prioritaires.
- Normalisation des erreurs et des statuts vers le modèle `onefacture`.
**Critères d'acceptation** : Round-trip complet (émission/réception) pour chaque PA supportée.

---

## EPIC 5 — Async, Events & Webhooks

### ISSUE-014 · Bus d'événements (NATS/Redis) & Workers
**Labels** : `infra`, `core`
**Description** :
- Mettre en place NATS ou Redis Streams pour le traitement asynchrone.
- Workers pour le polling des statuts PA et l'envoi des webhooks.
**Critères d'acceptation** : Traitement robuste aux pannes (retry avec backoff).

### ISSUE-015 · Webhooks sortants
**Labels** : `api`, `dx`
**Description** :
- Permettre aux clients de configurer une URL de webhook.
- Signature HMAC-SHA256 pour la sécurité.
- Événements : `invoice.submitted`, `invoice.accepted`, `invoice.received`.
**Critères d'acceptation** : Livraison fiable des notifications de changement d'état.

---

## EPIC 6 — DX & SDKs

### ISSUE-016 · SDK Python & TypeScript
**Labels** : `dx`
**Description** :
- Générer et publier les SDKs depuis la spec OpenAPI.
- Ajouter des wrappers haut-niveau pour simplifier l'intégration.
**Critères d'acceptation** : Installation via `pip` et `npm` fonctionnelle.

### ISSUE-017 · Sandbox & Playground
**Labels** : `dx`, `docs`
**Description** :
- Déployer une instance de test publique avec des PA mockées.
- Fournir des credentials de test immédiats.
**Critères d'acceptation** : Un développeur peut tester l'API sans configuration PA réelle.

---

## EPIC 7 — Sécurité, Compliance & Infra

### ISSUE-018 · Conformité RGPD & Chiffrement
**Labels** : `security`
**Description** :
- Endpoints d'export et de suppression de données.
- Chiffrement "at-rest" des données sensibles (AES-256).
**Critères d'acceptation** : DPO-friendly. Audit de sécurité passé.

### ISSUE-019 · Helm Charts & Déploiement Cloud
**Labels** : `infra`
**Description** :
- Créer les charts Helm pour Kubernetes.
- Configurer le monitoring (Prometheus/Grafana) et le tracing (OpenTelemetry).
**Critères d'acceptation** : Déploiement reproductible en un clic. Tableaux de bord opérationnels.
