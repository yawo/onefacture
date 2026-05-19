# AGENTS.md

```markdown
# AGENTS.md — Guide de construction de onefacture par un agent IA

## 1. Vision & Contexte

**onefacture** est une API unifiée open source qui abstrait la complexité
des Plateformes Agréées (PA) françaises pour la facturation électronique B2B.

### Contexte réglementaire
- Depuis le 1er septembre 2026, toute entreprise française assujettie à la TVA
  doit émettre et recevoir ses factures électroniques via une PA immatriculée
  par la DGFiP (schéma dit "en Y"). [web:7][web:12]
- Plus de 108 PA sont immatriculées (Sage, SAP, Axway, Pagero, Serensia, Azopio,
  Chaintrust, etc.), chacune avec sa propre API propriétaire. [web:21][web:27]
- Les formats obligatoires sont Factur-X (PDF/A-3 + XML), UBL 2.1 et CII,
  conformes à la norme européenne EN 16931. [web:3][web:26]
- Les normes AFNOR structurent le dispositif :
  - **XP Z12-012** : formats et profils des messages factures
  - **XP Z12-013** : API standard pour interfacer SI entreprise ↔ PA
  - **XP Z12-014** : cas d'usage B2B [web:19][web:25]

### Problème résolu
Les PA n'exposent pas toutes la norme XP Z12-013 de la même façon.
Chaque intégrateur doit maîtriser N APIs propriétaires. onefacture absorbe
cette complexité en exposant une seule API normée, ergonomique et évolutive.

---

## 2. Architecture cible

```
┌─────────────────────────────────────────────────┐
│              Client / ERP / SaaS                │
│         (Sage, Odoo, custom app...)             │
└───────────────────┬─────────────────────────────┘
                    │  REST API onefacture (OpenAPI 3.1)
                    ▼
┌─────────────────────────────────────────────────┐
│              onefacture API Gateway             │
│  - Auth (OAuth2 / API Key)                      │
│  - Routing logique PA                           │
│  - Validation Factur-X / UBL / CII              │
│  - Gestion statuts cycle de vie                 │
│  - Webhook & événements                         │
│  - Annuaire (lookup PA destinataire)            │
└───────────────────┬─────────────────────────────┘
                    │
      ┌─────────────┼─────────────┐
      ▼             ▼             ▼
  Adapter PA1   Adapter PA2   Adapter PAN
  (Sage)        (Axway)       (Pagero)
  XP Z12-013    propriétaire  REST custom
```

### Stack recommandée
| Composant         | Technologie                        |
|-------------------|------------------------------------|
| API Gateway       | Python (FastAPI) ou Go (Fiber)     |
| Adapters PA       | Python async (httpx) ou Go         |
| Validation XML    | lxml + schematron (AFNOR publics)  |
| Génération PDF/A3 | pypdf / fpdf2 + attachement XML    |
| Queue / Events    | Redis Streams ou NATS              |
| Base de données   | PostgreSQL (statuts, audit trail)  |
| Annuaire PA       | Cache Redis + sync PPF             |
| Auth              | OAuth2 / JWT (Keycloak ou built-in)|
| Docs              | OpenAPI 3.1 + Redoc / Scalar       |
| Tests             | pytest + testcontainers            |
| CI/CD             | GitHub Actions                     |
| Conteneurs        | Docker Compose (dev) + Helm (prod) |

---

## 3. Modèle de données core

### Invoice (ressource centrale)
```json
{
  "id": "uuid",
  "status": "DRAFT|SUBMITTED|ACCEPTED|REJECTED|PAID|CANCELLED",
  "format": "FACTURX|UBL|CII",
  "profile": "MINIMUM|BASIC_WL|BASIC|EN16931|EXTENDED",
  "seller": { "siren": "...", "vat_number": "...", "name": "...", "address": {} },
  "buyer": { "siren": "...", "vat_number": "...", "name": "...", "address": {} },
  "lines": [{ "description": "...", "quantity": 1, "unit_price": 100.0, "vat_rate": 0.20 }],
  "total_excl_tax": 100.0,
  "total_vat": 20.0,
  "total_incl_tax": 120.0,
  "issue_date": "2026-09-01",
  "due_date": "2026-09-30",
  "pa_id": "SAGE|AXWAY|PAGERO|...",
  "pa_ref": "ref_interne_pa",
  "lifecycle": [],
  "attachments": [],
  "raw_xml": "base64...",
  "raw_pdf": "base64..."
}
```

### LifecycleEvent
```json
{
  "event_type": "RECEIVED|APPROVED|REFUSED|PAID",
  "timestamp": "ISO8601",
  "actor": "buyer|seller|pa|dgfip",
  "comment": "..."
}
```

---

## 4. Endpoints API (OpenAPI 3.1)

### Invoices
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| POST   | /v1/invoices                      | Créer + émettre une facture           |
| GET    | /v1/invoices                      | Lister factures (filtres, pagination) |
| GET    | /v1/invoices/{id}                 | Détail d'une facture                  |
| PUT    | /v1/invoices/{id}                 | Mettre à jour (DRAFT seulement)       |
| DELETE | /v1/invoices/{id}                 | Annuler / mettre en corbeille         |
| POST   | /v1/invoices/{id}/submit          | Soumettre à la PA                     |
| POST   | /v1/invoices/{id}/cancel          | Émettre avoir d'annulation            |
| GET    | /v1/invoices/{id}/lifecycle       | Historique statuts cycle de vie       |
| GET    | /v1/invoices/{id}/download        | Télécharger PDF/A-3 ou XML            |

### Reception
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| GET    | /v1/inbox                         | Factures reçues                       |
| POST   | /v1/inbox/{id}/acknowledge        | Accuser réception                     |
| POST   | /v1/inbox/{id}/approve            | Approuver la facture                  |
| POST   | /v1/inbox/{id}/reject             | Rejeter avec motif                    |

### Validation
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| POST   | /v1/validate                      | Valider un fichier Factur-X/UBL/CII   |
| POST   | /v1/convert                       | Convertir entre formats               |

### Annuaire
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| GET    | /v1/directory/lookup              | Trouver la PA d'un SIREN destinataire |

### PA Management
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| GET    | /v1/platforms                     | Lister PA supportées                  |
| POST   | /v1/platforms/{id}/connect        | Connecter credentials PA              |
| GET    | /v1/platforms/{id}/status         | Santé / connectivité PA               |

### E-reporting
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| POST   | /v1/ereporting/transactions       | Déclarer transactions B2C / export    |
| POST   | /v1/ereporting/payments           | Déclarer données paiement             |

### Webhooks
| Method | Path                              | Description                           |
|--------|-----------------------------------|---------------------------------------|
| POST   | /v1/webhooks                      | Créer endpoint webhook                |
| GET    | /v1/webhooks                      | Lister webhooks                       |
| DELETE | /v1/webhooks/{id}                 | Supprimer webhook                     |

---

## 5. Adapters PA — Pattern

Chaque adapter implémente une interface commune :

```python
class PAAdapter(ABC):
    """Interface commune pour tous les adapters Plateforme Agréée."""

    @abstractmethod
    async def submit_invoice(self, invoice: Invoice) -> PASubmitResult:
        """Émet une facture vers la PA."""

    @abstractmethod
    async def get_status(self, pa_ref: str) -> LifecycleEvent:
        """Récupère le statut d'une facture côté PA."""

    @abstractmethod
    async def fetch_received(self) -> list[Invoice]:
        """Récupère les factures reçues depuis la PA."""

    @abstractmethod
    async def send_lifecycle_event(self, pa_ref: str, event: LifecycleEventType) -> bool:
        """Envoie un statut cycle de vie (approbation, rejet, paiement)."""

    @abstractmethod
    async def health_check(self) -> bool:
        """Vérifie la disponibilité de la PA."""
```

### Adapters à implémenter (priorité)
1. **Chorus Pro / PPF** (Portail Public de Facturation — référence)
2. **Sage PA**
3. **Axway**
4. **Pagero**
5. **B2Brouter** (PA agréée, API REST documentée) [web:15]
6. **Super PDP** (API simple et accessible) [web:6]
7. Generique basé sur XP Z12-013 AFNOR (auto-détection)

---

## 6. Validation Factur-X

L'agent doit implémenter un pipeline de validation en couches :

```
1. Validation PDF/A-3   → vérifier conteneur PDF valide
2. Extraction XML       → extraire fichier XML embarqué
3. Validation XSD       → schéma XSD EN 16931 / profil (MINIMUM → EXTENDED)
4. Validation Schematron → règles métier AFNOR XP Z12-012
5. Validation métier    → SIREN valide, TVA cohérente, totaux corrects
6. Score de conformité  → retourner liste d'erreurs structurées (code + message + path)
```

---

## 7. Génération Factur-X

L'agent doit implémenter la génération complète :

```
1. Recevoir payload JSON normalisé
2. Calculer totaux, TVA, arrondis
3. Générer XML CII (CrossIndustryInvoice) selon profil
4. Générer PDF/A-3 (layout paramétrable)
5. Embarquer XML dans PDF (conformément à PDF/A-3)
6. Retourner fichier binaire + métadonnées
```

---

## 8. Gestion des erreurs

Toutes les erreurs retournent un objet structuré :

```json
{
  "error": {
    "code": "VALIDATION_FAILED|PA_UNAVAILABLE|AUTH_ERROR|...",
    "message": "Message lisible",
    "details": [
      { "field": "seller.siren", "code": "INVALID_SIREN", "message": "SIREN invalide" }
    ],
    "request_id": "uuid",
    "timestamp": "ISO8601"
  }
}
```

---

## 9. Sécurité

- **Auth** : OAuth2 (client_credentials flow) + API Key (header `X-API-Key`)
- **Chiffrement** : TLS 1.3 obligatoire ; données at-rest chiffrées (AES-256)
- **Isolation** : multi-tenancy par `organization_id` sur toutes les ressources
- **Audit** : log immuable de chaque action (émission, réception, statut)
- **RGPD** : endpoint `/v1/gdpr/export` et `/v1/gdpr/delete`
- **Rate limiting** : par API key (configurable par plan)

---

## 10. Open Source & Gouvernance

- Licence : **Apache 2.0**
- Repo : `github.com/onefacture/onefacture`
- Structure :
  ```
  onefacture/
  ├── api/           # FastAPI app, routes, schemas
  ├── adapters/      # Un dossier par PA
  ├── core/          # Validation, génération, modèles
  ├── workers/       # Tâches async (polling PA, webhooks)
  ├── migrations/    # Alembic / SQL
  ├── tests/         # pytest, fixtures, mocks PA
  ├── docs/          # OpenAPI spec, guides
  ├── docker/
  ├── .github/workflows/
  ├── AGENTS.md
  ├── ISSUES.md
  └── README.md
  ```
- **SDK** à générer automatiquement depuis OpenAPI : Python, TypeScript, PHP
- **Sandbox** intégrée avec PA mockées pour les développeurs

---

## 11. Instructions pour l'agent IA

L'agent doit exécuter les étapes dans cet ordre strict :

### Phase 0 — Recherche & Spécifications
- [ ] Lire et parser les normes AFNOR XP Z12-012, Z12-013, Z12-014
- [ ] Scraper la liste officielle des PA immatriculées (impots.gouv.fr)
- [ ] Lire la documentation API de chaque PA prioritaire
- [ ] Extraire les schémas XSD EN 16931 et Factur-X depuis FNFE-MPE

### Phase 1 — Fondations
- [ ] Initialiser le projet (structure, dépendances, CI/CD)
- [ ] Créer les modèles de données core (Pydantic v2)
- [ ] Implémenter la validation Factur-X (pipeline 6 étapes)
- [ ] Implémenter la génération Factur-X (tous profils)
- [ ] Écrire les tests unitaires correspondants

### Phase 2 — API Gateway
- [ ] Implémenter tous les endpoints `/v1/invoices`
- [ ] Implémenter `/v1/inbox`
- [ ] Implémenter `/v1/validate` et `/v1/convert`
- [ ] Implémenter auth OAuth2 + API Key
- [ ] Générer documentation OpenAPI 3.1

### Phase 3 — Adapters PA
- [ ] Implémenter l'interface `PAAdapter`
- [ ] Adapter Chorus Pro / PPF (référence)
- [ ] Adapter Super PDP
- [ ] Adapter B2Brouter
- [ ] Adapter Sage, Axway, Pagero
- [ ] Tests d'intégration (sandbox)

### Phase 4 — Features avancées
- [ ] Annuaire / lookup SIREN → PA
- [ ] E-reporting (transactions + paiements)
- [ ] Webhooks (émission, réception, statuts)
- [ ] Workers async (polling statuts, retry)
- [ ] SDK Python + TypeScript

### Phase 5 — Qualité & Lancement
- [ ] Tests de charge
- [ ] Documentation complète (guides, tutoriels)
- [ ] Sandbox publique
- [ ] Packaging Docker + Helm chart
```

