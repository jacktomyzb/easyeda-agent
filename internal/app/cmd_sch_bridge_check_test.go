package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// The connector shape: one BRIDGE tree (共线合并短路) — a wire tree spanning two
// wires whose anchored flags carry two DIFFERENT net names, plus one ORPHAN tree
// (a stub touching a pin with no net flag).
func TestParseAndRenderBridge_BridgeAndOrphan(t *testing.T) {
	result := map[string]any{
		"passed": false,
		"summary": map[string]any{
			"trees":          float64(2),
			"bridges":        float64(1),
			"orphans":        float64(1),
			"wireTreesTotal": float64(9),
		},
		"trees": []any{
			map[string]any{
				"kind":    "BRIDGE",
				"wireIds": []any{"w1", "w2"},
				"flagIds": []any{"f1", "f2"},
				"pins":    []any{"U1:5", "R3:1"},
				"nets":    []any{"GND", "VCC"},
			},
			map[string]any{
				"kind":    "ORPHAN",
				"wireIds": []any{"w7"},
				"flagIds": []any{},
				"pins":    []any{"C2:2"},
				"nets":    []any{},
			},
		},
	}
	rep, err := parseBridgeReport(result)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rep.Passed {
		t.Error("expected passed=false")
	}
	if rep.Summary.Bridges != 1 || rep.Summary.Orphans != 1 || len(rep.Trees) != 2 {
		t.Errorf("unexpected summary/trees: %+v", rep)
	}

	var buf bytes.Buffer
	renderBridgeReport(rep, &buf)
	out := buf.String()
	for _, want := range []string{"ERROR", "BRIDGE", "nets=[GND,VCC]", "pins=[U1:5,R3:1]", "w1, w2", "WARN", "ORPHAN", "C2:2", "bridge", "orphan"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// Clean board: no problem trees → passed, and the "no bridges or orphans" line.
func TestRenderBridge_Clean(t *testing.T) {
	rep := bridgeReport{Passed: true, Summary: bridgeSummary{WireTreesTotal: 12}}
	var buf bytes.Buffer
	renderBridgeReport(rep, &buf)
	if !strings.Contains(buf.String(), "no bridges or orphans") {
		t.Errorf("expected clean line, got:\n%s", buf.String())
	}
}

// --json output must be wrapped in the {id,type,version,ok,result} envelope, so a
// uniform-envelope parser reading result.trees works consistently with sch check.
func TestEncodeResultEnvelope_BridgeReport(t *testing.T) {
	rep := bridgeReport{
		Passed:  false,
		Summary: bridgeSummary{Trees: 1, Bridges: 1},
		Trees: []bridgeTree{
			{Kind: "BRIDGE", WireIds: []string{"w1", "w2"}, Nets: []string{"GND", "VCC"}, Pins: []string{"U1:5"}},
		},
	}
	res := &actionResult{ID: "req-9", Type: "response", Version: "1", OK: true}

	var buf bytes.Buffer
	if err := encodeResultEnvelope(res, rep, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var env struct {
		ID     string `json:"id"`
		OK     bool   `json:"ok"`
		Result struct {
			Passed bool         `json:"passed"`
			Trees  []bridgeTree `json:"trees"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\n%s", err, buf.String())
	}
	if env.ID != "req-9" || !env.OK {
		t.Errorf("envelope metadata lost: %+v", env)
	}
	if len(env.Result.Trees) != 1 || env.Result.Trees[0].Kind != "BRIDGE" || len(env.Result.Trees[0].Nets) != 2 {
		t.Errorf("result.trees not reachable via envelope: %+v", env.Result)
	}
}
