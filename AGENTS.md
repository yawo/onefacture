# AGENTS.md — Guide de construction de onefacture

> Ce document est la source de vérité pour l'agent IA chargé de construire **onefacture**. Il définit la vision, l'architecture, les normes de qualité et la roadmap d'exécution.

---

## 1. Vision & Contexte

**onefacture** est une API unifiée open source qui abstrait la complexité des Plateformes Agréées (PA) françaises pour la facturation électronique B2B (Réforme 2026).

### Problème résolu
Depuis le 1er septembre 2026, toute entreprise française assujettie à la TVA doit émettre et recevoir ses factures via une PA immatriculée par la DGFiP (schéma en "Y"). Avec plus de 100 PA (Sage, SAP, Pennylane, Qonto, etc.), chacune possédant sa propre API propriétaire, l'intégration devient un cauchemar pour les ERP et SaaS. **onefacture** absorbe cette complexité en exposant une seule API normée (Factur-X / EN 16931), ergonomique et évolutive.

### Normes de référence
- **XP Z12-012** : formats et profils des messages factures.
- **XP Z12-013** : API standard pour interfacer SI entreprise ↔ PA.
- **XP Z12-014** : cas d'usage B2B.

---

## 2. Architecture Cible (Go-First)

Le système est conçu pour la performance, la concurrence (gestion des flux XML/PDF) et la robustesse.

```
┌─────────────────────────────────────────────────┐
│              Client / ERP / SaaS                │
│         (Sage, Odoo, custom app...)             │
└───────────────────┬─────────────────────────────┘
                    │  REST API onefacture (OpenAPI 3.1)
                    ▼
┌─────────────────────────────────────────────────┐
│              onefacture API Gateway             │
│  - Langage : Go 1.23+ (Fiber ou Chi)            │
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
| Couche | Technologie |
|---|---|
| **API Gateway** | Go (Fiber ou Chi) |
| **Adaptateurs PA** | Go (un package par PA) |
| **Validation Factur-X** | Go + sidecar Python (lxml/schematron) |
| **Stockage** | PostgreSQL + pgvector (audit trail, recherche) |
| **Messaging** | NATS ou Redis Streams |
| **Documentation** | OpenAPI 3.1 + Scalar |
| **CI/CD** | GitHub Actions |
| **Conteneurs** | Docker Compose (dev) + Helm (prod) |

---

## 3. Modèle de données Core (Unified Invoice)

L'Invoice est la ressource centrale, basée sur la norme **Factur-X 1.08 / EN 16931**.

### Structure Go (Conceptuelle)
```go
type Invoice struct {
    ID           string      `json:"id"`
    Status       Status      `json:"status"` // DRAFT, SUBMITTED, RECEIVED, etc.
    Profile      Profile     `json:"profile"` // MINIMUM, BASIC, EN16931, EXTENDED
    Seller       Party       `json:"seller"`
    Buyer        Party       `json:"buyer"`
    Lines        []Line      `json:"lines"`
    Totals       Totals      `json:"totals"`
    IssueDate    time.Time   `json:"issue_date"`
    DueDate      time.Time   `json:"due_date"`
    PAID         string      `json:"pa_id"`
    PARef        string      `json:"pa_ref"`
    RawXML       []byte      `json:"-"`
    RawPDF       []byte      `json:"-"`
}
```

---

## 4. Endpoints API (OpenAPI 3.1)

### Invoices & Reception
| Method | Path | Description |
|---|---|---|
| POST | `/v1/invoices` | Créer + émettre une facture |
| GET | `/v1/invoices/{id}` | Détail et statut actuel |
| POST | `/v1/invoices/{id}/submit` | Soumettre à la PA (si DRAFT) |
| GET | `/v1/inbox` | Lister les factures reçues |
| POST | `/v1/inbox/{id}/approve` | Approuver une facture reçue |

### Validation & Annuaire
| Method | Path | Description |
|---|---|---|
| POST | `/v1/validate` | Valider un fichier Factur-X/UBL/CII |
| GET | `/v1/directory/lookup` | Trouver la PA d'un SIREN destinataire |
| GET | `/v1/platforms` | Lister les PA supportées et leur santé |

---

## 5. Adapters PA — Interface Go

Chaque adaptateur implémente l'interface commune :

```go
type PAAdapter interface {
    Name() string
    Submit(ctx context.Context, inv *Invoice) (*SubmitResult, error)
    GetStatus(ctx context.Context, paRef string) (*LifecycleEvent, error)
    Webhook(ctx context.Context, payload []byte) (*WebhookEvent, error)
    HealthCheck(ctx context.Context) error
}
```

### Priorité d'implémentation
1. **Chorus Pro / PPF** (Référence d'État)
2. **Docaposte (SERES)** (~35% du marché)
3. **Pennylane** (Forte adoption PME)
4. **Cegid / Qonto** (ERP et Fintech)
5. **Super PDP / B2Brouter** (Accessibilité)

---

## 6. Pipeline de Validation & Génération

### Validation (6 couches)
1. **PDF/A-3** : Vérifier la validité du conteneur.
2. **Extraction** : Extraire le XML CII/UBL embarqué.
3. **XSD** : Valider contre les schémas EN 16931.
4. **Schematron** : Appliquer les règles métiers AFNOR (XP Z12-012).
5. **Métier** : SIREN, TVA cohérente, calculs des totaux.
6. **Score** : Retourner les erreurs structurées (code, path, message).

### Génération
1. Calcul des arrondis et taxes sur le payload JSON.
2. Génération du XML CII selon le profil choisi.
3. Génération du PDF/A-3 (layout paramétrable).
4. Injection du XML dans le PDF (Factur-X compliant).

---

## 7. Règles de Qualité & Contribution

### Code & Style
- **Go 1.23+** : Préférer la bibliothèque standard.
- **Erreurs** : Toujours wrapper avec du contexte (`fmt.Errorf("context: %w", err)`).
- **Tests** : Unitaires (logic) + Intégration (API/DB via Testcontainers). Couverture ≥ 80%.
- **Secrets** : Utiliser `ONEFACTURE_` prefix env vars. Jamais de hardcode.
- **Lint** : `golangci-lint` doit passer sans avertissement.

### Gestion des Erreurs (RFC 7807)
Les erreurs doivent être structurées :
```json
{
  "type": "https://onefacture.io/errors/validation-failed",
  "title": "Validation Failed",
  "status": 400,
  "detail": "The provided SIREN is invalid.",
  "instance": "/v1/invoices/123",
  "errors": [{ "field": "seller.siren", "code": "INVALID" }]
}
```

---

## 8. Instructions pour l'agent IA (Roadmap)

### Phase 0 — Recherche & Spécifications
- [x] Parser les normes AFNOR XP Z12-012/013.
- [x] Extraire les schémas XSD et Schematrons depuis FNFE-MPE.

### Phase 1 — Fondations (Go)
- [ ] Init projet, CI/CD GitHub Actions.
- [ ] Modèles Pydantic-like en Go (structs + tags).
- [ ] Implémenter le pipeline de validation (Sidecar Python).

### Phase 2 — API Gateway
- [ ] Endpoints `/v1/invoices` et `/v1/validate`.
- [ ] Auth API Key + Middleware logging.
- [ ] Documentation Scalar intégrée.

### Phase 3 — Adapters PA
- [ ] Interface `PAAdapter` et registre.
- [ ] Adapter Chorus Pro (Mocké puis réel).
- [ ] Adapter Pennylane/Docaposte.

### Phase 4 — Workers & Async
- [ ] Polling des statuts PA via NATS/Redis.
- [ ] Webhooks sortants pour notifier le client.

---

## 9. Sécurité & Gouvernance

- **Licence** : Apache 2.0.
- **Multi-tenancy** : Isolation stricte par `organization_id`.
- **Audit Log** : Chaque transition de statut doit être immuable en base.
- **GDPR** : Endpoints de suppression et d'export dédiés.
