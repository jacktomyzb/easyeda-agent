package app

import "testing"

func TestSegSegCross(t *testing.T) {
	// Proper X crossing.
	if !segSegCross(0, 0, 10, 10, 0, 10, 10, 0) {
		t.Error("expected proper crossing at (5,5)")
	}
	// Shared endpoint is NOT a crossing.
	if segSegCross(0, 0, 10, 0, 10, 0, 10, 10) {
		t.Error("shared endpoint must not count as a crossing")
	}
	// Parallel, no crossing.
	if segSegCross(0, 0, 10, 0, 0, 5, 10, 5) {
		t.Error("parallel segments must not cross")
	}
}

// A placed vertical track at x=5 (net P) sits between the hop's endpoints. The
// horizontal-first L would cross it at (5,10); the vertical-first L skirts it.
// routeWithAvoid must pick vertical-first.
func TestRouteWithAvoid_PicksClearOrientation(t *testing.T) {
	placed := []rtSeg{{Net: "P", X1: 5, Y1: 0, X2: 5, Y2: 20}}
	a := rtPad{comp: "U1", pin: "1", x: 0, y: 10, layer: 1}
	b := rtPad{comp: "U2", pin: "1", x: 10, y: 0, layer: 1}
	opt := defaultRtOptions()
	opt.corner = "90"

	got := routeWithAvoid("S", a, b, 10, opt, placed, nil, nil)
	if len(got) == 0 {
		t.Fatal("no segments returned")
	}
	// Vertical-first ⇒ the first segment runs from (0,10) straight down to (0,0).
	first := got[0]
	if !(first.X1 == 0 && first.Y1 == 10 && first.X2 == 0 && first.Y2 == 0) {
		t.Errorf("expected vertical-first (0,10)->(0,0), got (%.0f,%.0f)->(%.0f,%.0f)", first.X1, first.Y1, first.X2, first.Y2)
	}

	// With avoidance OFF, it reverts to the naive horizontal-first L (corner at 10,10).
	opt.avoid = false
	naive := routeWithAvoid("S", a, b, 10, opt, placed, nil, nil)
	if naive[0].X2 != 10 || naive[0].Y2 != 10 {
		t.Errorf("no-avoid should be horizontal-first (corner 10,10), got corner (%.0f,%.0f)", naive[0].X2, naive[0].Y2)
	}
}

// A hop should avoid running through another net's pad.
func TestHopCost_CountsOtherNetPad(t *testing.T) {
	a := rtPad{x: 0, y: 0, layer: 1}
	b := rtPad{x: 10, y: 10, layer: 1}
	cand := lShape90("S", a, b, 10, true) // (0,0)->(10,0)->(10,10)
	// A P-net pad sitting on the horizontal leg at (5,0), same layer as the track.
	obst := []obPad{{net: "P", x: 5, y: 0, layer: 1}}
	if c := hopCost(cand, "S", a, b, nil, obst, nil, 6); c == 0 {
		t.Error("expected non-zero cost for a track running over another net's pad")
	}
	// The SAME pad on net S (the hop's own net) is not an obstacle.
	if c := hopCost(cand, "S", a, b, nil, []obPad{{net: "S", x: 5, y: 0, layer: 1}}, nil, 6); c != 0 {
		t.Errorf("same-net pad must not add cost, got %d", c)
	}
	// A P-net pad on a DIFFERENT layer than the track adds no cost (layer-aware).
	if c := hopCost(cand, "S", a, b, nil, []obPad{{net: "P", x: 5, y: 0, layer: 2}}, nil, 6); c != 0 {
		t.Errorf("other-layer pad must not add cost, got %d", c)
	}
	// A P-net VIA on the horizontal leg adds cost on any layer.
	if c := hopCost(cand, "S", a, b, nil, nil, []obVia{{net: "P", x: 5, y: 0, r: 12}}, 6); c == 0 {
		t.Error("expected non-zero cost for a track running over another net's via")
	}
}
