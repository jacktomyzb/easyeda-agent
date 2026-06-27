package app

import (
	"bytes"
	"strings"
	"testing"
)

// The connector shape: floating pins grouped by component (the ESP32 case —
// many unused IOs on U1).
func TestParseAndRenderCheck_Floating(t *testing.T) {
	result := map[string]any{
		"passed": false,
		"summary": map[string]any{
			"floatingPins":           float64(33),
			"componentsWithFloating": float64(1),
			"total":                  float64(1),
		},
		"findings": []any{
			map[string]any{
				"type":       "floating-pin",
				"level":      "warn",
				"designator": "U1",
				"pins":       []any{"4", "5", "15", "36", "37"},
				"count":      float64(5),
				"message":    "5 个引脚悬空(无导线连接,未打 NC 标识)",
			},
		},
	}
	rep, err := parseCheckReport(result)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rep.Passed {
		t.Error("expected passed=false")
	}
	if rep.Summary.FloatingPins != 33 || len(rep.Findings) != 1 {
		t.Errorf("unexpected summary/findings: %+v", rep)
	}

	var buf bytes.Buffer
	renderCheckReport(rep, &buf)
	out := buf.String()
	for _, want := range []string{"WARN", "floating-pin", "U1", "[4,5,15,36,37]", "sch no-connect"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// Clean board: no findings → passed, and the "no findings" line.
func TestRenderCheck_Clean(t *testing.T) {
	rep := checkReport{Passed: true, Summary: checkSummary{}}
	var buf bytes.Buffer
	renderCheckReport(rep, &buf)
	if !strings.Contains(buf.String(), "no findings") {
		t.Errorf("expected clean line, got:\n%s", buf.String())
	}
}
