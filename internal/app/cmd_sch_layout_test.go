package app

import "testing"

func bb(minX, minY, maxX, maxY float64) *layoutBBox {
	return &layoutBBox{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
}

func TestAnalyzeLayout_Overlap(t *testing.T) {
	comps := []layoutComp{
		{Designator: "R1", BBox: bb(0, 0, 10, 10)},
		{Designator: "C2", BBox: bb(5, 5, 15, 15)}, // overlaps R1 by 5×5
	}
	rep := analyzeLayout(comps, 2.54)
	if rep.OK {
		t.Fatal("expected OK=false when components overlap")
	}
	if len(rep.Overlaps) != 1 {
		t.Fatalf("expected 1 overlap, got %d", len(rep.Overlaps))
	}
	f := rep.Overlaps[0]
	if f.A != "C2" || f.B != "R1" { // labels sorted
		t.Errorf("expected pair C2↔R1, got %s↔%s", f.A, f.B)
	}
	if f.OvX != 5 || f.OvY != 5 {
		t.Errorf("expected overlap 5×5, got %.2f×%.2f", f.OvX, f.OvY)
	}
}

func TestAnalyzeLayout_TightSpacing(t *testing.T) {
	comps := []layoutComp{
		{Designator: "U1", BBox: bb(0, 0, 10, 10)},
		{Designator: "C5", BBox: bb(11, 0, 21, 10)}, // 1mm gap horizontally
	}
	rep := analyzeLayout(comps, 2.54)
	if !rep.OK {
		t.Fatal("tight spacing alone should not fail OK (only overlaps do)")
	}
	if len(rep.TightPairs) != 1 {
		t.Fatalf("expected 1 tight pair, got %d", len(rep.TightPairs))
	}
	if g := rep.TightPairs[0].Gap; g != 1 {
		t.Errorf("expected gap 1.0mm, got %.2f", g)
	}
}

func TestAnalyzeLayout_Clear(t *testing.T) {
	comps := []layoutComp{
		{Designator: "U1", BBox: bb(0, 0, 10, 10)},
		{Designator: "C5", BBox: bb(20, 0, 30, 10)}, // 10mm gap, well clear
	}
	rep := analyzeLayout(comps, 2.54)
	if !rep.OK || len(rep.Overlaps) != 0 || len(rep.TightPairs) != 0 {
		t.Fatalf("expected clean report, got %+v", rep)
	}
}

func TestAnalyzeLayout_TouchingEdgesNotOverlap(t *testing.T) {
	comps := []layoutComp{
		{Designator: "A", BBox: bb(0, 0, 10, 10)},
		{Designator: "B", BBox: bb(10, 0, 20, 10)}, // shares an edge, gap 0
	}
	rep := analyzeLayout(comps, 2.54)
	if len(rep.Overlaps) != 0 {
		t.Fatalf("touching edges must not count as overlap, got %d", len(rep.Overlaps))
	}
	if len(rep.TightPairs) != 1 || rep.TightPairs[0].Gap != 0 {
		t.Fatalf("expected one tight pair at gap 0, got %+v", rep.TightPairs)
	}
}

func TestAnalyzeLayout_UnassignedDesignatorFallsBackToID(t *testing.T) {
	comps := []layoutComp{
		{ID: "aaa111", Designator: "C?", BBox: bb(0, 0, 10, 10)},
		{ID: "bbb222", Designator: "C?", BBox: bb(5, 5, 15, 15)}, // overlap
	}
	rep := analyzeLayout(comps, 2.54)
	if len(rep.Overlaps) != 1 {
		t.Fatalf("expected 1 overlap, got %d", len(rep.Overlaps))
	}
	f := rep.Overlaps[0]
	// Both designators are unassigned ("C?") → labels disambiguate via id.
	if f.A == f.B {
		t.Fatalf("unassigned designators must disambiguate, got %q ↔ %q", f.A, f.B)
	}
	if f.A != "C?@aaa111" || f.B != "C?@bbb222" {
		t.Errorf("expected id-suffixed labels, got %q ↔ %q", f.A, f.B)
	}
}

func TestAnalyzeLayout_NoBBoxSkipped(t *testing.T) {
	comps := []layoutComp{
		{Designator: "R1", BBox: bb(0, 0, 10, 10)},
		{Designator: "R2"}, // no bbox → skipped, recorded
	}
	rep := analyzeLayout(comps, 2.54)
	if rep.WithBBox != 1 {
		t.Errorf("expected WithBBox=1, got %d", rep.WithBBox)
	}
	if len(rep.NoBBox) != 1 || rep.NoBBox[0] != "R2" {
		t.Errorf("expected R2 recorded as no-bbox, got %v", rep.NoBBox)
	}
}
