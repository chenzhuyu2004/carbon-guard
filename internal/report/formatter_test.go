package report

import (
	"encoding/json"
	"math"
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

func TestBuildFromEmissionsTextUsesAutoUnitForReadability(t *testing.T) {
	out := BuildFromEmissions(300, false, 0.0067, BuildOptions{})
	if !strings.Contains(out, "Estimated Emissions: 6.70 gCO2 (0.0067 kgCO2)") {
		t.Fatalf("expected auto-scaled gCO2 display, got: %s", out)
	}
}

func TestAutoScaledEmissionKeepsValueInTargetRange(t *testing.T) {
	cases := []float64{0.0067, 12, 1200, 0.0002, 1_200_000}
	for _, emissionsKg := range cases {
		scaled := autoScaledEmission(emissionsKg)
		abs := math.Abs(scaled.Value)
		if abs < autoUnitMinValue || abs >= autoUnitMaxValue {
			t.Fatalf("value out of target range for %.6fkg: %.6f %s", emissionsKg, scaled.Value, scaled.Symbol)
		}
	}
}
