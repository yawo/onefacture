<h1 align="center">
  <img src="https://raw.githubusercontent.com/yawo/onefacture/main/docs/assets/logo.png" alt="Logo onefacture" width="200" onerror="this.src='https://via.placeholder.com/200x50?text=onefacture';">
  <br>
  onefacture
</h1>

<h4 align="center">La passerelle API Open Source unifiée pour la Facturation Électronique Française (Réforme 2026)</h4>

<p align="center">
  <a href="#-problème">Problème</a> •
  <a href="#-solution--onefacture">Solution</a> •
  <a href="#-architecture-cible">Architecture</a> •
  <a href="#-modèle-de-données-core-unified-invoice">Modèle Core</a> •
  <a href="#-endpoints-api-rest-openapi-31">Endpoints</a> •
  <a href="#-pipeline-de-validation--génération">Validation & Génération</a> •
  <a href="#-sécurité-kms--byok-workload-encryption">Sécurité & KMS</a> •
  <a href="#-démarrage-rapide--quickstart">Démarrage rapide</a> •
  <a href="#-cas-dusage-métier--payloads-réels">Cas Métier</a> •
  <a href="#-preuves-dacceptation-externes-external-acceptance">Preuves Externes</a> •
  <a href="#-roadmap--cycle-de-développement">Roadmap</a> •
  <a href="#-licence-license">Licence</a>
</p>

---

## 🇫🇷 Problème

À partir du **1er septembre 2026**, l'État français rend obligatoire l'émission, la transmission et la réception de factures au format électronique pour toutes les transactions B2B nationales assujetties à la TVA. Il ne s'agit pas d'un simple échange de PDF : cela implique des formats de données stricts (Factur-X, UBL, CII) et un routage via un réseau complexe de **Plateformes de Dématérialisation Partenaires (PDP)** et du **Portail Public de Facturation (PPF)** (le fameux "schéma en Y").

Pour les éditeurs d'ERP, les plateformes SaaS et les systèmes d'information internes, cela représente un véritable défi technique :
- **Fragmentation :** Plus de 100 Plateformes Agréées (PA) et PDP immatriculées (Sage, Pennylane, Docaposte, Cegid, Qonto, etc.), chacune imposant sa propre API propriétaire.
- **Complexité Normative :** Générer des fichiers PDF/A-3 conformes avec XML embarqué (Factur-X) et les valider face à des centaines de règles métier (Schematron / AFNOR) exige un outillage lourd.
- **Enfermement propriétaire (Vendor Lock-in) :** Se connecter directement à une seule PDP lie la logique de facturation de votre système à leur infrastructure spécifique.

---

##  Solution : onefacture

**onefacture** est une API Gateway unifiée et open source qui abstrait l'intégralité de la complexité de l'écosystème de facturation électronique français.

Au lieu de développer des dizaines d'intégrations point-à-point, votre application communique avec **une seule API REST élégante**. Nous gérant le travail difficile : génération de Factur-X, validation stricte EN 16931, routage dynamique vers les PDP, et suivi du cycle de vie.

### Standards de référence implémentés
*   **XP Z12-012** : formats et profils des messages factures.
*   **XP Z12-013** : API standard pour interfacer SI entreprise ↔ PA.
*   **XP Z12-014** : cas d'usage B2B.

---

##  Architecture 

`onefacture` est conçu pour offrir un haut débit, une faible latence et une fiabilité à toute épreuve, en respectant les standards de connectivité **AFNOR XP Z12-013**.

```
┌─────────────────────────────────────────────────┐
│              Client / ERP / SaaS                │
│         (Sage, Odoo, custom app...)             │
└───────────────────┬─────────────────────────────┘
                    │  REST API onefacture (OpenAPI 3.1)
                    ▼
┌─────────────────────────────────────────────────┐
│              onefacture API Gateway             │
│  - Langage : Go 1.23+ (Fiber / Chi)             │
│  - Auth : API Key / OAuth2                      │
│  - Routing logique PA                           │
│  - Validation Factur-X / UBL / CII              │
│  - Normalisation Request/Response               │
└──────────┬──────────────┬──────────────┬────────┘
           │              │              │
    ┌──────▼──────┐ ┌─────▼──────┐ ┌────▼───────┐
    │ Adaptateur  │ │ Adaptateur │ │ Adaptateur │
    │    Chorus   │ │  Pennylane │ │    Cegid   │
    └──────┬──────┘ └─────┬──────┘ └────┬───────┘
           │              │              │
    ┌──────▼──────────────▼──────────────▼────────┐
    │           Plateformes Agréées (PA)          │
    └─────────────────────────────────────────────┘
```

### Stack Technique
*   **API Gateway (Go 1.23+) :** Couche API hautement concurrente, routage et gestion des états.
*   **Moteur de Validation (Sidecar Python) :** Gère la manipulation complexe du XML et la validation Schematron via `lxml`, assurant le respect strict de la norme AFNOR XP Z12-012.
*   **Base de données :** PostgreSQL avec `pgvector` pour les pistes d'audit immuables et l'isolation des données multi-tenants.
*   **Enveloppe de Sécurité (KMS/BYOK) :** Chiffrement fort des fichiers XML et PDF avant stockage à l'aide d'enveloppes AES-256-GCM.
*   **Messagerie (Async) :** NATS ou Redis Streams pour la livraison asynchrone des webhooks et le polling des statuts PDP.

---

##  Modèle de Données Core (Unified Invoice)

L'Invoice est la ressource centrale, basée sur la norme **Factur-X 1.08 / EN 16931**.

```go
type Invoice struct {
    ID           string      `json:"id"`
    Status       Status      `json:"status"` // DRAFT, SUBMITTED, RECEIVED, REJECTED, PAID
    Profile      Profile     `json:"profile"` // MINIMUM, BASIC, EN16931, EXTENDED
    Type_Code    string      `json:"type_code"` // 380 (Facture), 381 (Avoir), 384 (Corrective)
    Number       string      `json:"number"`
    Currency     string      `json:"currency"`
    Seller       Party       `json:"seller"`
    Buyer        Party       `json:"buyer"`
    Lines        []Line      `json:"lines"`
    Totals       Totals      `json:"totals"`
    IssueDate    time.Time   `json:"issue_date"`
    DueDate      time.Time   `json:"due_date"`
    PAID         string      `json:"pa_id"`
    PARef        string      `json:"pa_ref"`
    RawXML       []byte      `json:"-"` // Chiffré via KMS
    RawPDF       []byte      `json:"-"` // Chiffré via KMS
}
```

---

##  Endpoints API REST (OpenAPI 3.1)

### Invoices & Réception
| Méthode | Route | Description |
|---|---|---|
| `POST` | `/v1/invoices` | Créer + émettre une facture (paramètre `?submit=true` optionnel) |
| `GET` | `/v1/invoices/{id}` | Détail et statut actuel |
| `GET` | `/v1/invoices/{id}/timeline` | Historique complet des événements du cycle de vie |
| `POST` | `/v1/invoices/{id}/submit` | Soumettre à la PA (si DRAFT) |
| `POST` | `/v1/invoices/{id}/retry` | Résoudre un rejet et soumettre à nouveau |
| `GET` | `/v1/inbox` | Lister les factures reçues depuis le réseau |
| `POST` | `/v1/inbox/{id}/approve` | Approuver une facture reçue |

### Validation, Webhooks & Sandbox
| Méthode | Route | Description |
|---|---|---|
| `POST` | `/v1/validate` | Valider un fichier Factur-X/UBL/CII brut |
| `POST` | `/v1/webhooks` | Enregistrer un endpoint de webhook de notification |
| `GET` | `/v1/webhooks/deliveries` | Suivi et journaux de livraison des webhooks |
| `POST` | `/v1/sandbox/credentials` | Provisionner des identifiants sandbox multi-tenant |
| `GET` | `/v1/directory/lookup` | Trouver la PA d'un destinataire par son SIREN |
| `GET` | `/v1/platforms` | Liste de santé et de diagnostic des PA supportées |

---

##  Interface des Adaptateurs PA (`PAAdapter`)

Chaque plateforme partenaire (PA) ou PDP est connectée via une interface Go unifiée :

```go
type PAAdapter interface {
    Name() string
    Submit(ctx context.Context, inv *Invoice) (*SubmitResult, error)
    GetStatus(ctx context.Context, paRef string) (*LifecycleEvent, error)
    Webhook(ctx context.Context, payload []byte) (*WebhookEvent, error)
    HealthCheck(ctx context.Context) error
}
```

### Ordre de priorité d'implémentation
1. **Chorus Pro / PPF** (Référence d'État - PISTE OAuth2 client credentials)
2. **Docaposte (SERES)** (~35% du marché)
3. **Pennylane** (Forte adoption PME)
4. **Cegid / Qonto** (ERP et Fintech)

---

##  Pipeline de Validation & Génération

`onefacture` garantit la conformité stricte grâce à un traitement structuré en plusieurs couches :

### Pipeline de Validation (6 couches)
1. **PDF/A-3 :** Validation du conteneur de document.
2. **Extraction :** Récupération sécurisée du XML CII/UBL embarqué.
3. **XSD :** Validation structurelle face aux schémas EN 16931 officiels.
4. **Schematron :** Application des règles métiers AFNOR (XP Z12-012).
5. **Métier :** Cohérence des SIREN, structure TVA, validation de calculs des totaux.
6. **Score :** Retour structuré d'erreurs au format RFC 7807 (JSON Problem details).

---

##  Sécurité, KMS & BYOK (Workload Encryption)

Afin de protéger les données fiscales hautement sensibles, `onefacture` chiffre les artefacts `raw_xml` et `raw_pdf` at-rest à l'aide d'enveloppes cryptographiques AES-256-GCM.

### Fonctionnement & Configuration du KMS HTTP
1. **HTTPKMSProvider :** Interroge un broker KMS via requêtes HTTP signées, sans embarquer de SDK cloud propriétaire.
2. **BYOK (Bring Your Own Key) :** L'identité workload est authentifiée par le KMS. La clé active est récupérée dynamiquement et n'est jamais persistée dans PostgreSQL.

```bash
# Activation locale / Clé statique
ONEFACTURE_ENCRYPTION_KEY_ID=local-v1
ONEFACTURE_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

# Activation KMS HTTP de production
ONEFACTURE_KMS_URL=https://kms-broker.example.com/onefacture
ONEFACTURE_KMS_TOKEN=workload-token-auth
```

L'API Gateway appelle ensuite les endpoints :
*   `GET /keys/active` : Récupère la clé active et son identifiant unique `key_id`.
*   `GET /keys/{key_id}` : Permet de décoder les anciennes enveloppes cryptées lors de la phase de rotation.

---

##  Démarrage Rapide (Quickstart)

### Prérequis
*   **Go 1.23+**
*   **Docker & Docker Compose**
*   **Python 3.10+** (pour le sidecar)
*   **Make**

### 1. Démarrer l'infrastructure locale
```bash
git clone https://github.com/yawo/onefacture.git
cd onefacture
make dev  # Lance Postgres, Redis, le sidecar de validation Python et l'API Go
```
L'API est alors accessible sur :
*   **API principale :** `http://localhost:8080`
*   **Documentation interactive (Scalar) :** `http://localhost:8080/docs`
*   **Fichier OpenAPI 3.1 :** `http://localhost:8080/openapi.json`

### 2. Migrations de base de données
```bash
make migrate-up
```

### 3. Exécution des tests et validation locale
```bash
make test          # Exécute la suite de tests unitaires (Couverture min 35%)
make verify-local  # Exécute tous les gates d'acceptance locaux, smoke tests et actionlint
```

---

##  Cas d'usage Métier & Payloads Réels

### Avoir (Type `381`)
```json
{
  "profile": "EN16931",
  "type_code": "381",
  "number": "AV-2026-0001",
  "currency": "EUR",
  "issue_date": "2026-05-22T00:00:00Z",
  "seller": {
    "name": "Acme SAS",
    "siren": "732829320",
    "address": { "line1": "1 rue Cler", "postal_code": "75007", "city": "Paris", "country_code": "FR" }
  },
  "buyer": {
    "name": "Globex SAS",
    "siren": "552120222",
    "address": { "line1": "2 avenue Foch", "postal_code": "75116", "city": "Paris", "country_code": "FR" }
  },
  "lines": [
    { "description": "Avoir commercial", "quantity": 1, "unit_code": "C62", "unit_price": 250, "tax_rate": 20, "tax_category": "S" }
  ],
  "notes": [{ "subject": "credit_note", "content": "Avoir lie a la facture INV-2026-0007." }]
}
```

### Facture corrective (Type `384`)
```json
{
  "profile": "EN16931",
  "type_code": "384",
  "number": "COR-2026-0001",
  "currency": "EUR",
  "issue_date": "2026-05-22T00:00:00Z",
  "buyer_ref": "INV-2026-0007",
  "seller": {
    "name": "Acme SAS",
    "siren": "732829320",
    "address": { "line1": "1 rue Cler", "postal_code": "75007", "city": "Paris", "country_code": "FR" }
  },
  "buyer": {
    "name": "Globex SAS",
    "siren": "552120222",
    "address": { "line1": "2 avenue Foch", "postal_code": "75116", "city": "Paris", "country_code": "FR" }
  },
  "lines": [
    { "description": "Correction prix unitaire", "quantity": 4, "unit_code": "HUR", "unit_price": 125, "tax_rate": 20, "tax_category": "S" }
  ],
  "notes": [{ "subject": "correction", "content": "Corrige le montant HT de la facture initiale." }]
}
```

### Résoudre un rejet et resoumettre
En cas de rejet métier de la PA (ex: SIREN destinataire erroné), soumettez la correction :
```bash
curl -X POST "http://localhost:8080/v1/invoices/{invoice_id}/retry" \
  -H "X-API-Key: $ONEFACTURE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"resolution_hint":"SIREN acheteur corrige dans ERP"}'
```

---

##  Preuves d'Acceptation Externes (External Acceptance)

Afin de valider le fonctionnement bout-en-bout face aux sandboxes réelles et aux registres tiers, nous utilisons des **Acceptance Gates externes**.

Toutes les variables requises (tokens d'API réels, endpoints des brokers KMS, URL de staging, etc.) doivent être configurées dans GitHub Actions ou fournies localement avant d'exécuter la collection de preuves.

### Commandes opérationnelles d'acceptance
```bash
# 1. Vérification locale de la configuration
make check-external-env
make check-github-external-config GITHUB_REPO=yawo/onefacture

# 2. Exécuter un gate d'acceptation ciblé
make verify-live-pa            # Test les connexions Chorus, Docaposte et Pennylane
make verify-public-sandbox     # Valide l'onboarding et le parcours quickstart
make verify-sdk-registries     # Valide l'installation npm et PyPI des SDK
make verify-kms-broker         # Valide l'authentification et l'échange de clés KMS
make verify-outcome-metrics    # Valide l'algorithme d'amélioration de la résoumission
make verify-external           # Lance TOUS les gates externes simultanément

# 3. Collecter et valider un bundle de preuves signé
make collect-external-evidence STAMP=2026-05-27
make verify-external-evidence BUNDLE=docs/operations/evidence/2026-05-27-external-acceptance
```

---

##  Feuille de Route (Roadmap & Cycle de Développement)

*   **Phase 0 : Spécifications & Normes (AFNOR, XSD, Schematron)** - 🟢 Terminé
*   **Phase 1 : Fondations Core (Modèles Go, DB, Sidecar Python)** - 🟢 Terminé
*   **Phase 2 : API Gateway (CRUD, OpenAPI 3.1, Scalar docs)** - 🟢 Terminé
*   **Phase 3 : Adaptateurs PA (Chorus, Pennylane, Docaposte Sandboxes)** - 🟢 Terminé
*   **Phase 4 : Workers Asynchrones (Redis Streams, Webhooks HMAC, Webhook Inspector)** - 🟢 Terminé
*   **Phase 5 : Expérience Développeur Premium & Production Ready (Multi-jurisdictions, BYOK/KMS, SDKs, Helm deploy)** - 🟢 Terminé

---

##  Licence (License)

Ce projet est distribué sous licence **Server Side Public License (SSPL)**. Voir le fichier `LICENSE` pour plus de détails.

This project is licensed under the **Server Side Public License (SSPL)**. See the `LICENSE` file for details.
