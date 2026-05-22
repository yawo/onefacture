#!/usr/bin/env ruby
# frozen_string_literal: true

require "json"
require "open3"
require "pathname"

root = Pathname.new(__dir__).join("..").expand_path
backlog_path = root.join("docs/backlog/github-issues-vagues.md")
manifest_path = Pathname.new(ENV.fetch("MANIFEST_PATH", root.join("docs/backlog/github-issues-vagues-acceptance.json").to_s))
manifest_path = root.join(manifest_path) unless manifest_path.absolute?
manifest_verifier = root.join("scripts/verify_backlog_acceptance_manifest.rb")
evidence_verifier = root.join("scripts/verify_external_evidence_bundle.sh")

def run_or_fail(*command)
  stdout, stderr, status = Open3.capture3(*command)
  unless status.success?
    warn stderr unless stderr.empty?
    warn stdout unless stdout.empty?
    exit status.exitstatus || 1
  end
  stdout
end

manifest_env = { "MANIFEST_PATH" => manifest_path.to_s }
%w[REVIEW_PATH AUDIT_PATH].each do |name|
  manifest_env[name] = ENV[name] if ENV[name].to_s.strip != ""
end
run_or_fail(manifest_env, "ruby", manifest_verifier.to_s)

manifest = JSON.parse(manifest_path.read)
issues = manifest.fetch("issues")
issue_statuses = issues.to_h { |issue| [issue.fetch("number"), issue.fetch("status")] }
source_descriptions = Hash.new { |hash, key| hash[key] = [] }
source_acceptance = Hash.new { |hash, key| hash[key] = [] }
current_issue = nil
in_acceptance = false
backlog_path.read.each_line do |line|
  if (match = line.match(/^### (\d+)\./))
    current_issue = match[1].to_i
    in_acceptance = false
  elsif current_issue && line.include?("**Critères d'acceptation**")
    in_acceptance = true
  elsif current_issue && in_acceptance && (match = line.match(/^- (.+)$/))
    source_acceptance[current_issue] << match[1].strip
  elsif current_issue && !in_acceptance && (match = line.match(/^- (.+)$/))
    source_descriptions[current_issue] << match[1].strip
  end
end
complete_statuses = %w[covered_local covered_external]
partials = issues.reject { |issue| complete_statuses.include?(issue.fetch("status")) }
reviewed_external = issues.select { |issue| issue.fetch("status") == "covered_external" }

puts "Objective: planifier, implementer et reviewer chaque issue de docs/backlog/github-issues-vagues.md."
puts "Prompt-to-artifact checklist: #{issues.length} issues mapped in docs/backlog/github-issues-vagues-acceptance.json."
puts "Local verification gate: make verify-local"

issues.each do |issue|
  number = issue.fetch("number")
  title = issue.fetch("title")
  artifacts = issue.fetch("artifacts")
  commands = issue.fetch("verification_commands")
  missing_artifacts = artifacts.reject { |artifact| root.join(artifact).exist? }
  artifact_status = missing_artifacts.empty? ? "artifacts ok" : "missing artifacts: #{missing_artifacts.join(", ")}"
  puts format(
    "- #%<number>02d %<title>s | %<status>s | %<artifact_status>s | commands: %<commands>s",
    number: number,
    title: title,
    status: issue.fetch("status"),
    artifact_status: artifact_status,
    commands: commands.join(", ")
  )
  next if missing_artifacts.empty?

  warn "Completion audit: incomplete; manifest artifact paths are missing."
  exit 1
end

status_counts = issues.group_by { |issue| issue.fetch("status") }.transform_values(&:length)
status_counts.sort.each do |status, count|
  puts "- #{status}: #{count}"
end

puts "Source acceptance checklist:"
source_acceptance.sort.each do |number, criteria|
  status = issue_statuses.fetch(number)
  criteria.each do |criterion|
    puts format("- #%<number>02d | %<status>s | %<criterion>s", number: number, status: status, criterion: criterion)
  end
end

puts "Source description checklist:"
source_descriptions.sort.each do |number, descriptions|
  status = issue_statuses.fetch(number)
  descriptions.each do |description|
    puts format("- #%<number>02d | %<status>s | %<description>s", number: number, status: status, description: description)
  end
end

unless reviewed_external.empty?
  current_commit = run_or_fail("git", "rev-parse", "HEAD").strip
  verified_bundles = {}
  puts "Reviewed external evidence:"
  reviewed_external.each do |issue|
    evidence = issue.fetch("reviewed_evidence")
    bundle_path = Pathname.new(evidence.fetch("bundle"))
    bundle_path = root.join(bundle_path) unless bundle_path.absolute?
    unless evidence.fetch("commit_sha") == current_commit
      warn "Completion audit: incomplete; reviewed evidence commit for issue #{issue.fetch("number")} does not match HEAD."
      exit 1
    end
    unless verified_bundles[bundle_path.to_s]
      stdout, stderr, status = Open3.capture3("bash", evidence_verifier.to_s, bundle_path.to_s)
      unless status.success?
        warn stderr unless stderr.empty?
        warn stdout unless stdout.empty?
        warn "Completion audit: incomplete; reviewed evidence bundle failed verification."
        exit 1
      end
      verified_bundles[bundle_path.to_s] = true
    end
    puts format(
      "- #%<number>02d %<title>s | gate: %<gate>s | bundle: %<bundle>s | commit: %<commit>s",
      number: issue.fetch("number"),
      title: issue.fetch("title"),
      gate: issue.fetch("external_gate"),
      bundle: evidence.fetch("bundle"),
      commit: evidence.fetch("commit_sha")
    )
  end
end

if partials.empty?
  puts "Completion audit: complete; all manifest issues are covered locally or by reviewed external evidence."
  exit 0
end

partial_numbers = partials.map { |issue| issue.fetch("number") }
puts "Completion audit: incomplete; external evidence is still required for issues #{partial_numbers.join(", ")}."
puts "External blocker checklist:"
partials.each do |issue|
  blockers = issue.fetch("external_blockers", [])
  puts format(
    "- #%<number>02d %<title>s | gate: %<gate>s | blockers: %<blockers>s",
    number: issue.fetch("number"),
    title: issue.fetch("title"),
    gate: issue.fetch("external_gate"),
    blockers: blockers.join("; ")
  )
end

bundle = ENV["BUNDLE"].to_s.strip
if bundle.empty?
  puts "BUNDLE not supplied. Provide BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance after collecting live evidence."
  puts "External evidence next steps:"
  puts "- make check-external-env"
  puts "- make collect-external-evidence STAMP=YYYY-MM-DD"
  puts "- make review-external-evidence BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance"
  puts "- make audit-backlog-completion BUNDLE=docs/operations/evidence/YYYY-MM-DD-external-acceptance"
  exit 1
end

bundle_path = Pathname.new(bundle)
bundle_path = root.join(bundle_path) unless bundle_path.absolute?
stdout, stderr, status = Open3.capture3("bash", evidence_verifier.to_s, bundle_path.to_s)
unless status.success?
  warn stderr unless stderr.empty?
  warn stdout unless stdout.empty?
  warn "Completion audit: incomplete; evidence bundle failed verification."
  exit 1
end

puts stdout unless stdout.empty?
puts "Verified external evidence is ready for review: #{bundle_path}"
puts "Review command: make review-external-evidence BUNDLE=#{bundle_path}"
partials.each do |issue|
  puts format(
    "- #%<number>02d %<title>s | verified gate: %<gate>s",
    number: issue.fetch("number"),
    title: issue.fetch("title"),
    gate: issue.fetch("external_gate")
  )
end
warn "Completion audit: evidence bundle is valid, but manifest still marks external issues partial. Update manifest/review/audit after reviewing evidence."
exit 1
