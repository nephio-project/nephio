package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// Structure for severity map: map[check_id]severity
type SeverityMap map[string]string

// Minimal structure to locate and update severity fields
type Check struct {
	CheckID  string  `json:"check_id"`
	Severity *string `json:"severity"`
}

type Results struct {
	FailedChecks  []json.RawMessage `json:"failed_checks"`
	PassedChecks  []json.RawMessage `json:"passed_checks"`
	SkippedChecks []json.RawMessage `json:"skipped_checks"`
}

type CheckovReport struct {
	Results json.RawMessage `json:"results"`
}

func patchSeverity(checks []json.RawMessage, severityMap SeverityMap) []json.RawMessage {
	for i, raw := range checks {
		var chk map[string]json.RawMessage
		if err := json.Unmarshal(raw, &chk); err != nil {
			continue
		}

		var checkID string
		if err := json.Unmarshal(chk["check_id"], &checkID); err != nil {
			continue
		}

		if _, exists := chk["severity"]; exists {
			var s interface{}
			_ = json.Unmarshal(chk["severity"], &s)
			if s == nil {
				if sev, ok := severityMap[checkID]; ok {
					newSeverity, _ := json.Marshal(sev)
					chk["severity"] = newSeverity
				}
			}
		}

		checks[i], _ = json.Marshal(chk)
	}
	return checks
}

func loadSeverityMap(path string) (SeverityMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m SeverityMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func main() {
	// Flag arguments
	inputFile := flag.String("input", "", "Path to Checkov raw JSON report")
	mapFile := flag.String("map", "", "Path to severity mapping JSON file")
	outputFile := flag.String("output", "", "Path to write enriched report")
	flag.Parse()

	if *inputFile == "" || *mapFile == "" || *outputFile == "" {
		fmt.Println("Usage: ./enrich-severity -input <checkov_raw.json.json> -map <severity_map.json> -output <checkov_enriched.json>")
		os.Exit(1)
	}

	// Load severity map
	severityMap, err := loadSeverityMap(*mapFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load severity map: %v\n", err)
		os.Exit(1)
	}

	// Load input Checkov report
	raw, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input report: %v\n", err)
		os.Exit(1)
	}

	var reports []json.RawMessage
	if err := json.Unmarshal(raw, &reports); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid JSON format in input: %v\n", err)
		os.Exit(1)
	}

	for i, report := range reports {
		var temp struct {
			Results Results `json:"results"`
		}
		if err := json.Unmarshal(report, &temp); err != nil {
			continue
		}

		temp.Results.FailedChecks = patchSeverity(temp.Results.FailedChecks, severityMap)
		temp.Results.PassedChecks = patchSeverity(temp.Results.PassedChecks, severityMap)
		temp.Results.SkippedChecks = patchSeverity(temp.Results.SkippedChecks, severityMap)

		updatedResults, _ := json.Marshal(temp.Results)
		var updatedReport map[string]json.RawMessage
		json.Unmarshal(reports[i], &updatedReport)
		updatedReport["results"] = updatedResults
		reports[i], _ = json.Marshal(updatedReport)
	}

	// Write enriched report
	output, _ := json.MarshalIndent(reports, "", "  ")
	if err := os.WriteFile(*outputFile, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Severity enriched report written to %s\n", *outputFile)
}
