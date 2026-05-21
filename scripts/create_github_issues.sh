#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "GITHUB_TOKEN is required" >&2
  exit 1
fi

OWNER_REPO="${1:-}"
if [[ -z "$OWNER_REPO" ]]; then
  echo "Usage: $0 <owner/repo>" >&2
  exit 1
fi

api() {
  curl -sS -X POST \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${OWNER_REPO}/issues" \
    -d "$1"
}

create_issue() {
  local title="$1" labels_json="$2" body="$3"
  payload=$(jq -n --arg t "$title" --arg b "$body" --argjson l "$labels_json" '{title:$t, body:$b, labels:$l}')
  echo "Creating: $title"
  api "$payload" | jq -r '.html_url // .message'
}

# Vague 1
create_issue "[Wave 1] Intégration Chorus Pro PISTE sandbox (round-trip complet)" '["adapter","priority:p0","wave:1"]' $'Implémenter OAuth2 PISTE, submit, status, mapping erreurs.\n\n**AC:** Round-trip sandbox + tests d’intégration.'
create_issue "[Wave 1] Intégration Docaposte sandbox" '["adapter","priority:p0","wave:1"]' $'Implémenter submit/status/webhook Docaposte.\n\n**AC:** tests sandbox verts.'
create_issue "[Wave 1] Intégration Pennylane sandbox" '["adapter","priority:p0","wave:1"]' $'Implémenter submit/status/webhook Pennylane.\n\n**AC:** round-trip automatisé.'
create_issue "[Wave 1] Idempotency-Key sur POST invoices et submit" '["api","reliability","priority:p0","wave:1"]' $'Ajouter le support et stockage persistant de la clé d’idempotence.\n\n**AC:** même clé => même résultat sans duplicat.'
create_issue "[Wave 1] Circuit breaker + retry policy soumission PA" '["reliability","worker","priority:p0","wave:1"]' $'Ajouter CB et retry exponentiel avec jitter.\n\n**AC:** dégradation contrôlée quand PA indisponible.'
create_issue "[Wave 1] Dead-letter queue pour échecs terminaux" '["infra","reliability","priority:p0","wave:1"]' $'DLQ + inspection + replay.\n\n**AC:** tout échec terminal rejouable.'
create_issue "[Wave 1] Annuaire SIREN avec cache TTL et fallback" '["api","directory","priority:p1","wave:1"]' $'Résolution PA par SIREN avec cache et provider secondaire.\n\n**AC:** P95<100ms en cache.'
create_issue "[Wave 1] Override routage PA par organisation" '["api","multitenancy","priority:p1","wave:1"]' $'Règles de routage forcé tenant-level + audit trail.\n\n**AC:** règles appliquées et traçables.'
create_issue "[Wave 1] Sandbox publique onefacture" '["dx","infra","priority:p0","wave:1"]' $'Déployer sandbox publique avec PA mockées.\n\n**AC:** quickstart externe <10min.'
create_issue "[Wave 1] Onboarding 5 minutes to first invoice" '["dx","docs","priority:p1","wave:1"]' $'Guide copy/paste + examples + webhook demo.\n\n**AC:** parcours validé sur compte vierge.'

# Vague 2
create_issue "[Wave 2] Publier SDK Python sur PyPI" '["dx","sdk","priority:p1","wave:2"]' $'Pipeline génération/publication.\n\n**AC:** pip install onefacture.'
create_issue "[Wave 2] Publier SDK TypeScript sur npm" '["dx","sdk","priority:p1","wave:2"]' $'Pipeline génération/publication npm.\n\n**AC:** npm install @onefacture/sdk.'
create_issue "[Wave 2] CLI onefacture doctor" '["dx","tooling","priority:p2","wave:2"]' $'Diagnostic config/API/reachability.\n\n**AC:** rapport terminal exploitable.'
create_issue "[Wave 2] Trace ID sur toutes les réponses API" '["api","observability","priority:p1","wave:2"]' $'Injecter X-Request-ID corrélé aux logs.\n\n**AC:** corrélation e2e.'
create_issue "[Wave 2] Endpoint timeline facture" '["api","observability","priority:p1","wave:2"]' $'Historique + latences + retries.\n\n**AC:** timeline complète par facture.'
create_issue "[Wave 2] Webhook inspector UI" '["dx","webhooks","priority:p2","wave:2"]' $'Visualiser tentatives, erreurs et replay.\n\n**AC:** replay one-click.'
create_issue "[Wave 2] Erreurs RFC7807 enrichies" '["api","ux","priority:p1","wave:2"]' $'Ajouter remediation_hint/docs_url/retryable.\n\n**AC:** hints sur top erreurs.'
create_issue "[Wave 2] Pack exemples métier" '["docs","dx","priority:p2","wave:2"]' $'Exemples avoir/correction/rejet + snippets SDK.\n\n**AC:** docs interactives enrichies.'

# Vague 3
create_issue "[Wave 3] Pré-validation bulk" '["validation","enterprise","priority:p2","wave:3"]' $'Endpoint de pré-validation de lots.\n\n**AC:** rapport agrégé exportable.'
create_issue "[Wave 3] Score conformité par tenant" '["analytics","enterprise","priority:p2","wave:3"]' $'Score qualité hebdo/mensuel.\n\n**AC:** dashboard tendances.'
create_issue "[Wave 3] Assistant correction des rejets" '["automation","ux","priority:p2","wave:3"]' $'Proposer patch JSON pour resoumission.\n\n**AC:** amélioration du taux de succès.'
create_issue "[Wave 3] Chiffrement at-rest BYOK/KMS" '["security","compliance","priority:p1","wave:3"]' $'Intégration KMS + rotation + runbook.\n\n**AC:** conformité audit sécurité.'
create_issue "[Wave 3] mTLS + IP allowlist webhooks" '["security","webhooks","priority:p2","wave:3"]' $'mTLS optionnel et allowlist par endpoint.\n\n**AC:** handshake et contrôle IP validés.'
create_issue "[Wave 3] Framework multi-juridiction" '["architecture","future","priority:p3","wave:3"]' $'Abstraction profils pays (PEPPOL/ViDA-ready).\n\n**AC:** ajout d’un profil sans casser le core.'

echo "Done."
