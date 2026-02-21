package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildFromEmissionsJSONIncludesBudgetAndBaseline(t *testing.T) {
	out := BuildFromEmissions(300, true, 0.0123, BuildOptions{
		BudgetKg:   0.01,
		BaselineKg: 0.02,
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if payload["duration_seconds"].(float64) != 300 {
		t.Fatalf("duration_seconds mismatch: %#v", payload["duration_seconds"])
	}
	if _, ok := payload["budget_kg"]; !ok {
		t.Fatalf("expected budget_kg in payload")
	}
	if payload["budget_exceeded"] != true {
		t.Fatalf("expected budget_exceeded=true, got %#v", payload["budget_exceeded"])
	}
	if _, ok := payload["delta_vs_baseline_pct"]; !ok {
		t.Fatalf("expected delta_vs_baseline_pct in payload")
	}
}

func TestBuildFromEmissionsTextIncludesBudgetAndBaselineLines(t *testing.T) {
	out := BuildFromEmissions(300, false, 0.007, BuildOptions{
		BudgetKg:   0.01,
		BaselineKg: 0.008,
	})

	checks := []string{
		"Carbon Report",
		"Carbon Score:",
		"Fun Facts:",
		"Budget:",
		"Baseline:",
	}
	for _, c := range checks {
		if !strings.Contains(out, c) {
			t.Fatalf("expected output to contain %q", c)
		}
	}
}
