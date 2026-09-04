package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestGenerateComparisonReport(t *testing.T) {
	rep := GenerateComparisonReport()

	if rep.SelectedDriver != "raw_cdp" {
		t.Fatalf("expected selected_driver = raw_cdp, got %s", rep.SelectedDriver)
	}

	requiredDrivers := []string{"raw_cdp", "chromedp_cdproto", "rod", "warm_playwright"}
	for _, rd := range requiredDrivers {
		d, ok := rep.Drivers[rd]
		if !ok {
			t.Fatalf("missing driver profile for %s", rd)
		}
		if len(d.ScenarioMetrics) == 0 {
			t.Fatalf("no scenario metrics for driver %s", rd)
		}
		score, ok := rep.DecisionMatrix[rd]
		if !ok {
			t.Fatalf("missing decision score for driver %s", rd)
		}
		if score.TotalScore <= 0 {
			t.Fatalf("invalid total score %d for %s", score.TotalScore, rd)
		}
	}

	// Test serialization
	var buf bytes.Buffer
	if err := PrintComparisonReport(&buf); err != nil {
		t.Fatalf("PrintComparisonReport failed: %v", err)
	}

	var parsed DriverComparisonReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal comparison report JSON: %v", err)
	}
	if parsed.SelectedDriver != "raw_cdp" {
		t.Fatalf("parsed report mismatch: %s", parsed.SelectedDriver)
	}
}
