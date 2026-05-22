# BYOK/KMS runbook

## Objectif

Chiffrer les donnees sensibles at-rest avec une enveloppe AES-256-GCM et un fournisseur de cle remplacable par KMS.

## Interface locale

- `internal/security.KeyProvider` fournit la cle active et son `key_id`.
- `internal/security.KeyResolver` permet de dechiffrer les anciennes enveloppes par `key_id` apres rotation.
- `internal/security.HTTPKMSProvider` branche un broker KMS HTTP sans dependance SDK cloud dans le binaire API.
- `internal/security.Encryptor` chiffre avec AES-256-GCM.
- L'AAD doit inclure au minimum `organization_id` et le type de ressource.
- `internal/storage.InvoiceRepo` chiffre `raw_xml` et `raw_pdf` quand `ONEFACTURE_KMS_URL` ou `ONEFACTURE_ENCRYPTION_KEY` est configuree.
- `internal/storage.InspectEncryptedArtifact` expose `encrypted`, `field` et `key_id` sans dechiffrer, pour auditer la rotation.

## Configuration locale

```bash
ONEFACTURE_ENCRYPTION_KEY_ID=local-v1
ONEFACTURE_ENCRYPTION_KEY=<32-byte-key-hex-or-base64>
```

## Configuration KMS HTTP

```bash
ONEFACTURE_KMS_URL=https://kms-broker.example.com/onefacture
ONEFACTURE_KMS_TOKEN=<workload-token>
```

Le provider HTTP appelle:

- `GET /keys/active` pour la cle active, reponse `{"key_id":"kms-v2","key":"<32-byte-key-hex-or-base64>"}`.
- `GET /keys/{key_id}` pour dechiffrer une ancienne enveloppe apres rotation.

Le broker KMS doit authentifier l'identite workload, journaliser l'usage cote KMS et ne jamais stocker les cles dans PostgreSQL.

Verification externe:

```bash
ONEFACTURE_KMS_URL=https://kms-broker.example.com/onefacture \
ONEFACTURE_KMS_TOKEN=<workload-token> \
scripts/verify_external_acceptance.sh kms-broker
```

## Rotation

1. Ajouter une nouvelle cle dans le provider KMS.
2. Basculer `ActiveKey` vers le nouveau `key_id`.
3. Garder les anciennes cles resolvables par `key_id` pendant la rotation.
4. Rechiffrer progressivement les enveloppes anciennes.
5. Conserver les anciennes cles en decrypt-only jusqu'a la fin de retention.
6. Auditer chaque lot de rotation dans `audit_log`.

## Audit

1. Echantillonner les colonnes `raw_xml` et `raw_pdf`.
2. Appeler `InspectEncryptedArtifact` sur chaque artefact.
3. Verifier que `encrypted=true` et que `key_id` appartient a la fenetre de rotation autorisee.
4. Journaliser le resultat d'audit dans `audit_log` avec le `key_id`, le champ et le lot de rotation.

## Production

Le provider statique est reserve aux tests et au developpement local. En production, brancher un provider KMS qui:

- recupere la cle par identite workload;
- journalise `key_id`, version et usage;
- refuse une cle hors rotation active;
- ne persiste jamais la cle en clair dans PostgreSQL.
