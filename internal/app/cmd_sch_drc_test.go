package app

import (
	"bytes"
	"strings"
	"testing"
)

// The connector-normalized aggregate shape (blink case): 3 warnings, no detail.
func TestParseDrcReport_Aggregate(t *testing.T) {
	result := map[string]any{
		"passed": false,
		"fatal":  float64(0),
		"summary": map[string]any{
			"fatal": float64(0), "error": float64(0), "warn": float64(3),
			"info": float64(0), "unknown": float64(0), "total": float64(3),
		},
		"violations": []any{
			map[string]any{"level": "warn", "type": "warn", "count": float64(3)},
		},
	}
	rep, err := parseDrcReport(result)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rep.Passed {
		t.Error("expected passed=false")
	}
	if rep.Fatal != 0 {
		t.Errorf("expected fatal=0, got %d", rep.Fatal)
	}
	if rep.Summary.Warn != 3 || rep.Summary.Total != 3 {
		t.Errorf("expected 3 warn/total, got %+v", rep.Summary)
	}
}

// A fully-detailed, fatal-bearing shape: human view must show rule+message+coords
// and the fatal count must drive a non-zero exit.
func TestRenderAndExit_FatalDetail(t *testing.T) {
	x, y := 100.0, 200.0
	rep := drcReport{
		Passed: false,
		Fatal:  1,
		Summary: drcSummary{
			Error: 1, Warn: 1, Total: 2,
		},
		Violations: []drcViolation{
			{Level: "error", Rule: "endpoints-overlap", Message: "端点重叠且未连接", X: &x, Y: &y, Designators: []string{"U1"}},
			{Level: "warn", Rule: "floating-io", Message: "IO 悬空"},
		},
	}
	var buf bytes.Buffer
	renderDrcReport(rep, false, &buf)
	out := buf.String()
	for _, want := range []string{"ERROR", "endpoints-overlap", "端点重叠且未连接", "@(100.00,200.00)", "[U1]", "WARN", "floating-io", "✗ 1 fatal"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// Warnings-only must render as a passing gate (the S5 "0 fatal" semantic).
func TestRenderWarnOnlyGatePasses(t *testing.T) {
	rep := drcReport{
		Passed:     false,
		Fatal:      0,
		Summary:    drcSummary{Warn: 3, Total: 3},
		Violations: []drcViolation{{Level: "warn", Count: intp(3)}},
	}
	var buf bytes.Buffer
	renderDrcReport(rep, false, &buf)
	out := buf.String()
	if !strings.Contains(out, "0 fatal") || !strings.Contains(out, "gate passes") {
		t.Errorf("expected warn-only gate-pass line, got:\n%s", out)
	}
	if !strings.Contains(out, "no per-item detail") {
		t.Errorf("aggregate-only node should explain the missing detail, got:\n%s", out)
	}
}

func intp(n int) *int { return &n }
