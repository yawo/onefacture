# Framework multi-juridiction

Le module `internal/jurisdiction` isole les profils pays/formats du coeur API.

## Profils initiaux

- `FR`: Factur-X EN16931, formats Factur-X, CII, UBL.
- `EU`: PEPPOL BIS Billing, format UBL.

## Extension

Ajouter un pays ne doit pas modifier les endpoints REST principaux:

1. Enregistrer un `jurisdiction.Profile`.
2. Brancher les regles de validation/generation propres au pays.
3. Ajouter les tests de mapping format/profil.

Cette structure prepare PEPPOL/ViDA sans coupler les regles pays aux handlers HTTP.
