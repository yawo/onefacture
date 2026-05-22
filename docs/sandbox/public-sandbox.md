# Sandbox publique onefacture

Ce runbook decrit l'instance publique de demonstration a provisionner pour la vague 1.

## Objectif

Permettre a un developpeur externe d'obtenir des credentials de test et d'emettre une facture en moins de 10 minutes, sans credentials PA reels.

## Configuration recommandee

- Adapter PA par defaut: `mock`.
- Sidecar validation actif.
- PostgreSQL et Redis managés.
- Rate limit faible par cle de test.
- Donnees reinitialisees regulierement.

## Variables minimales

```bash
ONEFACTURE_PUBLIC_BASE_URL=https://sandbox.onefacture.io
ONEFACTURE_HASH_PEPPER=<secret>
ONEFACTURE_DB_DSN=<postgres>
ONEFACTURE_REDIS_ADDR=<redis>
```

## Deploiement Helm

```bash
helm upgrade --install onefacture-sandbox deploy/helm/onefacture \
  -f deploy/helm/onefacture/values-sandbox.yaml \
  --set postgres.dsn='postgres://user:pass@host:5432/onefacture?sslmode=require' \
  --set secrets.hashPepper='<random-32-byte-secret>' \
  --set ingress.hosts[0].host='sandbox.onefacture.io'
```

Apres deploiement, configurer la variable GitHub Actions `ONEFACTURE_SANDBOX_URL` avec l'URL publique puis lancer `.github/workflows/sandbox-smoke.yml`.

## Parcours quickstart a verifier

1. Generer une organisation et une cle avec `POST /v1/sandbox/credentials`.
2. Recuperer la cle API `ofx_...` retournee.
3. Appeler `onefacture doctor`.
4. Appeler `POST /v1/invoices` avec `Idempotency-Key`.
5. Consulter `GET /v1/invoices/{id}/timeline`.
6. Declarer un webhook et verifier l'inspector `/tools/webhook-inspector`.

## Critere d'acceptation

Le chronometrage demarre quand le developpeur ouvre la page sandbox et s'arrete quand une facture mockee apparait en `SUBMITTED`.

## Verification continue

- Script local: `bash scripts/smoke_public_sandbox.sh`
- Workflow GitHub Actions: `.github/workflows/sandbox-smoke.yml`
- Variables requises: `ONEFACTURE_SANDBOX_URL` cote repo/Actions.
