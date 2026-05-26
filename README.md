<h1 align="center">
  <img src="https://raw.githubusercontent.com/yawo/onefacture/main/docs/assets/logo.png" alt="Logo onefacture" width="200" onerror="this.src='https://via.placeholder.com/200x50?text=onefacture';">
  <br>
  onefacture
</h1>

<h4 align="center">La passerelle API Open Source pour la Facturation Électronique Française (Réforme 2026)</h4>

<p align="center">
  <a href="#vision--le-problème">Problème</a> •
  <a href="#la-solution--onefacture">Solution</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#feuille-de-route-roadmap">Roadmap</a> •
  <a href="#démarrage-rapide">Démarrage rapide</a>
</p>

---

## 🇫🇷  Problème

À partir du **1er septembre 2026**, l'État français rend obligatoire l'émission, la transmission et la réception de factures au format électronique pour toutes les transactions B2B nationales assujetties à la TVA. Il ne s'agit pas d'un simple échange de PDF : cela implique des formats de données stricts (Factur-X, UBL, CII) et un routage via un réseau complexe de **Plateformes de Dématérialisation Partenaires (PDP)** et du **Portail Public de Facturation (PPF)** (le fameux "schéma en Y").

Pour les éditeurs d'ERP, les plateformes SaaS et les systèmes d'information internes, cela représente un cauchemar technique :
- **Fragmentation :** Il existe plus de 70 PDP immatriculées (Sage, Pennylane, Docaposte, Cegid, etc.), chacune imposant sa propre API propriétaire.
- **Complexité :** Générer des fichiers PDF/A-3 conformes avec XML embarqué (Factur-X) et les valider face à des centaines de règles métier (Schematron / AFNOR) est un défi majeur.
- **Enfermement propriétaire (Vendor Lock-in) :** Se connecter directement à une seule PDP lie la logique de facturation de votre système à leur infrastructure spécifique.

##  Solution : onefacture

**onefacture** est une API Gateway unifiée et open source qui abstrait l'intégralité de la complexité de l'écosystème de facturation électronique français.

Au lieu de développer des dizaines d'intégrations point-à-point, votre application communique avec **une seule API REST élégante**. Nous gérons le travail difficile : génération de Factur-X, validation stricte EN 16931, routage dynamique vers les PDP, et suivi du cycle de vie.

### Fonctionnalités Clés

-  **API Unifiée :** Une seule interface OpenAPI 3.1 orientée développeur pour tous vos besoins de facturation.
-  **Routage Intelligent :** Envoyez une facture ; `onefacture` interroge automatiquement l'Annuaire national et la route vers la PDP choisie par le destinataire.
-  **Validation Blindée :** Un pipeline de validation intégré à 6 couches (XSD + Schematron) garantit que vos factures ne seront jamais rejetées par l'administration fiscale.
-  **Natif Factur-X :** Génération à la volée de fichiers PDF/A-3 conformes avec XML embarqué (profils MINIMUM, BASIC, EN16931, EXTENDED).
-  **Webhooks Standardisés :** Recevez des événements de cycle de vie normalisés (ex: `invoice.submitted`, `invoice.paid`) quelles que soient les spécificités de la PDP sous-jacente.

---

##  Architecture

`onefacture` est conçu pour offrir un haut débit, une faible latence et une fiabilité à toute épreuve, en respectant les standards de connectivité **AFNOR XP Z12-013**.

**Stack Technique :**
*   **Gateway (Go 1.23+) :** La couche API hautement concurrente, le routage et la gestion des états (basée sur Fiber/Chi).
*   **Moteur de Validation (Sidecar Python) :** Gère la manipulation complexe du XML et la validation Schematron via `lxml`, assurant le respect strict de la norme AFNOR XP Z12-012.
*   **Base de données :** PostgreSQL avec `pgvector` pour les pistes d'audit immuables et l'isolation des données multi-tenants.
*   **Messagerie (Async) :** NATS ou Redis Streams pour la livraison asynchrone des webhooks et le polling des statuts PDP.

```mermaid
graph LR
    A[Votre ERP / SaaS] -->|REST / JSON| B(onefacture Gateway)
    B <--> C{Validation Sidecar Python}
    B -->|Adapter: Chorus Pro| D[PPF]
    B -->|Adapter: Pennylane| E[PDP 1]
    B -->|Adapter: Docaposte| F[PDP 2]
```

---

##  Feuille de route (Roadmap)

Nous sommes en plein développement actif pour respecter les échéances réglementaires de 2026.

- [x] **Phase 0 :** Recherche & Spécifications (Extraction des normes AFNOR, mapping XSD/Schematron).
- [x] **Phase 1 :** Fondations Core (Modèles Go, PostgreSQL, Sidecar de validation Python).
- [x] **Phase 2 :** API Gateway (CRUD Factures, définitions OpenAPI 3.1, Scalar docs).
- [x] **Phase 3 :** Adaptateurs PA — interface, registre et mock fonctionnel ; Chorus/Pennylane/Docaposte à brancher sur leurs sandboxes.
- [x] **Phase 4 :** Workers Asynchrones (Redis Streams, webhooks signés HMAC, polling lifecycle).
- [x] **Phase 5 :** Expérience Développeur (Vagues 1-4 complétées : sandbox, SDKs, PDF/A-3, Helm/obs, publication auto).

*(Consultez [ISSUES.md](./ISSUES.md) pour le backlog détaillé, [les exemples metier](./docs/examples/business-scenarios.md) pour les cas avoir/correction/rejet, et [les gates d'acceptance externes](./docs/operations/external-acceptance.md) pour les validations qui exigent des services reels).*

---

##  Démarrage rapide

### Prérequis
- **Go** 1.23+ (`go version`)
- **Docker** + Docker Compose (`docker --version && docker-compose --version`)
- **Python** 3.10+ for sidecar (`python --version`)
- **Make** for commands (`make --version`)

### 1. Lancer via Docker Compose (développement)
```bash
git clone https://github.com/yawo/onefacture.git
cd onefacture
docker-compose -f deploy/docker-compose.yml up -d
make migrate-up       # Apply database migrations
```

API: http://localhost:8080 · Docs Scalar: http://localhost:8080/docs · OpenAPI: http://localhost:8080/openapi.json

**Architecture Docker:**
- `postgres:5432` - Base de données PostgreSQL
- `redis:6379` - Messagerie Redis Streams
- `sidecar:8081` - Moteur validation Python
- `api:8080` - Gateway Go

### 2. Configuration environnement
Copiez `.env.example` vers `.env` et configurez:

```bash
# API Key for gateway auth
ONEFACTURE_API_KEY=your-secret-key

# PostgreSQL connection
DATABASE_URL=postgres://onefacture:onefacture@localhost:5432/onefacture?sslmode=disable

# Redis pour async workers
REDIS_URL=redis://localhost:6379

# Chorus Pro (sandbox par défaut)
ONEFACTURE_CHORUS_BASE_URL=https://sandbox-api.piste.gouv.fr/cpro
ONEFACTURE_CHORUS_CLIENT_ID=your-client-id
ONEFACTURE_CHORUS_CLIENT_SECRET=your-client-secret
ONEFACTURE_CHORUS_ACCESS_TOKEN=or-use-static-token
```

### 3. Migrations de base de données
```bash
# Developpement
make migrate-up

# Production (via psql ou migrate CLI)
psql $DATABASE_URL -f internal/storage/migrations/2024*.sql
```

### 4. Tests complets
```bash
make test              # Tests unitaires avec couverture
go test -race -count=1 ./...  # Tous les tests

# Tests integration (nécessite Docker pour dépendances)
make test-integration
```

### 5. Validation locale
```bash
make verify-local      # Vérifie manifest, smoke, scripts
make lint              # golangci-lint (si installé)
```

### 6. Connexion à une PA (adapter)
Chaque adapter peut être configuré via variables d'environnement:

```bash
# Chorus Pro (sandbox ou production)
ONEFACTURE_CHORUS_BASE_URL=https://sandbox-api.piste.gouv.fr/cpro  # sandbox
# ou
ONEFACTURE_CHORUS_BASE_URL=https://api.piste.gouv.fr/cpro         # production

# Pennylane
ONEFACTURE_PENNYLANE_BASE_URL=https://api.pennylane.fr/invoices
ONEFACTURE_PENNYLANE_API_TOKEN=your-api-token

# Docaposte
ONEFACTURE_DOCAPOSTE_BASE_URL=https://api.docaposte.fr
ONEFACTURE_DOCAPOSTE_API_TOKEN=your-api-token

# Cegid
ONEFACTURE_CEGID_BASE_URL=https://api.cegid.fr
ONEFACTURE_CEGID_API_TOKEN=your-api-token

# Qonto
ONEFACTURE_QONTO_BASE_URL=https://api.qonto.com
ONEFACTURE_QONTO_API_TOKEN=your-api-token
```

### 7. Sidecar Python (validation PDF/A-3 + XML)
Le sidecar tourne dans Docker (`make dev`) ou separément:

```bash
cd sidecar/pdf
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python main.py  # listens on :8081
```

### 8. Workers asynchrones
```bash
# Démarrer les workers (polling + webhooks)
make dev  # inclut Redis + workers

# Ou separément
go run ./cmd/worker
```

### 9. Production (Docker)
```bash
cp .env.example .env.prod
# Configure for production
docker-compose -f docker-compose.prod.yml up -d
```

---

##  Contribuer

Les contributions sont les bienvenues ! Que ce soit pour construire un adaptateur pour une PDP spécifique, améliorer le moteur de validation, ou enrichir la documentation, votre aide est essentielle pour démocratiser la facturation électronique en France.

Veuillez lire notre [Guide de contribution](./CONTRIBUTING.md) (en cours de rédaction) pour commencer.

### Verification rapide

```bash
make verify-local   # tests, smokes, manifest, YAML et actionlint locaux du backlog
make verify-sdk     # artefacts SDK installables localement
```

Les gates qui dependent de sandboxes PA, de registres publics ou d'un broker KMS deploye sont documentes dans [docs/operations/external-acceptance.md](./docs/operations/external-acceptance.md). Avant de les lancer, verifier la configuration locale et GitHub Actions:

```bash
make check-external-env
make check-github-external-config GITHUB_REPO=yawo/onefacture
```

##  Licence

Ce projet est sous licence **Apache 2.0** - voir le fichier `LICENSE` pour plus de détails.
