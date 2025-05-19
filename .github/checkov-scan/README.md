## Checkov Scan Severity Enrichment Utility

This folder contains a custom toolchain to post-process [Checkov](https://github.com/bridgecrewio/checkov) scan results, enrich them with severity information, and generate a human-readable summary.

### Contents

- `main.go` – Go script to enrich Checkov results using a severity mapping.
- `severity_map.json` – JSON mapping of Checkov check IDs to severity levels (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`). [Reference](https://docs.prismacloud.io/en/enterprise-edition/policy-reference/kubernetes-policies/kubernetes-policy-index/kubernetes-policy-index)
- `summary_report.jq` – JQ expression to generate a summary report from enriched Checkov output.
- `go.mod` – Go module definition.

