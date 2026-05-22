#!/usr/bin/env ruby
# frozen_string_literal: true

require "json"
require "open3"
require "pathname"

if ARGV.length != 1
  warn "usage: #{$PROGRAM_NAME} docs/operations/evidence/YYYY-MM-DD-external-acceptance"
  exit 2
end

root = Pathname.new(__dir__).join("..").expand_path
bundle = Pathname.new(ARGV.fetch(0))
bundle = root.join(bundle) unless bundle.absolute?

verifier = root.join("scripts/verify_external_evidence_bundle.sh")
stdout, stderr, status = Open3.capture3("bash", verifier.to_s, bundle.to_s)
unless status.success?
  warn stderr unless stderr.empty?
  warn stdout unless stdout.empty?
  exit status.exitstatus || 1
end

manifest = JSON.parse(root.join("docs/backlog/github-issues-vagues-acceptance.json").read)
external_issues = manifest.fetch("issues").reject { |issue| issue.fetch("status") == "covered_local" }
summary = bundle.join("summary.md").read
summary_commit = summary[/^Commit SHA:\s*(.+)$/, 1].to_s.strip
summary_branch = summary[/^Branch:\s*(.+)$/, 1].to_s.strip
summary_operator = summary[/^Operator:\s*(.+)$/, 1].to_s.strip
summary_timestamp = summary[/^Timestamp:\s*(.+)$/, 1].to_s.strip
summary_environment = summary[/^Environment:\s*(.+)$/, 1].to_s.strip
summary_reruns = summary[/^Reruns:\s*(.+)$/, 1].to_s.strip
summary_links = summary[/^Links:\s*(.+)$/, 1].to_s.strip
bundle_label = bundle.to_s.delete_prefix("#{root}/")

puts stdout unless stdout.empty?
puts "External evidence review checklist: #{bundle}"
puts "Evidence commit: #{summary_commit}"
puts "Evidence branch: #{summary_branch}"
puts "Evidence operator: #{summary_operator}"
puts "Evidence timestamp: #{summary_timestamp}"
puts "Evidence environment: #{summary_environment}"
puts "Evidence reruns: #{summary_reruns}"
puts "Evidence links: #{summary_links}"
external_issues.each do |issue|
  puts format(
    "- #%<number>02d %<title>s | gate: %<gate>s | current status: %<status>s",
    number: issue.fetch("number"),
    title: issue.fetch("title"),
    gate: issue.fetch("external_gate"),
    status: issue.fetch("status")
  )
end
puts "Manifest update template after human review:"
puts JSON.pretty_generate(
  "status" => "covered_external",
  "external_blockers" => [],
  "reviewed_evidence" => {
    "bundle" => bundle_label,
    "commit_sha" => summary_commit,
    "reviewed_at" => summary_timestamp,
    "reviewed_by" => summary_operator
  }
)
puts "Review document markers after human review:"
external_issues.each do |issue|
  puts "#{issue.fetch("number")}. #{issue.fetch("title")}: covered_external"
end
puts "Completion audit status rows after human review:"
external_issues.each do |issue|
  puts "| #{issue.fetch("number")} | covered_external |"
end
puts "Apply that template to each reviewed external issue in docs/backlog/github-issues-vagues-acceptance.json, then update docs/backlog/github-issues-vagues-review.md and docs/backlog/github-issues-vagues-completion-audit.md together."
puts "Final gate: make audit-backlog-completion"
