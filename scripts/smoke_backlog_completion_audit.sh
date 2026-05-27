#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

expect_audit_failure() {
	local label="$1"
	local expected="$2"
	shift 2
	local stdout="$tmpdir/${label}.out"
	local stderr="$tmpdir/${label}.err"
	if "$@" >"$stdout" 2>"$stderr"; then
		echo "expected audit-backlog-completion to fail for $label" >&2
		exit 1
	fi
	if ! grep -Fq "$expected" "$stdout" "$stderr"; then
		echo "audit failure for $label did not include expected text: $expected" >&2
		exit 1
	fi
}

make_valid_bundle() {
	local bundle="$1"
	local commit_sha
	commit_sha="$(git rev-parse HEAD)"
	mkdir -p "$bundle"
	cat >"$bundle/live-pa.log" <<'EOF'
PASS
ok  	github.com/yawo/onefacture/internal/adapters/live	0.123s
EOF
	cat >"$bundle/public-sandbox.log" <<'EOF'
Sandbox smoke test passed
EOF
	cat >"$bundle/sdk-registries.log" <<'EOF'
PyPI onefacture install ok
npm @onefacture/sdk install ok
EOF
	cat >"$bundle/kms-broker.log" <<'EOF'
KMS active key ok: redacted-key-id
EOF
	cat >"$bundle/outcome-metrics.log" <<'EOF'
outcome metric ok: retried=2 accepted_after_retry=1 success_rate=0.5
EOF
	cat >"$bundle/all.log" <<'EOF'
PASS
ok  	github.com/yawo/onefacture/internal/adapters/live	0.123s
Sandbox smoke test passed
PyPI onefacture install ok
npm @onefacture/sdk install ok
KMS active key ok: redacted-key-id
outcome metric ok: retried=2 accepted_after_retry=1 success_rate=0.5
EOF
	cat >"$bundle/summary.md" <<EOF
# External acceptance summary

Commit SHA: ${commit_sha}
Branch: main
Operator: local-smoke
Timestamp: 2026-05-22T00:00:00Z
Environment: redacted-live-targets

Commands:
- make verify-live-pa: PASS
- make verify-public-sandbox: PASS
- make verify-sdk-registries: PASS
- make verify-kms-broker: PASS
- make verify-outcome-metrics: PASS
- make verify-external: PASS

Reruns: none
Links: https://github.com/yawo/onefacture/actions/runs/123456789
EOF
}

expect_audit_failure "no-bundle" "BUNDLE not supplied" make audit-backlog-completion
expect_audit_failure "objective-line" "Objective: planifier, implementer et reviewer chaque issue de docs/backlog/github-issues-vagues.md." make audit-backlog-completion
expect_audit_failure "prompt-artifact-checklist" "Prompt-to-artifact checklist: 28 issues mapped in docs/backlog/github-issues-vagues-acceptance.json." make audit-backlog-completion
expect_audit_failure "local-verification-gate" "Local verification gate: make verify-local" make audit-backlog-completion
expect_audit_failure "source-acceptance-checklist" "Source acceptance checklist:" make audit-backlog-completion
expect_audit_failure "source-acceptance-criterion" "Round-trip sur sandbox Chorus validé end-to-end." make audit-backlog-completion
expect_audit_failure "source-description-checklist" "Source description checklist:" make audit-backlog-completion
expect_audit_failure "source-description-bullet" "Implémenter OAuth2 client credentials PISTE." make audit-backlog-completion
expect_audit_failure "no-bundle-next-steps" "make collect-external-evidence STAMP=YYYY-MM-DD" make audit-backlog-completion
expect_audit_failure "no-bundle-review-step" "make review-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance" make audit-backlog-completion
expect_audit_failure "issue-title-checklist" "Intégration Chorus Pro PISTE sandbox" make audit-backlog-completion

valid_bundle="$tmpdir/valid-evidence"
make_valid_bundle "$valid_bundle"
expect_audit_failure \
	"valid-bundle-review-map" \
	"Verified external evidence is ready for review" \
	make audit-backlog-completion BUNDLE="$valid_bundle"
expect_audit_failure \
	"valid-bundle-path-map" \
	"$valid_bundle" \
	make audit-backlog-completion BUNDLE="$valid_bundle"
expect_audit_failure \
	"valid-bundle-review-command" \
	"Review command: make review-external-evidence BUNDLE=$valid_bundle" \
	make audit-backlog-completion BUNDLE="$valid_bundle"
expect_audit_failure \
	"valid-bundle-gate-map" \
	"#01 Intégration Chorus Pro PISTE sandbox (round-trip complet) | gate: verify-live-pa" \
	make audit-backlog-completion BUNDLE="$valid_bundle"

covered_manifest="$tmpdir/covered-external.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  issue = data.fetch("issues").find { |candidate| candidate.fetch("number") == 1 }
  issue["status"] = "covered_external"
  issue["external_blockers"] = []
  issue.delete("reviewed_evidence")
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' docs/backlog/github-issues-vagues-acceptance.json "$covered_manifest"

expect_audit_failure \
	"covered_external-no-evidence" \
	"covered_external but has no reviewed_evidence" \
	env MANIFEST_PATH="$covered_manifest" ruby scripts/verify_backlog_acceptance_manifest.rb

external_as_local_manifest="$tmpdir/external-gated-covered-local.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  issue = data.fetch("issues").find { |candidate| candidate.fetch("number") == 1 }
  issue["status"] = "covered_local"
  issue["external_blockers"] = []
  issue.delete("reviewed_evidence")
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' docs/backlog/github-issues-vagues-acceptance.json "$external_as_local_manifest"

expect_audit_failure \
	"external-gated-covered-local" \
	"uses external gate command but is marked covered_local" \
	env MANIFEST_PATH="$external_as_local_manifest" ruby scripts/verify_backlog_acceptance_manifest.rb

wrong_external_gate_manifest="$tmpdir/wrong-external-gate.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  issue = data.fetch("issues").find { |candidate| candidate.fetch("number") == 1 }
  issue["external_gate"] = "verify-public-sandbox"
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' docs/backlog/github-issues-vagues-acceptance.json "$wrong_external_gate_manifest"

expect_audit_failure \
	"wrong-external-gate" \
	"external_gate is not listed in verification_commands" \
	env MANIFEST_PATH="$wrong_external_gate_manifest" ruby scripts/verify_backlog_acceptance_manifest.rb

wrong_outcome_status_manifest="$tmpdir/wrong-outcome-status.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  issue = data.fetch("issues").find { |candidate| candidate.fetch("number") == 1 }
  issue["status"] = "partial_outcome_external"
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' docs/backlog/github-issues-vagues-acceptance.json "$wrong_outcome_status_manifest"

expect_audit_failure \
	"wrong-outcome-status" \
	"cannot use partial_outcome_external" \
	env MANIFEST_PATH="$wrong_outcome_status_manifest" ruby scripts/verify_backlog_acceptance_manifest.rb

generic_outcome_status_manifest="$tmpdir/generic-outcome-status.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  issue = data.fetch("issues").find { |candidate| candidate.fetch("number") == 21 }
  issue["status"] = "partial_external"
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' docs/backlog/github-issues-vagues-acceptance.json "$generic_outcome_status_manifest"

expect_audit_failure \
	"generic-outcome-status" \
	"issue 21 must use partial_outcome_external" \
	env MANIFEST_PATH="$generic_outcome_status_manifest" ruby scripts/verify_backlog_acceptance_manifest.rb

ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  issue = data.fetch("issues").find { |candidate| candidate.fetch("number") == 1 }
  issue["reviewed_evidence"] = {
    "bundle" => "docs/operations/evidence/2026-05-22-external-acceptance",
    "commit_sha" => `git rev-parse HEAD`.strip,
    "reviewed_at" => "2026-05-22T00:00:00Z",
    "reviewed_by" => "local-smoke"
  }
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' "$covered_manifest" "$covered_manifest"

covered_review="$tmpdir/covered-review.md"
covered_audit="$tmpdir/covered-audit.md"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  File.open(ARGV.fetch(1), "w") do |file|
     file.puts "# Review fixture"
     file.puts "covered_external reviewed_evidence"
     file.puts "local-acceptance gofmt parse YAML"
     file.puts "go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12"
     file.puts "make check-github-external-config"
     file.puts "make smoke-github-external-config"
     file.puts "make verify-sdk-registries PyPI onefacture install failed npm @onefacture/sdk install failed"
     file.puts "## Titres source couverts"
    data.fetch("issues").each do |issue|
      file.puts "#{issue.fetch("number")}. #{issue.fetch("title")}: #{issue.fetch("status")}"
    end
  end
  File.open(ARGV.fetch(2), "w") do |file|
     file.puts "# Audit fixture"
     file.puts "covered_external reviewed_evidence.bundle"
     file.puts "docs/operations/external-acceptance.env.example"
     file.puts "docs/operations/external-closure-matrix.md"
     file.puts "local-acceptance gofmt parse YAML"
     file.puts "go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12"
     file.puts "make check-github-external-config"
     file.puts "make smoke-github-external-config"
     file.puts "make verify-sdk-registries PyPI onefacture install failed npm @onefacture/sdk install failed"
     file.puts "## Titres source couverts"
    data.fetch("issues").each do |issue|
      file.puts "#{issue.fetch("number")}. #{issue.fetch("title")}"
    end
    file.puts "## Criteres d'\''acceptation source couverts"
    current_issue = nil
    in_acceptance = false
    File.read("docs/backlog/github-issues-vagues.md").each_line do |line|
      if (match = line.match(/^### (\d+)\./))
        current_issue = match[1]
        in_acceptance = false
      elsif line.include?("**Critères d'\''acceptation**")
        in_acceptance = true
      elsif in_acceptance && (match = line.match(/^- (.+)$/))
        file.puts "#{current_issue}. #{match[1]}"
      end
    end
    file.puts "## Descriptions source couvertes"
    current_issue = nil
    in_acceptance = false
    File.read("docs/backlog/github-issues-vagues.md").each_line do |line|
      if (match = line.match(/^### (\d+)\./))
        current_issue = match[1]
        in_acceptance = false
      elsif line.include?("**Critères d'\''acceptation**")
        in_acceptance = true
      elsif current_issue && !in_acceptance && (match = line.match(/^- (.+)$/))
        file.puts "#{current_issue}. #{match[1]}"
      end
    end
    file.puts "| # | Status |"
    file.puts "|---|---|"
    data.fetch("issues").each do |issue|
      file.puts "| #{issue.fetch("number")} | #{issue.fetch("status")} |"
    end
  end
' "$covered_manifest" "$covered_review" "$covered_audit"

MANIFEST_PATH="$covered_manifest" REVIEW_PATH="$covered_review" AUDIT_PATH="$covered_audit" ruby scripts/verify_backlog_acceptance_manifest.rb >/dev/null

fully_reviewed_manifest="$tmpdir/fully-reviewed.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  commit_sha = `git rev-parse HEAD`.strip
  data.fetch("issues").each do |issue|
    next if issue.fetch("status") == "covered_local"

    issue["status"] = "covered_external"
    issue["external_blockers"] = []
    issue["reviewed_evidence"] = {
      "bundle" => ARGV.fetch(1),
      "commit_sha" => commit_sha,
      "reviewed_at" => "2026-05-22T00:00:00Z",
      "reviewed_by" => "local-smoke"
    }
  end
  File.write(ARGV.fetch(2), JSON.pretty_generate(data))
' docs/backlog/github-issues-vagues-acceptance.json "$valid_bundle" "$fully_reviewed_manifest"

docs_not_updated_review="$tmpdir/not-updated-review.md"
ruby -e '
  text = File.read("docs/backlog/github-issues-vagues-review.md")
  text = text.gsub(": covered_external", "")
  File.write(ARGV.fetch(0), text)
' "$docs_not_updated_review"

expect_audit_failure \
	"covered-external-docs-not-updated" \
	"review doc missing covered_external marker" \
	env MANIFEST_PATH="$fully_reviewed_manifest" REVIEW_PATH="$docs_not_updated_review" ruby scripts/audit_backlog_completion.rb


reviewed_review="$tmpdir/review.md"
reviewed_audit="$tmpdir/audit.md"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  File.open(ARGV.fetch(1), "w") do |file|
     file.puts "# Review fixture"
     file.puts "covered_external reviewed_evidence"
     file.puts "local-acceptance gofmt parse YAML"
     file.puts "go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12"
     file.puts "make check-github-external-config"
     file.puts "make smoke-github-external-config"
     file.puts "make verify-sdk-registries PyPI onefacture install failed npm @onefacture/sdk install failed"
     file.puts "## Titres source couverts"
    data.fetch("issues").each do |issue|
      file.puts "#{issue.fetch("number")}. #{issue.fetch("title")}: #{issue.fetch("status")}"
    end
  end
  File.open(ARGV.fetch(2), "w") do |file|
     file.puts "# Audit fixture"
     file.puts "covered_external reviewed_evidence.bundle"
     file.puts "docs/operations/external-acceptance.env.example"
     file.puts "docs/operations/external-closure-matrix.md"
     file.puts "local-acceptance gofmt parse YAML"
     file.puts "go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12"
     file.puts "make check-github-external-config"
     file.puts "make smoke-github-external-config"
     file.puts "make verify-sdk-registries PyPI onefacture install failed npm @onefacture/sdk install failed"
     file.puts "## Titres source couverts"
    data.fetch("issues").each do |issue|
      file.puts "#{issue.fetch("number")}. #{issue.fetch("title")}"
    end
    file.puts "## Criteres d'\''acceptation source couverts"
    current_issue = nil
    in_acceptance = false
    File.read("docs/backlog/github-issues-vagues.md").each_line do |line|
      if (match = line.match(/^### (\d+)\./))
        current_issue = match[1]
        in_acceptance = false
      elsif line.include?("**Critères d'\''acceptation**")
        in_acceptance = true
      elsif in_acceptance && (match = line.match(/^- (.+)$/))
        file.puts "#{current_issue}. #{match[1]}"
      end
    end
    file.puts "## Descriptions source couvertes"
    current_issue = nil
    in_acceptance = false
    File.read("docs/backlog/github-issues-vagues.md").each_line do |line|
      if (match = line.match(/^### (\d+)\./))
        current_issue = match[1]
        in_acceptance = false
      elsif line.include?("**Critères d'\''acceptation**")
        in_acceptance = true
      elsif current_issue && !in_acceptance && (match = line.match(/^- (.+)$/))
        file.puts "#{current_issue}. #{match[1]}"
      end
    end
    file.puts "| # | Status |"
    file.puts "|---|---|"
    data.fetch("issues").each do |issue|
      file.puts "| #{issue.fetch("number")} | #{issue.fetch("status")} |"
    end
  end
' "$fully_reviewed_manifest" "$reviewed_review" "$reviewed_audit"

fully_reviewed_out="$tmpdir/fully-reviewed-covered-external.out"
MANIFEST_PATH="$fully_reviewed_manifest" REVIEW_PATH="$reviewed_review" AUDIT_PATH="$reviewed_audit" BUNDLE="$valid_bundle" ruby scripts/audit_backlog_completion.rb >"$fully_reviewed_out"
if ! grep -Fq "Completion audit: complete; all manifest issues are covered locally or by reviewed external evidence." "$fully_reviewed_out"; then
	echo "fully-reviewed-covered-external audit did not complete" >&2
	exit 1
fi
if ! grep -Fq "Reviewed external evidence:" "$fully_reviewed_out"; then
	echo "fully-reviewed-covered-external audit did not print reviewed evidence" >&2
	exit 1
fi

invalid_bundle="$tmpdir/invalid-reviewed-evidence"
mkdir -p "$invalid_bundle"
printf "Paste redacted output\n" >"$invalid_bundle/live-pa.log"
invalid_bundle_manifest="$tmpdir/covered-external-invalid-bundle.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  data.fetch("issues").each do |issue|
    next unless issue["reviewed_evidence"]

    issue["reviewed_evidence"]["bundle"] = ARGV.fetch(1)
  end
  File.write(ARGV.fetch(2), JSON.pretty_generate(data))
' "$fully_reviewed_manifest" "$invalid_bundle" "$invalid_bundle_manifest"

expect_audit_failure \
	"covered-external-invalid-bundle" \
	"reviewed evidence bundle failed verification" \
	env MANIFEST_PATH="$invalid_bundle_manifest" REVIEW_PATH="$reviewed_review" AUDIT_PATH="$reviewed_audit" BUNDLE="$valid_bundle" ruby scripts/audit_backlog_completion.rb

wrong_commit_manifest="$tmpdir/covered-external-wrong-commit.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  data.fetch("issues").each do |issue|
    next unless issue["reviewed_evidence"]

    issue["reviewed_evidence"]["commit_sha"] = "0000000000000000000000000000000000000000"
  end
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' "$fully_reviewed_manifest" "$wrong_commit_manifest"

expect_audit_failure \
	"covered-external-wrong-commit" \
	"does not match HEAD" \
	env MANIFEST_PATH="$wrong_commit_manifest" REVIEW_PATH="$reviewed_review" AUDIT_PATH="$reviewed_audit" BUNDLE="$valid_bundle" ruby scripts/audit_backlog_completion.rb

bad_timestamp_manifest="$tmpdir/covered-external-bad-reviewed-at.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  data.fetch("issues").each do |issue|
    next unless issue["reviewed_evidence"]

    issue["reviewed_evidence"]["reviewed_at"] = "2026-05-22 00:00:00"
  end
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' "$fully_reviewed_manifest" "$bad_timestamp_manifest"

expect_audit_failure \
	"covered-external-bad-reviewed-at" \
	"reviewed_at must be an ISO-8601 UTC timestamp" \
	env MANIFEST_PATH="$bad_timestamp_manifest" REVIEW_PATH="$reviewed_review" AUDIT_PATH="$reviewed_audit" BUNDLE="$valid_bundle" ruby scripts/audit_backlog_completion.rb

invalid_timestamp_manifest="$tmpdir/covered-external-invalid-reviewed-at.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  data.fetch("issues").each do |issue|
    next unless issue["reviewed_evidence"]

    issue["reviewed_evidence"]["reviewed_at"] = "2026-99-99T99:99:99Z"
  end
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' "$fully_reviewed_manifest" "$invalid_timestamp_manifest"

expect_audit_failure \
	"covered-external-invalid-reviewed-at" \
	"reviewed_at must be a valid UTC timestamp" \
	env MANIFEST_PATH="$invalid_timestamp_manifest" REVIEW_PATH="$reviewed_review" AUDIT_PATH="$reviewed_audit" BUNDLE="$valid_bundle" ruby scripts/audit_backlog_completion.rb

placeholder_reviewer_manifest="$tmpdir/covered-external-placeholder-reviewer.json"
ruby -rjson -e '
  data = JSON.parse(File.read(ARGV.fetch(0)))
  data.fetch("issues").each do |issue|
    next unless issue["reviewed_evidence"]

    issue["reviewed_evidence"]["reviewed_by"] = "TODO"
  end
  File.write(ARGV.fetch(1), JSON.pretty_generate(data))
' "$fully_reviewed_manifest" "$placeholder_reviewer_manifest"

expect_audit_failure \
	"covered-external-placeholder-reviewer" \
	"reviewed_by must name the actual reviewer" \
	env MANIFEST_PATH="$placeholder_reviewer_manifest" REVIEW_PATH="$reviewed_review" AUDIT_PATH="$reviewed_audit" BUNDLE="$valid_bundle" ruby scripts/audit_backlog_completion.rb

echo "backlog completion audit smoke passed"
