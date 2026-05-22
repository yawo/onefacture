# External acceptance gates

Ce runbook decrit les preuves externes restantes pour fermer les criteres qui ne peuvent pas etre valides uniquement dans le repo.

Checklist de preuves a conserver: `docs/operations/external-acceptance-evidence.md`.
Template env sans secrets reels: `docs/operations/external-acceptance.env.example`.
Matrice de fermeture par issue: `docs/operations/external-closure-matrix.md`.

## Workflow GitHub

Workflow manuel: `.github/workflows/external-acceptance.yml`
Le workflow preflight les variables du gate choisi avec `scripts/check_external_acceptance_env.sh`. Le choix `all` execute `scripts/collect_external_acceptance_evidence.sh` et publie le bundle `docs/operations/evidence/*-external-acceptance` comme artifact GitHub Actions, meme si une gate echoue, pour conserver les logs rediges de diagnostic.

Entrer `gate`:

- `all`: execute tous les gates ci-dessous.
- `live-pa`: round-trips Chorus, Docaposte et Pennylane sandbox.
- `public-sandbox`: quickstart complet sur l'URL sandbox publique.
- `sdk-registries`: installation depuis PyPI et npm publics.
- `kms-broker`: verification du broker KMS HTTP.
- `outcome-metrics`: verification de la metrique de resoumission apres rejet.

## Variables GitHub Actions

Configurer dans repository variables:

- `ONEFACTURE_CHORUS_BASE_URL`
- `ONEFACTURE_DOCAPOSTE_BASE_URL`
- `ONEFACTURE_PENNYLANE_BASE_URL`
- `ONEFACTURE_SANDBOX_URL`
- `ONEFACTURE_KMS_URL`
- `ONEFACTURE_PROD_API_URL`
- `ONEFACTURE_EVIDENCE_LINKS` (URL vers le run CI, deploiement, releases package ou audit KMS prouvant l'execution)
- `ONEFACTURE_EVIDENCE_OPERATOR` (optionnel dans le workflow GitHub: defaut `github.actor`; obligatoire en local; identite non generique de la personne ou equipe qui collecte et revoit les preuves)
- `ONEFACTURE_EVIDENCE_ENVIRONMENT` (obligatoire: nom non generique de l'environnement reel teste, par exemple `sandbox-piste`, `public-sandbox-prod` ou `prod-eu`)
- `ONEFACTURE_MIN_RETRIED_INVOICES` (optionnel, defaut script: `1`)
- `ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE` (taux de succes avant assistant de correction; requis pour prouver l'amelioration de l'item 21)

Configurer dans repository secrets:

- `ONEFACTURE_CHORUS_ACCESS_TOKEN`
- `ONEFACTURE_DOCAPOSTE_API_TOKEN`
- `ONEFACTURE_PENNYLANE_API_TOKEN`
- `ONEFACTURE_KMS_TOKEN` (si le broker KMS l'exige)
- `ONEFACTURE_PROD_API_KEY`
- `NPM_TOKEN` (workflow publication SDK)

PyPI utilise l'OIDC trusted publishing configure pour `pypa/gh-action-pypi-publish`.

Configuration GitHub CLI equivalente, apres avoir exporte les valeurs reelles en local:

```bash
gh variable set ONEFACTURE_CHORUS_BASE_URL --body "$ONEFACTURE_CHORUS_BASE_URL"
gh variable set ONEFACTURE_DOCAPOSTE_BASE_URL --body "$ONEFACTURE_DOCAPOSTE_BASE_URL"
gh variable set ONEFACTURE_PENNYLANE_BASE_URL --body "$ONEFACTURE_PENNYLANE_BASE_URL"
gh variable set ONEFACTURE_SANDBOX_URL --body "$ONEFACTURE_SANDBOX_URL"
gh variable set ONEFACTURE_KMS_URL --body "$ONEFACTURE_KMS_URL"
gh variable set ONEFACTURE_PROD_API_URL --body "$ONEFACTURE_PROD_API_URL"
gh variable set ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE --body "$ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE"
gh variable set ONEFACTURE_EVIDENCE_ENVIRONMENT --body "$ONEFACTURE_EVIDENCE_ENVIRONMENT"
gh variable set ONEFACTURE_EVIDENCE_OPERATOR --body "${ONEFACTURE_EVIDENCE_OPERATOR:-$(gh api user --jq .login)}"
gh variable set ONEFACTURE_EVIDENCE_LINKS --body "$ONEFACTURE_EVIDENCE_LINKS"
gh secret set ONEFACTURE_CHORUS_ACCESS_TOKEN --body "$ONEFACTURE_CHORUS_ACCESS_TOKEN"
gh secret set ONEFACTURE_DOCAPOSTE_API_TOKEN --body "$ONEFACTURE_DOCAPOSTE_API_TOKEN"
gh secret set ONEFACTURE_PENNYLANE_API_TOKEN --body "$ONEFACTURE_PENNYLANE_API_TOKEN"
gh secret set ONEFACTURE_PROD_API_KEY --body "$ONEFACTURE_PROD_API_KEY"
gh secret set NPM_TOKEN --body "$NPM_TOKEN"
# Si le broker KMS exige un token:
gh secret set ONEFACTURE_KMS_TOKEN --body "$ONEFACTURE_KMS_TOKEN"
make check-github-external-config GITHUB_REPO=yawo/onefacture
```

## Publication SDKs

Avant de fermer les items 11-12:

1. Verifier les artefacts locaux: `make verify-sdk`.
2. Configurer PyPI trusted publishing pour le package `onefacture` avec le workflow `.github/workflows/sdk-publish.yml` et l'environnement GitHub attendu par PyPI.
3. Configurer le secret `NPM_TOKEN` avec le droit de publier le package public `@onefacture/sdk`.
4. Lancer le workflow manuel `.github/workflows/sdk-publish.yml` avec `publish_python=true` et `publish_typescript=true`.
5. Conserver dans `ONEFACTURE_EVIDENCE_LINKS` les URLs du run GitHub Actions, de la page PyPI `onefacture` et de la page npm `@onefacture/sdk`.
6. Relancer `make verify-sdk-registries` depuis un environnement frais; la preuve attendue contient `PyPI onefacture install ok` et `npm @onefacture/sdk install ok`.
7. Inclure le log `sdk-registries.log` dans le bundle collecte par `make collect-external-evidence STAMP=YYYY-MM-DD`.

## Commandes locales equivalentes

```bash
# Remplir les valeurs depuis docs/operations/external-acceptance.env.example.
make create-external-evidence STAMP=YYYY-MM-DD
make check-external-env
make check-github-external-config GITHUB_REPO=yawo/onefacture
make check-external-env GATE=public-sandbox
make verify-live-pa
make verify-public-sandbox
make verify-sdk-registries
make verify-kms-broker
make verify-outcome-metrics
make verify-external
make collect-external-evidence STAMP=YYYY-MM-DD
make verify-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance
make review-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance
make audit-backlog-completion BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance
```

Le bundle collecte doit conserver le `Commit SHA` correspondant au `HEAD` du repo au moment de `make verify-external-evidence` ou `make review-external-evidence`. Apres revue humaine des logs rediges, marquer chaque item externe prouve en `covered_external` dans le manifest avec `reviewed_evidence.bundle`, `reviewed_evidence.commit_sha`, `reviewed_evidence.reviewed_at` et `reviewed_evidence.reviewed_by`, puis mettre a jour la review et l'audit de completion avant de relancer `make audit-backlog-completion`.

## Criteres couverts

- Items 1-3: `make verify-live-pa`.
- Items 9-10: `make verify-public-sandbox`.
- Items 11-12: `make verify-sdk-registries`.
- Item 21: `make verify-outcome-metrics`.
- Item 22: `make verify-kms-broker`.

Ces commandes doivent produire une sortie verte sur les services reels avant de marquer les criteres externes comme termines.
