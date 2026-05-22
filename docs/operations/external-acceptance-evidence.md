# External acceptance evidence checklist

Date: 2026-05-22

This checklist defines the proof bundle required before marking the external acceptance criteria in `docs/backlog/github-issues-vagues.md` complete. Local smokes are useful regression evidence, but they do not replace these live proofs.

## Evidence bundle layout

Store one dated bundle per acceptance run:

```text
docs/operations/evidence/YYYY-MM-DD-external-acceptance/
  live-pa.log
  public-sandbox.log
  sdk-registries.log
  kms-broker.log
  outcome-metrics.log
  all.log
  summary.md
```

Do not commit secrets, bearer tokens, raw invoices containing real personal data, or private KMS key material. Redact sensitive values while preserving command names, timestamps, target hostnames, HTTP statuses, package versions, and final success marker lines.

The scaffold is intentionally not valid evidence: `summary.md` fields must be filled, every gate command must be marked `PASS`, placeholder log text must be replaced with redacted live output, and each log must retain the command's terminal success marker.

Create the bundle scaffold:

```bash
make create-external-evidence STAMP=YYYY-MM-DD
```

Or collect a bundle directly from a live environment. This preflights the required environment, runs every external gate, writes redacted logs, fills `summary.md`, and verifies the bundle when all gates pass:

```bash
# Fill real values from docs/operations/external-acceptance.env.example first.
make check-external-env
make check-github-external-config GITHUB_REPO=yawo/onefacture
make check-external-env GATE=public-sandbox
make collect-external-evidence STAMP=YYYY-MM-DD
```

The manual GitHub Actions workflow `.github/workflows/external-acceptance.yml` uses the same collector for `gate=all` and uploads the generated evidence bundle as an artifact, including partial redacted logs when a gate fails.

Validate the collector locally without contacting external services:

```bash
make smoke-external-env
make smoke-github-external-config
make smoke-external-evidence-collector
make smoke-external-evidence-review
```

## Required gate evidence

| Gate | Command | Required proof |
|---|---|---|
| `live-pa` | `make verify-live-pa` | One successful strict run against Chorus PISTE, Docaposte and Pennylane sandboxes with `ONEFACTURE_REQUIRE_LIVE_PA=true`; include adapter subtest names, submit success, non-empty PA refs and status retrieval. |
| `public-sandbox` | `make verify-public-sandbox` | One successful run against the public sandbox URL using a fresh sandbox credential; include `/healthz`, `/v1/sandbox/credentials`, invoice submit and timeline checks. |
| `sdk-registries` | `make verify-sdk-registries` | Fresh environment install from public PyPI and npm registries; include package versions and successful Python/Node import smoke output. |
| `kms-broker` | `make verify-kms-broker` | Successful check against the deployed KMS broker active-key endpoint; include key id, decoded key length validation and separate rotation/audit evidence in `summary.md`. |
| `outcome-metrics` | `make verify-outcome-metrics` | Successful deployed analytics check with `retried_invoices >= ONEFACTURE_MIN_RETRIED_INVOICES` and `success_rate > ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE`; include metric name, retried count, accepted-after-retry count, success rate and baseline rate. |
| `all` | `make verify-external` | One aggregate run that executes every external gate above in the same environment window; include the full log in `all.log` and keep the individual gate logs for easier review. |

The verifier requires these success markers in the logs: live PA Go test `PASS` or package `ok`, `Sandbox smoke test passed`, `PyPI onefacture install ok`, `npm @onefacture/sdk install ok`, `KMS active key ok`, and `outcome metric ok`. `all.log` must contain every marker.

For SDK registry evidence, first publish through `.github/workflows/sdk-publish.yml` after `make verify-sdk` passes, then include evidence links for the GitHub Actions run, the public PyPI `onefacture` package page, and the public npm `@onefacture/sdk` package page. The registry proof is not accepted if it only installs local tarballs or local source directories.

## Summary requirements

`summary.md` must include:

- Commit SHA and branch tested; the commit SHA must match the repository `HEAD` when `make verify-external-evidence` or `make review-external-evidence` is run.
- Exact command run for each gate.
- Environment target names, with secrets redacted.
- Non-placeholder operator and valid UTC timestamp (`YYYY-MM-DDTHH:MM:SSZ`).
- Pass/fail result for each gate.
- Explanation for any rerun.
- Links to deployment, CI run, package release pages or KMS audit record when applicable. The `Links:` field must include at least one non-placeholder evidence URL; `example.*`, `localhost`, and `127.0.0.1` are rejected.

Only after all five gates have current successful evidence should the corresponding external blockers in `docs/backlog/github-issues-vagues-completion-audit.md` be changed from partial to complete.

After human review, mark each externally proven issue as `covered_external` in `docs/backlog/github-issues-vagues-acceptance.json`, remove its `external_blockers`, and add `reviewed_evidence` with `bundle`, full lowercase `commit_sha`, UTC `reviewed_at` (`YYYY-MM-DDTHH:MM:SSZ`), and non-placeholder `reviewed_by`. In the real manifest, `reviewed_evidence.bundle` must point to an existing bundle directory under `docs/operations/evidence/`. Then update the review and completion audit documents before rerunning `make audit-backlog-completion`.

The review document must include a per-issue marker in the form `<number>. <title>: covered_external`, and the completion audit document must include `| <number> | covered_external |` for each externally completed issue. The manifest verifier fails if those markers are missing.

`make audit-backlog-completion` re-runs the external evidence bundle verifier for every `covered_external` bundle and fails if `reviewed_evidence.commit_sha` no longer matches the current `HEAD`.

Validate a collected bundle before review:

```bash
make verify-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance
make review-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance
```

The review helper prints the manifest template plus the exact per-issue review markers and completion-audit status rows to apply after human review.

Validate the verifier itself with synthetic positive and negative fixtures:

```bash
make verify-external-evidence-smoke
```

Audit strict backlog completion. This command fails until every manifest issue is locally covered or a valid external evidence bundle has been reviewed and the manifest, review and completion audit have been updated accordingly:

```bash
make audit-backlog-completion
make audit-backlog-completion BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance
```
