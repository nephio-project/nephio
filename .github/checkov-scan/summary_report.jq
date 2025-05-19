map({
  check_type: .check_type,
  summary: {
    passed: .summary.passed,
    failed: .summary.failed,
    skipped: .summary.skipped,
    parsing_errors: .summary.parsing_errors,
    resource_count: .summary.resource_count,
    checkov_version: .summary.checkov_version,
    severity_count: (
      .results.failed_checks
      | group_by(.severity)
      | map({ (.[0].severity // "UNKNOWN"): length })
      | add
    )
  },
  results: {
    failed_checks: [
      .results.failed_checks[]
      | select(.severity != "LOW" and .severity != "INFO")
      | {
          check_id,
          check_name,
          result: .check_result.result,
          file_path,
          guideline,
          severity
        }
    ],
    skipped_checks: .results.skipped_checks,
    parsing_errors: .results.parsing_errors
  }
})
