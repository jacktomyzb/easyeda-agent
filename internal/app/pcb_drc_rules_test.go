package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ─── parseRulesJSON ──────────────────────────────────────────────────

func TestParseRulesJSON_InlineArray(t *testing.T) {
	got, err := parseRulesJSON(`[{"net":"A","trackWidth":10},{"net":"B","clearance":8}]`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0]["net"] != "A" {
		t.Errorf("first entry net = %v, want A", got[0]["net"])
	}
}

func TestParseRulesJSON_WrappedObject(t *testing.T) {
	// A verbatim read result saved back to disk — the connector returns
	// `{netRules: [...], count: N}`. We pick the first known array field.
	got, err := parseRulesJSON(`{"netRules":[{"net":"A"}],"count":1}`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0]["net"] != "A" {
		t.Errorf("expected [{net:A}], got %+v", got)
	}
}

func TestParseRulesJSON_WrappedObjectNetByNet(t *testing.T) {
	got, err := parseRulesJSON(`{"netByNetRules":[{"netA":"X","netB":"Y","clearance":12}]}`, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0]["netA"] != "X" {
		t.Errorf("expected netA=X, got %+v", got)
	}
}

func TestParseRulesJSON_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`[{"regionId":"r1","clearance":20}]`), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	got, err := parseRulesJSON("", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0]["regionId"] != "r1" {
		t.Errorf("expected regionId=r1, got %+v", got)
	}
}

func TestParseRulesJSON_FileMissing(t *testing.T) {
	if _, err := parseRulesJSON("", "nope/nope.json"); err == nil {
		t.Errorf("expected an error for a missing file, got nil")
	}
}

func TestParseRulesJSON_BothEmpty(t *testing.T) {
	if _, err := parseRulesJSON("", ""); err == nil {
		t.Errorf("expected an error when both inline and file are empty")
	}
}

func TestParseRulesJSON_BothSet(t *testing.T) {
	if _, err := parseRulesJSON(`[]`, "x.json"); err == nil {
		t.Errorf("expected an error when both --rules and --file are given")
	}
}

func TestParseRulesJSON_NotJSON(t *testing.T) {
	if _, err := parseRulesJSON(`not json at all`, ""); err == nil {
		t.Errorf("expected an error for invalid JSON")
	}
}

func TestParseRulesJSON_ObjectWithoutArray(t *testing.T) {
	if _, err := parseRulesJSON(`{"foo":"bar","baz":42}`, ""); err == nil {
		t.Errorf("expected an error for an object with no array field")
	}
}

// ─── buildNetRulesSetPayload ─────────────────────────────────────────

func TestBuildNetRulesSetPayload_ReplaceInline(t *testing.T) {
	got, err := buildNetRulesSetPayload("replace", `[{"net":"A","trackWidth":10}]`, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "replace" {
		t.Errorf("mode = %v, want replace", got["mode"])
	}
	arr, ok := got["netRules"].([]map[string]any)
	if !ok {
		t.Fatalf("netRules should be []map[string]any, got %T", got["netRules"])
	}
	if len(arr) != 1 || arr[0]["net"] != "A" {
		t.Errorf("unexpected netRules: %+v", arr)
	}
	if _, hasUpserts := got["upserts"]; hasUpserts {
		t.Errorf("replace mode must not set 'upserts'")
	}
}

func TestBuildNetRulesSetPayload_MergeUpserts(t *testing.T) {
	got, err := buildNetRulesSetPayload("merge", `[{"net":"A","trackWidth":12}]`, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "merge" {
		t.Errorf("mode = %v, want merge", got["mode"])
	}
	if _, hasReplace := got["netRules"]; hasReplace {
		t.Errorf("merge mode must not set 'netRules' (replace key)")
	}
	if _, ok := got["upserts"].([]map[string]any); !ok {
		t.Fatalf("upserts should be []map[string]any, got %T", got["upserts"])
	}
}

func TestBuildNetRulesSetPayload_RemoveNetsOnly(t *testing.T) {
	got, err := buildNetRulesSetPayload("merge", "", "", nil, []string{"A", "B"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "merge" {
		t.Errorf("mode = %v, want merge", got["mode"])
	}
	rn, ok := got["removeNets"].([]string)
	if !ok {
		t.Fatalf("removeNets should be []string, got %T", got["removeNets"])
	}
	if want := []string{"A", "B"}; !reflect.DeepEqual(rn, want) {
		t.Errorf("removeNets = %v, want %v", rn, want)
	}
}

func TestBuildNetRulesSetPayload_RemoveNetsWithUpserts(t *testing.T) {
	// merge + --rules + --remove-nets → both fields present.
	got, err := buildNetRulesSetPayload("merge", `[{"net":"C"}]`, "", nil, []string{"A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["upserts"]; !ok {
		t.Errorf("expected upserts to be set")
	}
	if rn, _ := got["removeNets"].([]string); len(rn) != 1 || rn[0] != "A" {
		t.Errorf("removeNets = %v, want [A]", got["removeNets"])
	}
}

func TestBuildNetRulesSetPayload_PatchesShape(t *testing.T) {
	patch := map[string]any{"net": "A", "patch": map[string]any{"trackWidth": 12.0}}
	got, err := buildNetRulesSetPayload("merge", "", "", []map[string]any{patch}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "merge" {
		t.Errorf("mode = %v, want merge", got["mode"])
	}
	ps, ok := got["patches"].([]map[string]any)
	if !ok {
		t.Fatalf("patches should be []map[string]any, got %T", got["patches"])
	}
	if len(ps) != 1 || ps[0]["net"] != "A" {
		t.Errorf("unexpected patches: %+v", ps)
	}
}

func TestBuildNetRulesSetPayload_PatchesRejectReplace(t *testing.T) {
	patch := map[string]any{"net": "A", "patch": map[string]any{}}
	if _, err := buildNetRulesSetPayload("replace", "", "", []map[string]any{patch}, nil); err == nil {
		t.Errorf("expected error: structured flags forbid replace mode (would clobber)")
	}
}

func TestBuildNetRulesSetPayload_PatchesRejectRulesJSON(t *testing.T) {
	patch := map[string]any{"net": "A", "patch": map[string]any{}}
	if _, err := buildNetRulesSetPayload("merge", `[{"net":"A"}]`, "", []map[string]any{patch}, nil); err == nil {
		t.Errorf("expected error: structured flags conflict with --rules")
	}
}

func TestBuildNetRulesSetPayload_ReplaceRequiresRulesOrFile(t *testing.T) {
	if _, err := buildNetRulesSetPayload("replace", "", "", nil, nil); err == nil {
		t.Errorf("expected error: replace with no input")
	}
}

func TestBuildNetRulesSetPayload_MergeWithNoInput(t *testing.T) {
	if _, err := buildNetRulesSetPayload("merge", "", "", nil, nil); err == nil {
		t.Errorf("expected error: merge with no input")
	}
}

func TestBuildNetRulesSetPayload_BadMode(t *testing.T) {
	if _, err := buildNetRulesSetPayload("upsert", `[{"net":"A"}]`, "", nil, nil); err == nil {
		t.Errorf("expected error for invalid mode")
	}
}

// ─── buildNetRuleSinglePatch ─────────────────────────────────────────

func TestBuildNetRuleSinglePatch_TrackWidthOnly(t *testing.T) {
	tw := 12.0
	got, err := buildNetRuleSinglePatch("USB_DP", &tw, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["net"] != "USB_DP" {
		t.Errorf("net = %v, want USB_DP", got["net"])
	}
	patch, ok := got["patch"].(map[string]any)
	if !ok {
		t.Fatalf("patch should be map[string]any, got %T", got["patch"])
	}
	if len(patch) != 1 || patch["trackWidth"] != 12.0 {
		t.Errorf("patch = %+v, want {trackWidth:12}", patch)
	}
}

func TestBuildNetRuleSinglePatch_AllFields(t *testing.T) {
	tw, cl, vd, vg := 12.0, 8.0, 0.4, 24.0
	got, err := buildNetRuleSinglePatch("+3V3", &tw, &cl, &vd, &vg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	patch := got["patch"].(map[string]any)
	if len(patch) != 4 {
		t.Errorf("expected 4 patch fields, got %d", len(patch))
	}
}

func TestBuildNetRuleSinglePatch_NoNet(t *testing.T) {
	tw := 12.0
	if _, err := buildNetRuleSinglePatch("", &tw, nil, nil, nil); err == nil {
		t.Errorf("expected error: --net required")
	}
}

func TestBuildNetRuleSinglePatch_NoFields(t *testing.T) {
	if _, err := buildNetRuleSinglePatch("A", nil, nil, nil, nil); err == nil {
		t.Errorf("expected error: at least one field required")
	}
}

// ─── buildNetByNetRulesSetPayload ───────────────────────────────────

func TestBuildNetByNetRulesSetPayload_Replace(t *testing.T) {
	got, err := buildNetByNetRulesSetPayload("replace", `[{"netA":"X","netB":"Y","clearance":16}]`, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "replace" {
		t.Errorf("mode = %v, want replace", got["mode"])
	}
	if _, ok := got["netByNetRules"].([]map[string]any); !ok {
		t.Fatalf("netByNetRules should be []map[string]any, got %T", got["netByNetRules"])
	}
}

func TestBuildNetByNetRulesSetPayload_MergeRemovePairs(t *testing.T) {
	rp := []map[string]string{{"netA": "X", "netB": "Y"}}
	got, err := buildNetByNetRulesSetPayload("merge", "", "", nil, rp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "merge" {
		t.Errorf("mode = %v, want merge", got["mode"])
	}
	out, ok := got["removePairs"].([]any)
	if !ok {
		t.Fatalf("removePairs should be []any, got %T", got["removePairs"])
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(out))
	}
	first, _ := out[0].(map[string]string)
	if first["netA"] != "X" || first["netB"] != "Y" {
		t.Errorf("pair = %+v, want {X,Y}", first)
	}
}

func TestBuildNetByNetRulesSetPayload_BadMode(t *testing.T) {
	if _, err := buildNetByNetRulesSetPayload("patch", "", "", nil, nil); err == nil {
		t.Errorf("expected error for invalid mode")
	}
}

// ─── buildNetByNetRuleSinglePatch ───────────────────────────────────

func TestBuildNetByNetRuleSinglePatch_OK(t *testing.T) {
	cl := 16.0
	got, err := buildNetByNetRuleSinglePatch("SW", "VREF", &cl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["netA"] != "SW" || got["netB"] != "VREF" {
		t.Errorf("pair = %v/%v, want SW/VREF", got["netA"], got["netB"])
	}
	patch := got["patch"].(map[string]any)
	if patch["clearance"] != 16.0 {
		t.Errorf("clearance = %v, want 16", patch["clearance"])
	}
}

func TestBuildNetByNetRuleSinglePatch_SameNet(t *testing.T) {
	cl := 16.0
	if _, err := buildNetByNetRuleSinglePatch("X", "X", &cl); err == nil {
		t.Errorf("expected error: --net-a and --net-b must differ")
	}
}

func TestBuildNetByNetRuleSinglePatch_NoClearance(t *testing.T) {
	if _, err := buildNetByNetRuleSinglePatch("X", "Y", nil); err == nil {
		t.Errorf("expected error: --clearance required")
	}
}

// ─── buildRegionRulesSetPayload ─────────────────────────────────────

func TestBuildRegionRulesSetPayload_Replace(t *testing.T) {
	got, err := buildRegionRulesSetPayload("replace", `[{"regionId":"r1","clearance":20}]`, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "replace" {
		t.Errorf("mode = %v, want replace", got["mode"])
	}
	if _, ok := got["regionRules"].([]map[string]any); !ok {
		t.Fatalf("regionRules should be []map[string]any, got %T", got["regionRules"])
	}
}

func TestBuildRegionRulesSetPayload_MergeRemoveIds(t *testing.T) {
	got, err := buildRegionRulesSetPayload("merge", "", "", nil, []string{"r1", "r2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mode"] != "merge" {
		t.Errorf("mode = %v, want merge", got["mode"])
	}
	ri, _ := got["removeIds"].([]string)
	if !reflect.DeepEqual(ri, []string{"r1", "r2"}) {
		t.Errorf("removeIds = %v, want [r1 r2]", ri)
	}
}

func TestBuildRegionRulesSetPayload_PatchesShape(t *testing.T) {
	patch := map[string]any{"regionId": "r1", "patch": map[string]any{"clearance": 20.0}}
	got, err := buildRegionRulesSetPayload("merge", "", "", []map[string]any{patch}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps, _ := got["patches"].([]map[string]any); len(ps) != 1 {
		t.Errorf("expected 1 patch, got %d", len(ps))
	}
}

// ─── buildRegionRuleSinglePatch ─────────────────────────────────────

func TestBuildRegionRuleSinglePatch_AllFields(t *testing.T) {
	cl, tw, vd, vg := 20.0, 8.0, 0.4, 24.0
	got, err := buildRegionRuleSinglePatch("rf_zone", &cl, &tw, &vd, &vg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["regionId"] != "rf_zone" {
		t.Errorf("regionId = %v, want rf_zone", got["regionId"])
	}
	patch := got["patch"].(map[string]any)
	if len(patch) != 4 {
		t.Errorf("expected 4 patch fields, got %d", len(patch))
	}
}

func TestBuildRegionRuleSinglePatch_NoRegionId(t *testing.T) {
	cl := 20.0
	if _, err := buildRegionRuleSinglePatch("", &cl, nil, nil, nil); err == nil {
		t.Errorf("expected error: --region-id required")
	}
}

func TestBuildRegionRuleSinglePatch_NoFields(t *testing.T) {
	if _, err := buildRegionRuleSinglePatch("r1", nil, nil, nil, nil); err == nil {
		t.Errorf("expected error: at least one field required")
	}
}

// ─── f64Ptr ─────────────────────────────────────────────────────────

func TestF64Ptr_SetTrue(t *testing.T) {
	p := f64Ptr(12.0, true)
	if p == nil {
		t.Fatalf("expected non-nil pointer when set=true")
	}
	if *p != 12.0 {
		t.Errorf("value = %v, want 12", *p)
	}
}

func TestF64Ptr_SetFalse(t *testing.T) {
	if p := f64Ptr(12.0, false); p != nil {
		t.Errorf("expected nil when set=false, got %v", *p)
	}
}
