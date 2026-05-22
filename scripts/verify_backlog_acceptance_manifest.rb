#!/usr/bin/env ruby
# frozen_string_literal: true

require "json"
require "pathname"
require "tempfile"
require "time"
require "yaml"

root = Pathname.new(__dir__).join("..").expand_path
backlog_path = root.join("docs/backlog/github-issues-vagues.md")
manifest_path = Pathname.new(ENV.fetch("MANIFEST_PATH", root.join("docs/backlog/github-issues-vagues-acceptance.json").to_s))
manifest_path = root.join(manifest_path) unless manifest_path.absolute?
manifest_fixture = ENV.key?("MANIFEST_PATH")
plan_path = root.join("docs/backlog/github-issues-vagues-plan.md")
review_path = Pathname.new(ENV.fetch("REVIEW_PATH", root.join("docs/backlog/github-issues-vagues-review.md").to_s))
review_path = root.join(review_path) unless review_path.absolute?
audit_path = Pathname.new(ENV.fetch("AUDIT_PATH", root.join("docs/backlog/github-issues-vagues-completion-audit.md").to_s))
audit_path = root.join(audit_path) unless audit_path.absolute?
makefile_path = root.join("Makefile")
external_acceptance_path = root.join("scripts/verify_external_acceptance.sh")
completion_audit_path = root.join("scripts/audit_backlog_completion.rb")
completion_audit_smoke_path = root.join("scripts/smoke_backlog_completion_audit.sh")
external_evidence_collector_path = root.join("scripts/collect_external_acceptance_evidence.sh")
external_evidence_collector_smoke_path = root.join("scripts/smoke_external_evidence_collector.sh")
external_env_checker_path = root.join("scripts/check_external_acceptance_env.sh")
external_env_smoke_path = root.join("scripts/smoke_external_acceptance_env.sh")
external_evidence_review_path = root.join("scripts/review_external_evidence_bundle.rb")
external_evidence_review_smoke_path = root.join("scripts/smoke_external_evidence_review.sh")
local_acceptance_path = root.join("scripts/verify_local_acceptance.sh")
external_smokes_path = root.join("scripts/verify_external_gate_smokes.sh")
external_evidence_verifier_path = root.join("scripts/verify_external_evidence_bundle.sh")
external_evidence_path = root.join("docs/operations/external-acceptance-evidence.md")
external_env_template_path = root.join("docs/operations/external-acceptance.env.example")
external_closure_matrix_path = root.join("docs/operations/external-closure-matrix.md")
external_acceptance_runbook_path = root.join("docs/operations/external-acceptance.md")
external_acceptance_workflow_path = root.join(".github/workflows/external-acceptance.yml")
ci_workflow_path = root.join(".github/workflows/ci.yml")
openapi_spec_path = root.join("internal/gateway/openapi/spec.yaml")
openapi_test_path = root.join("internal/gateway/openapi/openapi_test.go")
postman_collection_path = root.join("docs/onboarding/onefacture.postman_collection.json")
routes_test_path = root.join("internal/gateway/routes/handlers_test.go")
doctor_test_path = root.join("cmd/onefacture/main_test.go")
problem_test_path = root.join("internal/gateway/problem/problem_test.go")
events_bus_path = root.join("internal/events/bus.go")
directory_test_path = root.join("internal/directory/directory_test.go")
webhook_test_path = root.join("internal/webhooks/deliverer_test.go")
reliability_test_path = root.join("internal/reliability/adapter_test.go")
middleware_test_path = root.join("internal/gateway/middleware/middleware_test.go")
jurisdiction_test_path = root.join("internal/jurisdiction/registry_test.go")
security_encryption_test_path = root.join("internal/security/encryption_test.go")
security_http_kms_test_path = root.join("internal/security/http_kms_test.go")
storage_test_path = root.join("internal/storage/storage_test.go")
chorus_test_path = root.join("internal/adapters/chorus/chorus_test.go")
docaposte_test_path = root.join("internal/adapters/docaposte/docaposte_test.go")
pennylane_test_path = root.join("internal/adapters/pennylane/pennylane_test.go")
sandbox_client_test_path = root.join("internal/adapters/sandbox/client_test.go")
sdk_release_verifier_path = root.join("scripts/verify_sdk_release_artifacts.sh")
python_sdk_pyproject_path = root.join("sdks/python/pyproject.toml")
typescript_sdk_package_path = root.join("sdks/typescript/package.json")
sdk_publish_workflow_path = root.join(".github/workflows/sdk-publish.yml")

backlog_text = backlog_path.read
issues = backlog_text.scan(/^### (\d+)\. (.+)$/).map { |number, title| [number.to_i, title.strip] }.to_h
source_metadata = {}
source_descriptions = Hash.new { |hash, key| hash[key] = [] }
source_acceptance = Hash.new { |hash, key| hash[key] = [] }
current_wave = nil
current_issue = nil
in_acceptance = false
backlog_text.each_line do |line|
  if (match = line.match(/^## Vague (\d+)/))
    current_wave = match[1].to_i
    in_acceptance = false
  elsif (match = line.match(/^### (\d+)\./))
    current_issue = match[1].to_i
    source_metadata[current_issue] = { "wave" => current_wave, "labels" => [] }
    in_acceptance = false
  elsif current_issue && (match = line.match(/^\*\*Labels\*\*: (.+)$/))
    source_metadata.fetch(current_issue)["labels"] = match[1].scan(/`([^`]+)`/).flatten
  elsif current_issue && line.include?("**Critères d'acceptation**")
    in_acceptance = true
  elsif current_issue && in_acceptance && (match = line.match(/^- (.+)$/))
    source_acceptance[current_issue] << match[1].strip
  elsif current_issue && !in_acceptance && (match = line.match(/^- (.+)$/))
    source_descriptions[current_issue] << match[1].strip
  end
end
manifest = JSON.parse(manifest_path.read)
entries = manifest.fetch("issues")
make_targets = makefile_path.read.scan(/^([a-zA-Z0-9_-]+):(?:\s|$)/).flatten.to_h { |target| [target, true] }
external_modes = external_acceptance_path.read.scan(/^\s*([a-z0-9-]+)\)\s*$/).flatten.to_h { |mode| [mode, true] }
external_required_env = external_acceptance_path.read.scan(/require_env ([A-Z0-9_]+)/).flatten.uniq
abort "completion audit script missing" unless completion_audit_path.exist?
abort "completion audit smoke script missing" unless completion_audit_smoke_path.exist?
abort "external evidence collector missing" unless external_evidence_collector_path.exist?
abort "external evidence collector smoke missing" unless external_evidence_collector_smoke_path.exist?
abort "external env checker missing" unless external_env_checker_path.exist?
abort "external env smoke missing" unless external_env_smoke_path.exist?
abort "external evidence review helper missing" unless external_evidence_review_path.exist?
abort "external evidence review smoke missing" unless external_evidence_review_smoke_path.exist?
abort "local acceptance wrapper missing" unless local_acceptance_path.exist?
completion_audit = completion_audit_path.read
completion_audit_smoke = completion_audit_smoke_path.read
external_evidence_collector = external_evidence_collector_path.read
external_evidence_collector_smoke = external_evidence_collector_smoke_path.read
external_env_checker = external_env_checker_path.read
external_env_smoke = external_env_smoke_path.read
external_evidence_review = external_evidence_review_path.read
external_evidence_review_smoke = external_evidence_review_smoke_path.read
local_acceptance = local_acceptance_path.read
external_smokes = external_smokes_path.read
abort "external evidence bundle verifier missing" unless external_evidence_verifier_path.exist?
external_evidence_verifier = external_evidence_verifier_path.read
external_evidence = external_evidence_path.read
abort "external acceptance env template missing" unless external_env_template_path.exist?
external_env_template = external_env_template_path.read
abort "external closure matrix missing" unless external_closure_matrix_path.exist?
external_closure_matrix = external_closure_matrix_path.read
external_acceptance_runbook = external_acceptance_runbook_path.read
openapi_spec = openapi_spec_path.read
openapi_test = openapi_test_path.read
routes_test = routes_test_path.read
doctor_test = doctor_test_path.read
problem_test = problem_test_path.read
events_bus = events_bus_path.read
directory_test = directory_test_path.read
webhook_test = webhook_test_path.read
reliability_test = reliability_test_path.read
middleware_test = middleware_test_path.read
jurisdiction_test = jurisdiction_test_path.read
security_encryption_test = security_encryption_test_path.read
security_http_kms_test = security_http_kms_test_path.read
storage_test = storage_test_path.read
chorus_test = chorus_test_path.read
docaposte_test = docaposte_test_path.read
pennylane_test = pennylane_test_path.read
sandbox_client_test = sandbox_client_test_path.read
sdk_release_verifier = sdk_release_verifier_path.read
python_sdk_pyproject = python_sdk_pyproject_path.read
typescript_sdk_package = JSON.parse(typescript_sdk_package_path.read)
sdk_publish_workflow = sdk_publish_workflow_path.read
postman_collection = JSON.parse(postman_collection_path.read)
python = %w[python3 python].find { |cmd| system(cmd, "--version", out: File::NULL, err: File::NULL) }
abort "python is required to validate embedded external-acceptance snippets" unless python
workflow = YAML.load_file(external_acceptance_workflow_path)
workflow_text = external_acceptance_workflow_path.read
workflow_trigger = workflow["on"] || workflow[true]
workflow_gate_options = workflow_trigger.fetch("workflow_dispatch").fetch("inputs").fetch("gate").fetch("options")
ci_workflow = YAML.load_file(ci_workflow_path)
ci_jobs = ci_workflow.fetch("jobs")
plan_text = plan_path.read
plan_rows = plan_text.scan(/^\|\s*(\d+)\s*\|.*?\|\s*(.*?)\s*\|$/).to_h do |number, gate_cell|
  gates = gate_cell.scan(/`(make [^`]+)`/).flatten
  [number.to_i, gates]
end
plan_metadata_rows = plan_text.scan(/^\|\s*(\d+)\s*\|\s*(\d+)\s*\|\s*([^|]+?)\s*\|$/).to_h do |number, wave, labels|
  [number.to_i, { "wave" => wave.to_i, "labels" => labels.strip.split(/\s*,\s*/) }]
end
review_issues = review_path.read.scan(/^(\d+)\.\s/).flatten.map(&:to_i).uniq
review_text = review_path.read
audit_issues = audit_path.read.scan(/^\|\s*(\d+)\s*\|/).flatten.map(&:to_i).uniq
audit_text = audit_path.read
expected_issue_numbers = (1..24).to_a

abort "expected 24 backlog issues, got #{issues.length}" unless issues.length == 24
abort "expected 24 source metadata rows, got #{source_metadata.length}" unless source_metadata.length == 24
abort "expected 24 source description rows, got #{source_descriptions.length}" unless source_descriptions.length == 24
abort "expected 24 source acceptance rows, got #{source_acceptance.length}" unless source_acceptance.length == 24
abort "expected 24 plan metadata rows, got #{plan_metadata_rows.length}" unless plan_metadata_rows.length == 24
abort "expected 24 manifest issues, got #{entries.length}" unless entries.length == 24
abort "expected 24 plan issues, got #{plan_rows.length}" unless plan_rows.length == 24
abort "backlog issue numbers must be exactly 1..24" unless issues.keys.sort == expected_issue_numbers
abort "source metadata issue numbers must be exactly 1..24" unless source_metadata.keys.sort == expected_issue_numbers
abort "source description issue numbers must be exactly 1..24" unless source_descriptions.keys.sort == expected_issue_numbers
abort "source acceptance issue numbers must be exactly 1..24" unless source_acceptance.keys.sort == expected_issue_numbers
abort "plan metadata issue numbers must be exactly 1..24" unless plan_metadata_rows.keys.sort == expected_issue_numbers
abort "manifest issue numbers must be exactly 1..24" unless entries.map { |entry| entry.fetch("number") }.sort == expected_issue_numbers
abort "plan issue numbers must be exactly 1..24" unless plan_rows.keys.sort == expected_issue_numbers
abort "external workflow gate options do not match script modes" unless workflow_gate_options.sort == external_modes.keys.sort
abort "external workflow missing env readiness check" unless workflow_text.include?("check_external_acceptance_env.sh")
abort "external workflow env readiness check is not gate-aware" unless workflow_text.include?("check_external_acceptance_env.sh \"${{ inputs.gate }}\"")
abort "external workflow single-gate path missing" unless workflow_text.include?("if: inputs.gate != 'all'")
abort "external workflow single-gate command missing" unless workflow_text.include?("verify_external_acceptance.sh \"${{ inputs.gate }}\"")
abort "external workflow all-gate collector guard missing" unless workflow_text.include?("if: inputs.gate == 'all'")
abort "external workflow missing evidence collector" unless workflow_text.include?("collect_external_acceptance_evidence.sh")
abort "external workflow missing evidence artifact upload" unless workflow_text.include?("actions/upload-artifact@v4")
abort "external workflow evidence artifact upload must run on failure" unless workflow_text.include?("if: always() && inputs.gate == 'all'")
abort "external workflow evidence artifact path missing" unless workflow_text.include?("docs/operations/evidence/*-github-${{ github.run_id }}-external-acceptance")
abort "external workflow evidence links must use github.repository" unless workflow_text.include?("ONEFACTURE_EVIDENCE_LINKS: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}")
abort "external workflow missing evidence operator" unless workflow_text.include?("ONEFACTURE_EVIDENCE_OPERATOR:")
abort "external workflow missing evidence environment" unless workflow_text.include?("ONEFACTURE_EVIDENCE_ENVIRONMENT:")
external_modes.keys.each do |mode|
  abort "external evidence checklist missing gate #{mode}" unless external_evidence.include?("`#{mode}`")
end
abort "external evidence checklist missing verifier command" unless external_evidence.include?("make verify-external-evidence")
abort "external acceptance runbook missing env template link" unless external_acceptance_runbook.include?("docs/operations/external-acceptance.env.example")
abort "external acceptance runbook missing closure matrix link" unless external_acceptance_runbook.include?("docs/operations/external-closure-matrix.md")
abort "external evidence checklist missing env template link" unless external_evidence.include?("docs/operations/external-acceptance.env.example")
abort "completion audit doc missing env template link" unless audit_text.include?("docs/operations/external-acceptance.env.example")
abort "completion audit doc missing closure matrix link" unless audit_text.include?("docs/operations/external-closure-matrix.md")
abort "external evidence checklist missing verifier smoke command" unless external_evidence.include?("make verify-external-evidence-smoke")
abort "external evidence checklist missing scaffold command" unless external_evidence.include?("make create-external-evidence")
abort "external evidence checklist missing collector command" unless external_evidence.include?("make collect-external-evidence")
abort "external evidence checklist missing review helper marker note" unless external_evidence.include?("review helper prints the manifest template plus the exact per-issue review markers")
%w[check-external-env collect-external-evidence verify-external-evidence review-external-evidence audit-backlog-completion].each do |target|
  abort "external acceptance runbook missing make #{target}" unless external_acceptance_runbook.include?("make #{target}")
end
%w[ONEFACTURE_EVIDENCE_LINKS ONEFACTURE_EVIDENCE_OPERATOR ONEFACTURE_EVIDENCE_ENVIRONMENT].each do |var|
  abort "external acceptance runbook missing #{var}" unless external_acceptance_runbook.include?(var)
end
abort "external acceptance runbook missing retry baseline requirement" unless external_acceptance_runbook.include?("ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE") && external_acceptance_runbook.include?("amelioration de l'item 21")
abort "external acceptance runbook missing SDK publication handoff" unless external_acceptance_runbook.include?("## Publication SDKs")
abort "external acceptance runbook missing SDK local verifier before publish" unless external_acceptance_runbook.include?("make verify-sdk")
abort "external acceptance runbook missing PyPI trusted publishing handoff" unless external_acceptance_runbook.include?("PyPI trusted publishing") && external_acceptance_runbook.include?("onefacture")
abort "external acceptance runbook missing npm token handoff" unless external_acceptance_runbook.include?("NPM_TOKEN") && external_acceptance_runbook.include?("@onefacture/sdk")
abort "external acceptance runbook missing sdk-publish workflow dispatch" unless external_acceptance_runbook.include?(".github/workflows/sdk-publish.yml") && external_acceptance_runbook.include?("publish_python=true") && external_acceptance_runbook.include?("publish_typescript=true")
abort "external evidence checklist missing SDK publication evidence links" unless external_evidence.include?("public PyPI `onefacture` package page") && external_evidence.include?("public npm `@onefacture/sdk` package page")
abort "external evidence checklist must reject local-only SDK installs" unless external_evidence.include?("not accepted if it only installs local tarballs or local source directories")
abort "external evidence checklist missing retry baseline comparison" unless external_evidence.include?("success_rate > ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE")
abort "external acceptance runbook missing current HEAD requirement" unless external_acceptance_runbook.include?("correspondant au `HEAD`")
abort "external evidence checklist missing covered_external handoff" unless external_evidence.include?("covered_external") && external_evidence.include?("reviewed_evidence")
abort "external evidence checklist missing covered_external audit verification note" unless external_evidence.include?("re-runs the external evidence bundle verifier")
abort "external evidence checklist missing per-issue covered_external marker format" unless external_evidence.include?("<number>. <title>: covered_external") && external_evidence.include?("| <number> | covered_external |")
abort "external evidence checklist missing persisted bundle path rule" unless external_evidence.include?("reviewed_evidence.bundle") && external_evidence.include?("docs/operations/evidence/")
abort "external evidence checklist missing persisted bundle existence rule" unless external_evidence.include?("existing bundle directory")
abort "external evidence checklist missing reviewed commit SHA format rule" unless external_evidence.include?("full lowercase `commit_sha`")
abort "external evidence checklist missing reviewed_at timestamp format rule" unless external_evidence.include?("YYYY-MM-DDTHH:MM:SSZ")
abort "external evidence checklist missing valid UTC timestamp wording" unless external_evidence.include?("valid UTC timestamp")
abort "external evidence checklist missing non-placeholder URL wording" unless external_evidence.include?("non-placeholder evidence URL") && external_evidence.include?("127.0.0.1")
abort "external evidence checklist missing reviewed_by placeholder rule" unless external_evidence.include?("non-placeholder `reviewed_by`")
abort "external acceptance runbook missing covered_external handoff" unless external_acceptance_runbook.include?("covered_external") && external_acceptance_runbook.include?("reviewed_evidence.bundle")
abort "Makefile missing audit-backlog-completion" unless make_targets["audit-backlog-completion"]
abort "Makefile missing smoke-backlog-completion-audit" unless make_targets["smoke-backlog-completion-audit"]
abort "Makefile missing collect-external-evidence" unless make_targets["collect-external-evidence"]
abort "Makefile missing smoke-external-evidence-collector" unless make_targets["smoke-external-evidence-collector"]
abort "Makefile missing check-external-env" unless make_targets["check-external-env"]
abort "Makefile missing smoke-external-env" unless make_targets["smoke-external-env"]
abort "Makefile missing review-external-evidence" unless make_targets["review-external-evidence"]
abort "Makefile missing smoke-external-evidence-review" unless make_targets["smoke-external-evidence-review"]
abort "Makefile check-external-env missing GATE support" unless makefile_path.read.include?("GATE ?= all")
abort "external evidence collector missing env preflight" unless external_evidence_collector.include?("check_external_acceptance_env.sh")
abort "external evidence collector missing operator identity preflight" unless external_evidence_collector.include?("ONEFACTURE_EVIDENCE_OPERATOR must name the actual reviewer or automation identity")
abort "external evidence collector missing environment identity preflight" unless external_evidence_collector.include?("ONEFACTURE_EVIDENCE_ENVIRONMENT must name the external acceptance target")
abort "external evidence collector missing verifier call" unless external_evidence_collector.include?("verify_external_evidence_bundle.sh")
abort "external evidence collector missing redaction" unless external_evidence_collector.include?("redact_stream")
abort "external evidence collector smoke missing fake make harness" unless external_evidence_collector_smoke.include?("unexpected fake make target")
abort "external evidence collector smoke missing bundle verification" unless external_evidence_collector_smoke.include?("verify_external_evidence_bundle.sh")
abort "external evidence collector smoke missing redaction fixture" unless external_evidence_collector_smoke.include?("Bearer abcdefghijklmnopqrstuvwxyz123456")
abort "external evidence collector smoke missing redaction assertion" unless external_evidence_collector_smoke.include?("collector failed to redact secret-like output")
abort "external evidence collector smoke missing failed-gate fixture" unless external_evidence_collector_smoke.include?("ONEFACTURE_FAKE_FAIL_PUBLIC_SANDBOX")
abort "external evidence collector smoke missing failed summary assertion" unless external_evidence_collector_smoke.include?("make verify-public-sandbox: FAIL")
abort "external evidence collector smoke missing bad-links preflight fixture" unless external_evidence_collector_smoke.include?("expected collector preflight to reject evidence links without URL")
abort "external evidence collector smoke missing no-bundle assertion for bad links" unless external_evidence_collector_smoke.include?("collector created evidence bundle despite bad links preflight")
abort "external evidence collector smoke missing bad-operator preflight fixture" unless external_evidence_collector_smoke.include?("expected collector preflight to reject unknown operator")
abort "external evidence collector smoke missing no-bundle assertion for bad operator" unless external_evidence_collector_smoke.include?("collector created evidence bundle despite bad operator preflight")
abort "external evidence collector smoke missing bad-environment preflight fixture" unless external_evidence_collector_smoke.include?("expected collector preflight to reject unknown environment")
abort "external evidence collector smoke missing no-bundle assertion for bad environment" unless external_evidence_collector_smoke.include?("collector created evidence bundle despite bad environment preflight")
abort "external env checker missing Chorus preflight" unless external_env_checker.include?("ONEFACTURE_CHORUS_BASE_URL")
abort "external env checker missing production API preflight" unless external_env_checker.include?("ONEFACTURE_PROD_API_KEY")
abort "external env checker missing gate mode switch" unless external_env_checker.include?("public-sandbox)")
abort "external env checker missing evidence links URL preflight" unless external_env_checker.include?("ONEFACTURE_EVIDENCE_LINKS") && external_env_checker.include?("must include at least one evidence URL")
abort "external env smoke missing evidence links URL fixture" unless external_env_smoke.include?("expected external env check to reject evidence links without URL")
abort "external env checker missing evidence links placeholder preflight" unless external_env_checker.include?("must not use placeholder or localhost URLs")
abort "external env smoke missing evidence links placeholder fixture" unless external_env_smoke.include?("expected external env check to reject placeholder evidence URL")
abort "external env checker missing evidence operator preflight" unless external_env_checker.include?("ONEFACTURE_EVIDENCE_OPERATOR") && external_env_checker.include?("must name the actual reviewer or automation identity")
abort "external env checker must require evidence operator for all gates" unless external_env_checker.include?("ONEFACTURE_EVIDENCE_OPERATOR") && external_env_checker.include?('operator="${ONEFACTURE_EVIDENCE_OPERATOR:-unknown}"')
abort "external env smoke missing evidence operator fixture" unless external_env_smoke.include?("expected external env check to reject unknown evidence operator")
abort "external env checker missing evidence environment preflight" unless external_env_checker.include?("ONEFACTURE_EVIDENCE_ENVIRONMENT") && external_env_checker.include?("must name the external acceptance target")
abort "external env checker must require evidence environment for all gates" unless external_env_checker.include?("ONEFACTURE_EVIDENCE_ENVIRONMENT") && external_env_checker.include?('environment="${ONEFACTURE_EVIDENCE_ENVIRONMENT:-unknown}"')
abort "external env smoke missing evidence environment fixture" unless external_env_smoke.include?("expected external env check to reject unknown evidence environment")
abort "external env smoke missing explicit evidence metadata success fixture" unless external_env_smoke.include?('ONEFACTURE_EVIDENCE_OPERATOR="env-smoke"') && external_env_smoke.include?('ONEFACTURE_EVIDENCE_ENVIRONMENT="external-env-smoke"')
abort "external evidence collector must not default operator from local user" unless external_evidence_collector.include?('operator="${ONEFACTURE_EVIDENCE_OPERATOR:-unknown}"')
abort "external evidence collector must not default environment to redacted-live-targets" unless external_evidence_collector.include?('environment="${ONEFACTURE_EVIDENCE_ENVIRONMENT:-unknown}"')
external_required_env.each do |name|
  abort "external env checker missing required env from acceptance script: #{name}" unless external_env_checker.include?(name)
  abort "external env template missing required env #{name}" unless external_env_template.include?(name)
end
%w[ONEFACTURE_EVIDENCE_OPERATOR ONEFACTURE_EVIDENCE_ENVIRONMENT ONEFACTURE_MIN_RETRIED_INVOICES].each do |name|
  abort "external env template missing optional env #{name}" unless external_env_template.include?(name)
end
abort "external env template must not contain real localhost evidence URL" if external_env_template.include?("localhost") || external_env_template.include?("127.0.0.1")
abort "external env template must use invalid/example placeholders only" unless external_env_template.include?("example.invalid")
%w[
  scripts/verify_external_gate_smokes.sh
  scripts/smoke_external_evidence_bundle.sh
  scripts/smoke_backlog_completion_audit.sh
  scripts/smoke_external_evidence_collector.sh
  scripts/smoke_external_acceptance_env.sh
  scripts/smoke_external_evidence_review.sh
].each do |script|
  abort "local acceptance wrapper missing #{script}" unless local_acceptance.include?(script)
end
abort "local acceptance wrapper missing manifest verifier" unless local_acceptance.include?("ruby scripts/verify_backlog_acceptance_manifest.rb")
abort "local acceptance wrapper missing diff hygiene check" unless local_acceptance.include?("git diff --check")
abort "local acceptance wrapper missing gofmt check" unless local_acceptance.include?("gofmt -l")
abort "local acceptance wrapper missing all shell syntax check" unless local_acceptance.include?("find scripts -name '*.sh'") && local_acceptance.include?("bash -n")
abort "local acceptance wrapper missing all ruby syntax check" unless local_acceptance.include?("find scripts -name '*.rb'") && local_acceptance.include?("ruby -c")
abort "local acceptance wrapper missing external acceptance workflow YAML parse" unless local_acceptance.include?(".github/workflows/external-acceptance.yml")
abort "local acceptance wrapper missing short storage unit tests" unless local_acceptance.include?("go test -short ./internal/storage")
%w[
  ./internal/adapters/mock
  ./internal/config
  ./internal/core/facturx
  ./internal/core/invoice
  ./internal/events
  ./internal/gateway
  ./internal/validation
  ./internal/workers
].each do |pkg|
  abort "local acceptance wrapper missing runnable test package #{pkg}" unless local_acceptance.include?(pkg)
end
%w[commercial_invoice credit_note correction_invoice /v1/invoices/{id}/retry].each do |marker|
  abort "OpenAPI spec missing interactive example marker #{marker}" unless openapi_spec.include?(marker)
  abort "OpenAPI test missing interactive example marker #{marker}" unless openapi_test.include?(marker)
end
%w[TestValidateBulkReturnsAggregateReport TestValidateBulkExportsCSVErrors].each do |test_name|
  abort "routes test missing #{test_name}" unless routes_test.include?(test_name)
end
abort "compliance dashboard test missing monthly trends assertion" unless routes_test.include?("monthly_trends") && routes_test.include?("Tendances mensuelles")
abort "webhook inspector test missing one-click replay assertion" unless routes_test.include?("Replay</button>") && routes_test.include?("/v1/webhooks/deliveries/") && routes_test.include?("/replay")
abort "routes test missing Idempotency-Key required assertion" unless routes_test.include?("TestIdempotencyKeyIsRequired")
abort "routes test missing PA routing override coverage" unless routes_test.include?("TestResolvePAIDUsesBuyerOverride")
abort "routes test missing invoice timeline retry/latency coverage" unless routes_test.include?("TestBuildTimelineIncludesLatencyAndRejectionRetry")
abort "routes test missing rejection patch suggestion coverage" unless routes_test.include?("TestSuggestRejectionPatchForSIREN") && routes_test.include?("outcome_metric")
abort "routes test missing rejection retry metric coverage" unless routes_test.include?("TestBuildRejectionRetrySuccessRate")
abort "doctor CLI test missing terminal report assertion" unless doctor_test.include?("TestFormatDoctorReportShowsClearTerminalStatus")
abort "problem tests missing top error enrichment coverage" unless problem_test.include?("TestTopErrorHelpersHaveActionableEnrichment")
%w[XAdd XGroupCreateMkStream XReadGroup XAck].each do |marker|
  abort "events bus missing Redis Streams operation #{marker}" unless events_bus.include?(marker)
end
abort "directory tests missing cached lookup P95 gate" unless directory_test.include?("TestResolverCachedLookupP95Under100ms") && directory_test.include?("100*time.Millisecond")
abort "webhook tests missing mTLS handshake coverage" unless webhook_test.include?("TestClientForEndpointPerformsMTLSHandshake") && webhook_test.include?("tls.RequireAnyClientCert")
%w[TestAdapterRetriesSubmitUntilSuccess TestAdapterOpensCircuitAfterFailures].each do |test_name|
  abort "reliability tests missing #{test_name}" unless reliability_test.include?(test_name)
end
abort "middleware tests missing request id access log coverage" unless middleware_test.include?("TestAccessLogIncludesRequestID") && middleware_test.include?("request_id=")
abort "jurisdiction tests missing add-profile-without-core-change coverage" unless jurisdiction_test.include?("TestRegistryCanAddJurisdictionWithoutCoreAPIChange")
abort "security tests missing encryption rotation coverage" unless security_encryption_test.include?("TestEncryptorDecryptsOldEnvelopeAfterRotation")
abort "security tests missing HTTP KMS rotation coverage" unless security_http_kms_test.include?("TestHTTPKMSProviderRoundTripAndRotation")
abort "storage tests missing encrypted artifact metadata coverage" unless storage_test.include?("TestInvoiceRepoEncryptsAndDecryptsArtifacts") && storage_test.include?("InspectEncryptedArtifact")
abort "chorus tests missing OAuth2 sandbox config coverage" unless chorus_test.include?("ClientID") && chorus_test.include?("ClientSecret") && chorus_test.include?("TokenURL")
abort "docaposte tests missing sandbox env config coverage" unless docaposte_test.include?("TestNewConfiguresSandboxClientFromEnv") && docaposte_test.include?("ONEFACTURE_DOCAPOSTE_API_TOKEN")
abort "pennylane tests missing sandbox env config coverage" unless pennylane_test.include?("TestNewConfiguresSandboxClientFromEnv") && pennylane_test.include?("ONEFACTURE_PENNYLANE_API_TOKEN")
%w[TestClientSubmitAndStatusRoundTrip TestClientUsesOAuthClientCredentials TestClientMapsPAErrorResponse TestClientWebhookDecode].each do |test_name|
  abort "sandbox adapter client tests missing #{test_name}" unless sandbox_client_test.include?(test_name)
end
abort "Python SDK pyproject must publish package onefacture" unless python_sdk_pyproject.include?('name = "onefacture"')
abort "Python SDK pyproject missing release metadata" unless %w[version description readme requires-python dependencies].all? { |field| python_sdk_pyproject.include?(field) }
abort "Python SDK verifier missing local pip install smoke" unless sdk_release_verifier.include?("pip install ./sdks/python")
abort "Python SDK verifier missing import smoke" unless sdk_release_verifier.include?("from onefacture import Client")
abort "TypeScript SDK package must publish @onefacture/sdk" unless typescript_sdk_package.fetch("name") == "@onefacture/sdk"
abort "TypeScript SDK package missing dist entrypoints" unless typescript_sdk_package.fetch("main") == "dist/index.js" && typescript_sdk_package.fetch("types") == "dist/index.d.ts" && typescript_sdk_package.fetch("files").include?("dist")
abort "TypeScript SDK verifier missing npm pack smoke" unless sdk_release_verifier.include?("npm pack --json")
abort "TypeScript SDK verifier missing tarball install smoke" unless sdk_release_verifier.include?('npm install "$ROOT/sdks/typescript/$package_file"')
abort "TypeScript SDK verifier missing ESM import smoke" unless sdk_release_verifier.include?('import { OnefactureClient } from "@onefacture/sdk";')
abort "SDK publish workflow missing PyPI trusted publishing action" unless sdk_publish_workflow.include?("pypa/gh-action-pypi-publish@release/v1")
abort "SDK publish workflow missing npm public publish" unless sdk_publish_workflow.include?("npm publish --access public")
postman_text = JSON.generate(postman_collection)
%w[/v1/sandbox/credentials /v1/invoices?submit=true /v1/webhooks /v1/webhooks/deliveries].each do |marker|
  abort "Postman onboarding collection missing #{marker}" unless postman_text.include?(marker)
end
abort "external env smoke missing success marker" unless external_env_smoke.include?("external acceptance environment ok")
abort "external env smoke missing single-gate coverage" unless external_env_smoke.include?("public-sandbox env check")
abort "external evidence review missing bundle verifier call" unless external_evidence_review.include?("verify_external_evidence_bundle.sh")
abort "external evidence review missing issue checklist output" unless external_evidence_review.include?("External evidence review checklist")
abort "external evidence review missing covered_external template" unless external_evidence_review.include?("covered_external")
abort "external evidence review missing reviewed_evidence template" unless external_evidence_review.include?("reviewed_evidence")
abort "external evidence review missing per-issue review markers" unless external_evidence_review.include?("Review document markers after human review")
abort "external evidence review missing per-issue audit markers" unless external_evidence_review.include?("Completion audit status rows after human review")
abort "external evidence review missing final audit gate" unless external_evidence_review.include?("make audit-backlog-completion")
abort "external evidence review smoke missing issue mapping assertion" unless external_evidence_review_smoke.include?("#01 Intégration Chorus Pro PISTE sandbox")
abort "external evidence review smoke missing per-issue review marker assertion" unless external_evidence_review_smoke.include?("review output missing per-issue review marker")
abort "external evidence review smoke missing per-issue audit marker assertion" unless external_evidence_review_smoke.include?("review output missing per-issue audit marker")
abort "external evidence review missing evidence links output" unless external_evidence_review.include?("Evidence links:")
abort "external evidence review smoke missing evidence links assertion" unless external_evidence_review_smoke.include?("review output missing evidence links")
abort "external evidence review missing evidence context output" unless external_evidence_review.include?("Evidence branch:") && external_evidence_review.include?("Evidence environment:") && external_evidence_review.include?("Evidence reruns:")
abort "external evidence review smoke missing evidence context assertions" unless external_evidence_review_smoke.include?("review output missing evidence branch") && external_evidence_review_smoke.include?("review output missing evidence environment") && external_evidence_review_smoke.include?("review output missing evidence reruns")
abort "external evidence review smoke missing invalid bundle rejection" unless external_evidence_review_smoke.include?("expected review helper to reject invalid evidence bundle")
abort "external evidence review smoke missing covered_external template assertion" unless external_evidence_review_smoke.include?("review output missing covered_external template")
abort "external evidence review smoke missing reviewed_at propagation assertion" unless external_evidence_review_smoke.include?("review output missing summary timestamp")
abort "external evidence review smoke missing reviewed_by propagation assertion" unless external_evidence_review_smoke.include?("review output missing summary operator")
abort "external evidence review smoke missing final audit gate assertion" unless external_evidence_review_smoke.include?("review output missing final audit gate")
abort "external evidence verifier missing current HEAD check" unless external_evidence_verifier.include?("git rev-parse HEAD")
abort "external evidence verifier missing timestamp format check" unless external_evidence_verifier.include?("summary timestamp must be ISO-8601 UTC")
abort "external evidence verifier missing timestamp validity check" unless external_evidence_verifier.include?("summary timestamp is not a valid UTC instant")
abort "external evidence verifier missing reruns/links summary fields" unless external_evidence_verifier.include?('"Reruns"') && external_evidence_verifier.include?('"Links"')
abort "external evidence verifier missing summary placeholder check" unless external_evidence_verifier.include?("summary field still contains placeholder text")
abort "external evidence verifier missing operator identity check" unless external_evidence_verifier.include?("summary operator must name the actual reviewer or automation identity")
abort "external evidence verifier missing environment identity check" unless external_evidence_verifier.include?("summary environment must name the external acceptance target")
abort "external evidence verifier missing evidence URL check" unless external_evidence_verifier.include?("summary links must include at least one evidence URL")
abort "external evidence verifier missing placeholder URL check" unless external_evidence_verifier.include?("summary links must not use placeholder or localhost URLs")
abort "external evidence bundle smoke missing mismatched commit rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject mismatched commit")
abort "external evidence bundle smoke missing non-PASS summary rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject non-PASS summary command")
abort "external evidence bundle smoke missing malformed timestamp rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject malformed summary timestamp")
abort "external evidence bundle smoke missing invalid timestamp rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject invalid summary timestamp")
abort "external evidence bundle smoke missing reruns summary rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject missing reruns summary field")
abort "external evidence bundle smoke missing links summary rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject empty links summary field")
abort "external evidence bundle smoke missing links URL rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject summary links without URL")
abort "external evidence bundle smoke missing placeholder URL rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject placeholder evidence URL")
abort "external evidence bundle smoke missing placeholder summary rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject placeholder reruns summary field")
abort "external evidence bundle smoke missing unknown operator rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject unknown operator summary field")
abort "external evidence bundle smoke missing unknown environment rejection" unless root.join("scripts/smoke_external_evidence_bundle.sh").read.include?("expected evidence verifier to reject unknown environment summary field")
abort "external gate smoke missing invalid gate rejection" unless root.join("scripts/verify_external_gate_smokes.sh").read.include?("expected verify_external_acceptance.sh to reject invalid gate")
abort "external SDK registry gate must report PyPI failure independently" unless external_acceptance_path.read.include?("PyPI onefacture install failed")
abort "external SDK registry gate must report npm failure independently" unless external_acceptance_path.read.include?("npm @onefacture/sdk install failed")
abort "external gate smoke missing SDK registry dual-failure assertion" unless root.join("scripts/verify_external_gate_smokes.sh").read.include?("sdk-registries smoke did not report PyPI failure") && root.join("scripts/verify_external_gate_smokes.sh").read.include?("sdk-registries smoke did not report npm failure")
abort "external evidence guide missing current HEAD summary requirement" unless external_evidence.include?("commit SHA must match the repository `HEAD`")
abort "completion audit missing BUNDLE evidence path" unless completion_audit.include?("BUNDLE")
abort "completion audit missing manifest verifier call" unless completion_audit.include?("verify_backlog_acceptance_manifest.rb")
abort "completion audit missing external evidence verifier call" unless completion_audit.include?("verify_external_evidence_bundle.sh")
abort "completion audit missing objective output" unless completion_audit.include?("Objective: planifier, implementer et reviewer chaque issue de docs/backlog/github-issues-vagues.md.")
abort "completion audit missing prompt-to-artifact checklist output" unless completion_audit.include?("Prompt-to-artifact checklist")
abort "completion audit missing local verification gate output" unless completion_audit.include?("Local verification gate: make verify-local")
abort "completion audit missing source acceptance checklist output" unless completion_audit.include?("Source acceptance checklist:")
abort "completion audit missing source description checklist output" unless completion_audit.include?("Source description checklist:")
abort "completion audit missing backlog acceptance parser" unless completion_audit.include?("**Critères d'acceptation**")
abort "completion audit missing issue title output" unless completion_audit.include?("issue.fetch(\"title\")")
abort "completion audit missing external blocker checklist output" unless completion_audit.include?("External blocker checklist")
abort "completion audit missing external evidence next steps output" unless completion_audit.include?("External evidence next steps") && completion_audit.include?("make collect-external-evidence STAMP=YYYY-MM-DD")
abort "completion audit missing reviewed external evidence output" unless completion_audit.include?("Reviewed external evidence")
abort "completion audit missing covered_external completion support" unless completion_audit.include?("covered_external")
abort "completion audit missing MANIFEST_PATH fixture support" unless completion_audit.include?("MANIFEST_PATH")
abort "completion audit missing reviewed evidence verifier call" unless completion_audit.include?("reviewed evidence bundle failed verification")
abort "completion audit missing reviewed evidence HEAD check" unless completion_audit.include?("reviewed evidence commit")
abort "completion audit missing verified evidence review map" unless completion_audit.include?("Verified external evidence is ready for review")
abort "completion audit missing valid-bundle review command" unless completion_audit.include?("Review command: make review-external-evidence BUNDLE=")
abort "completion audit missing reviewed bundle path output" unless completion_audit.include?("Verified external evidence is ready for review: ") && completion_audit.include?("bundle_path")
abort "completion audit smoke missing no-bundle failure" unless completion_audit_smoke.include?("BUNDLE not supplied")
abort "completion audit smoke missing objective assertion" unless completion_audit_smoke.include?("objective-line")
abort "completion audit smoke missing prompt-to-artifact assertion" unless completion_audit_smoke.include?("prompt-artifact-checklist")
abort "completion audit smoke missing local verification gate assertion" unless completion_audit_smoke.include?("local-verification-gate")
abort "completion audit smoke missing source acceptance checklist assertion" unless completion_audit_smoke.include?("source-acceptance-checklist")
abort "completion audit smoke missing source acceptance criterion assertion" unless completion_audit_smoke.include?("source-acceptance-criterion")
abort "completion audit smoke missing source description checklist assertion" unless completion_audit_smoke.include?("source-description-checklist")
abort "completion audit smoke missing source description bullet assertion" unless completion_audit_smoke.include?("source-description-bullet")
abort "completion audit smoke missing no-bundle next steps assertions" unless completion_audit_smoke.include?("no-bundle-next-steps") && completion_audit_smoke.include?("no-bundle-review-step")
abort "completion audit smoke missing issue title assertion" unless completion_audit_smoke.include?("Intégration Chorus Pro PISTE sandbox")
abort "completion audit smoke missing blocker checklist assertion" unless completion_audit_smoke.include?("Credentials Chorus Pro PISTE sandbox requis")
abort "completion audit smoke missing gate checklist assertion" unless completion_audit_smoke.include?("gate: verify-live-pa")
abort "completion audit smoke missing valid-bundle partial failure" unless completion_audit_smoke.include?("manifest still marks external issues partial")
abort "completion audit smoke missing valid-bundle review map assertion" unless completion_audit_smoke.include?("verified gate: verify-live-pa")
abort "completion audit smoke missing valid-bundle path assertion" unless completion_audit_smoke.include?("valid-bundle-path-map")
abort "completion audit smoke missing valid-bundle review command assertion" unless completion_audit_smoke.include?("valid-bundle-review-command")
abort "completion audit smoke missing covered_external fixture" unless completion_audit_smoke.include?("covered_external-no-evidence")
abort "completion audit smoke missing external-gated covered_local fixture" unless completion_audit_smoke.include?("external-gated-covered-local")
abort "completion audit smoke missing wrong external gate fixture" unless completion_audit_smoke.include?("wrong-external-gate")
abort "completion audit smoke missing wrong outcome status fixture" unless completion_audit_smoke.include?("wrong-outcome-status")
abort "completion audit smoke missing generic outcome status fixture" unless completion_audit_smoke.include?("generic-outcome-status")
abort "completion audit smoke missing fully reviewed completion fixture" unless completion_audit_smoke.include?("fully-reviewed-covered-external")
abort "completion audit smoke missing covered_external doc update fixture" unless completion_audit_smoke.include?("covered-external-docs-not-updated")
abort "completion audit smoke missing covered_external invalid bundle fixture" unless completion_audit_smoke.include?("covered-external-invalid-bundle")
abort "completion audit smoke missing covered_external commit mismatch fixture" unless completion_audit_smoke.include?("covered-external-wrong-commit")
abort "completion audit smoke missing covered_external reviewed_at fixture" unless completion_audit_smoke.include?("covered-external-bad-reviewed-at")
abort "completion audit smoke missing covered_external invalid reviewed_at fixture" unless completion_audit_smoke.include?("covered-external-invalid-reviewed-at")
abort "completion audit smoke missing covered_external reviewer placeholder fixture" unless completion_audit_smoke.include?("covered-external-placeholder-reviewer")
abort "manifest verifier missing MANIFEST_PATH fixture support" unless File.read(__FILE__).include?("MANIFEST_PATH")
abort "manifest verifier missing review/audit path fixture support" unless File.read(__FILE__).include?("REVIEW_PATH") && File.read(__FILE__).include?("AUDIT_PATH")
required_evidence_markers = [
  "PASS",
  "Sandbox smoke test passed",
  "PyPI onefacture install ok",
  "npm @onefacture/sdk install ok",
  "KMS active key ok",
  "outcome metric ok"
]
required_evidence_markers.each do |marker|
  abort "external evidence verifier missing success marker #{marker}" unless external_evidence_verifier.include?(marker)
  abort "external evidence checklist missing success marker #{marker}" unless external_evidence.include?(marker)
end
%w[live-pa.log public-sandbox.log sdk-registries.log kms-broker.log outcome-metrics.log all.log].each do |log|
  abort "external evidence verifier missing marker check for #{log}" unless external_evidence_verifier.include?("require_log_marker \"#{log}\"")
end
%w[create-external-evidence verify-external-evidence verify-external-evidence-smoke].each do |target|
  abort "Makefile missing #{target}" unless make_targets[target]
end
abort "review issue coverage mismatch: #{review_issues.inspect}" unless review_issues.sort == expected_issue_numbers
abort "audit issue coverage mismatch: #{audit_issues.inspect}" unless audit_issues.sort == expected_issue_numbers
abort "plan doc missing source title coverage section" unless plan_text.include?("## Titres source couverts")
abort "plan doc missing source metadata coverage section" unless plan_text.include?("## Metadata source couverte")
abort "review doc missing source title coverage section" unless review_text.include?("## Titres source couverts")
abort "completion audit doc missing source title coverage section" unless audit_text.include?("## Titres source couverts")
abort "completion audit doc missing source acceptance criteria coverage section" unless audit_text.include?("## Criteres d'acceptation source couverts")
abort "completion audit doc missing source description coverage section" unless audit_text.include?("## Descriptions source couvertes")
issues.each do |number, title|
  abort "plan doc missing exact backlog title for issue #{number}" unless plan_text.include?(title)
  abort "review doc missing exact backlog title for issue #{number}" unless review_text.include?(title)
  abort "completion audit doc missing exact backlog title for issue #{number}" unless audit_text.include?(title)
end
source_acceptance.each do |number, criteria|
  abort "source acceptance criteria missing for issue #{number}" if criteria.empty?
  criteria.each do |criterion|
    abort "completion audit doc missing exact source acceptance criterion for issue #{number}: #{criterion}" unless audit_text.include?(criterion)
  end
end
source_descriptions.each do |number, descriptions|
  abort "source description bullets missing for issue #{number}" if descriptions.empty?
  descriptions.each do |description|
    abort "completion audit doc missing exact source description for issue #{number}: #{description}" unless audit_text.include?(description)
  end
end
source_metadata.each do |number, metadata|
  plan_metadata = plan_metadata_rows.fetch(number) { abort "plan doc missing metadata row for issue #{number}" }
  abort "plan metadata wave mismatch for issue #{number}" unless plan_metadata.fetch("wave") == metadata.fetch("wave")
  abort "plan metadata labels mismatch for issue #{number}" unless plan_metadata.fetch("labels") == metadata.fetch("labels")
  abort "source labels missing wave label for issue #{number}" unless metadata.fetch("labels").include?("wave:#{metadata.fetch('wave')}")
end
external_issue_numbers = entries.select { |entry| entry["external_gate"] }.map { |entry| entry.fetch("number") }.sort
closure_issue_numbers = external_closure_matrix.scan(/^\|\s*(\d+)\.\s/).flatten.map(&:to_i).sort
abort "external closure matrix issue set mismatch: #{closure_issue_numbers.inspect}" unless closure_issue_numbers == external_issue_numbers
external_issue_numbers.each do |number|
  issue = entries.find { |entry| entry.fetch("number") == number }
  abort "external closure matrix missing issue #{number}" unless external_closure_matrix.include?("#{number}. #{issue.fetch('title')}")
  abort "external closure matrix missing gate for issue #{number}" unless external_closure_matrix.include?("`make #{issue.fetch('external_gate')}`")
  abort "external closure matrix missing covered_external update for issue #{number}" unless external_closure_matrix.include?("Mark issue #{number} `covered_external`")
end
%w[
  live-pa.log
  public-sandbox.log
  sdk-registries.log
  kms-broker.log
  outcome-metrics.log
  ONEFACTURE_MIN_RETRIED_INVOICES
  ONEFACTURE_BASELINE_RETRY_SUCCESS_RATE
  KMS active key ok
  PyPI onefacture install ok
  npm @onefacture/sdk install ok
].each do |marker|
  abort "external closure matrix missing marker #{marker}" unless external_closure_matrix.include?(marker)
end
abort "review doc missing covered_external review standard" unless review_text.include?("covered_external") && review_text.include?("reviewed_evidence")
abort "completion audit doc missing covered_external review standard" unless audit_text.include?("covered_external") && audit_text.include?("reviewed_evidence.bundle")
abort "review doc missing local-acceptance CI/local gate summary" unless review_text.include?("local-acceptance") && review_text.include?("gofmt") && review_text.include?("parse YAML")
abort "completion audit doc missing local-acceptance CI/local gate summary" unless audit_text.include?("local-acceptance") && audit_text.include?("gofmt") && audit_text.include?("parse YAML")
abort "review doc missing current PyPI negative evidence" unless review_text.include?("make verify-sdk-registries") && review_text.include?("PyPI onefacture install failed")
abort "review doc missing current npm negative evidence" unless review_text.include?("make verify-sdk-registries") && review_text.include?("npm @onefacture/sdk install failed")
abort "completion audit doc missing current PyPI negative evidence" unless audit_text.include?("make verify-sdk-registries") && audit_text.include?("PyPI onefacture install failed")
abort "completion audit doc missing current npm negative evidence" unless audit_text.include?("make verify-sdk-registries") && audit_text.include?("npm @onefacture/sdk install failed")

ci_expectations = {
  "backlog-acceptance-manifest" => {
    "run" => "ruby scripts/verify_backlog_acceptance_manifest.rb",
    "uses" => []
  },
  "local-acceptance" => {
    "run" => "make verify-local",
    "uses" => ["actions/setup-go@v5", "actions/setup-python@v5", "actions/setup-node@v4"]
  },
  "sdk-artifacts" => {
    "run" => "bash scripts/verify_sdk_release_artifacts.sh",
    "uses" => ["actions/setup-python@v5", "actions/setup-node@v4"]
  },
  "external-gate-smokes" => {
    "run" => "bash scripts/verify_external_gate_smokes.sh",
    "uses" => ["actions/setup-go@v5", "actions/setup-python@v5", "actions/setup-node@v4"]
  },
  "external-evidence-verifier" => {
    "run" => "bash scripts/smoke_external_evidence_bundle.sh",
    "uses" => []
  },
  "backlog-completion-audit" => {
    "run" => "bash scripts/smoke_backlog_completion_audit.sh",
    "uses" => []
  },
  "external-evidence-collector" => {
    "run" => "bash scripts/smoke_external_evidence_collector.sh",
    "uses" => []
  },
  "external-env-readiness" => {
    "run" => "bash scripts/smoke_external_acceptance_env.sh",
    "uses" => []
  },
  "external-evidence-review" => {
    "run" => "bash scripts/smoke_external_evidence_review.sh",
    "uses" => []
  }
}
ci_expectations.each do |job_name, expectation|
  job = ci_jobs.fetch(job_name) { abort "CI job missing: #{job_name}" }
  steps = job.fetch("steps")
  runs = steps.filter_map { |step| step["run"] }
  uses = steps.filter_map { |step| step["uses"] }
  abort "CI job #{job_name} missing run #{expectation['run']}" unless runs.include?(expectation["run"])
  expectation["uses"].each do |required_action|
    abort "CI job #{job_name} missing action #{required_action}" unless uses.include?(required_action)
  end
end

expected_external_smoke_scripts = [
  "scripts/smoke_public_sandbox_local.sh",
  "scripts/smoke_live_pa_gate_local.sh",
  "scripts/smoke_kms_gate_local.sh",
  "scripts/smoke_outcome_metrics_gate_local.sh",
  "scripts/verify_sdk_release_artifacts.sh",
  "scripts/verify_external_acceptance.sh"
]
expected_external_smoke_scripts.each do |script|
  abort "external smokes wrapper missing #{script}" unless external_smokes.include?(script)
end

evidence_smoke = root.join("scripts/smoke_external_evidence_bundle.sh").read
abort "external evidence smoke missing missing-marker rejection" unless evidence_smoke.include?("missing success marker")

external_acceptance_path.readlines.each_with_index do |line, index|
  next unless line.include?("<<'PY'")

  body = []
  external_acceptance_path.readlines[(index + 1)..].each do |candidate|
    break if candidate.chomp == "PY"

    body << candidate
  end
  Tempfile.create(["external-acceptance-python", ".py"]) do |file|
    file.write(body.join)
    file.flush
    ok = system(
      python,
      "-c",
      "import ast, pathlib, sys; ast.parse(pathlib.Path(sys.argv[1]).read_text())",
      file.path
    )
    abort "invalid embedded Python heredoc after line #{index + 1} in #{external_acceptance_path}" unless ok
  end
end

valid_statuses = %w[covered_local covered_external partial_external partial_outcome_external]
external_gate_commands = [
  "make verify-live-pa",
  "make verify-public-sandbox",
  "make verify-sdk-registries",
  "make verify-kms-broker",
  "make verify-outcome-metrics"
]
seen = {}

entries.each do |entry|
  number = entry.fetch("number")
  title = entry.fetch("title")
  status = entry.fetch("status")

  abort "duplicate manifest issue #{number}" if seen[number]
  seen[number] = true
  abort "manifest issue #{number} missing from backlog" unless issues.key?(number)
  abort "title mismatch for issue #{number}: #{title.inspect} != #{issues[number].inspect}" unless title == issues[number]
  abort "invalid status for issue #{number}: #{status}" unless valid_statuses.include?(status)
  abort "issue #{number} cannot use partial_outcome_external" if status == "partial_outcome_external" && number != 21
  abort "issue 21 must use partial_outcome_external until outcome evidence is reviewed" if number == 21 && status == "partial_external"
  if number == 11 && status != "covered_external"
    abort "issue 11 blocker missing current PyPI negative evidence" unless entry.fetch("external_blockers").join(" ").include?("PyPI onefacture install failed")
  end
  if number == 12 && status != "covered_external"
    abort "issue 12 blocker missing current npm negative evidence" unless entry.fetch("external_blockers").join(" ").include?("npm @onefacture/sdk install failed")
  end

  artifacts = entry.fetch("artifacts")
  abort "issue #{number} has no artifacts" if artifacts.empty?
  artifacts.each do |artifact|
    path = root.join(artifact)
    abort "issue #{number} artifact missing: #{artifact}" unless path.exist?
  end

  commands = entry.fetch("verification_commands")
  abort "issue #{number} has no verification commands" if commands.empty?
  plan_commands = plan_rows[number]
  abort "issue #{number} missing from plan" unless plan_commands
  abort "issue #{number} has no plan gates" if plan_commands.empty?
  abort "issue #{number} plan gates do not match manifest commands" unless plan_commands.sort == commands.sort

  commands.each do |command|
    next unless command.start_with?("make ")

    target = command.split.fetch(1)
    abort "issue #{number} references missing make target: #{target}" unless make_targets[target]
  end
  has_external_gate_command = commands.any? { |command| external_gate_commands.include?(command) }

  blockers = entry.fetch("external_blockers", [])
  if status == "covered_local"
    abort "issue #{number} is local but has external blockers" unless blockers.empty?
    abort "issue #{number} uses external gate command but is marked covered_local" if has_external_gate_command
  else
    abort "issue #{number} is external but has no external verification command" unless has_external_gate_command
    gate = entry["external_gate"]
    abort "issue #{number} is external but has no external gate" unless gate
    abort "issue #{number} references missing external make target: #{gate}" unless make_targets[gate]
    abort "issue #{number} external_gate is not listed in verification_commands" unless commands.include?("make #{gate}")
    mode = gate.delete_prefix("verify-")
    abort "issue #{number} references unsupported external gate mode: #{mode}" unless external_modes[mode]
    if status == "covered_external"
      abort "issue #{number} is covered_external but still has external blockers" unless blockers.empty?
      evidence = entry["reviewed_evidence"]
      abort "issue #{number} is covered_external but has no reviewed_evidence" unless evidence.is_a?(Hash)
      %w[bundle commit_sha reviewed_at reviewed_by].each do |field|
        abort "issue #{number} reviewed_evidence missing #{field}" if evidence[field].to_s.strip.empty?
      end
      unless evidence.fetch("commit_sha").match?(/\A[0-9a-f]{40}\z/)
        abort "issue #{number} reviewed_evidence commit_sha must be a full lowercase git SHA"
      end
      unless evidence.fetch("reviewed_at").match?(/\A\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\z/)
        abort "issue #{number} reviewed_evidence reviewed_at must be an ISO-8601 UTC timestamp"
      end
      begin
        Time.iso8601(evidence.fetch("reviewed_at"))
      rescue ArgumentError
        abort "issue #{number} reviewed_evidence reviewed_at must be a valid UTC timestamp"
      end
      if evidence.fetch("reviewed_by").match?(/\A(todo|tbd|placeholder|reviewer|unknown)\z/i)
        abort "issue #{number} reviewed_evidence reviewed_by must name the actual reviewer"
      end
      unless manifest_fixture || evidence.fetch("bundle").start_with?("docs/operations/evidence/")
        abort "issue #{number} reviewed_evidence bundle must be under docs/operations/evidence"
      end
      unless manifest_fixture || root.join(evidence.fetch("bundle")).directory?
        abort "issue #{number} reviewed_evidence bundle path does not exist"
      end
      review_marker = "#{number}. #{title}: covered_external"
      audit_marker = "| #{number} | covered_external |"
      abort "review doc missing covered_external marker for issue #{number}" unless review_text.include?(review_marker)
      abort "completion audit doc missing covered_external marker for issue #{number}" unless audit_text.include?(audit_marker)
    else
      abort "issue #{number} is external but has no blockers" if blockers.empty?
      abort "issue #{number} has reviewed_evidence before external completion" if entry.key?("reviewed_evidence")
    end
  end
end

missing = issues.keys - seen.keys
abort "manifest missing issues: #{missing.join(', ')}" unless missing.empty?

puts "backlog acceptance manifest ok"
