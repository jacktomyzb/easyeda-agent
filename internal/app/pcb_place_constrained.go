package app

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

// pcb_place_constrained.go — constraint-driven TIERED placement (daemon-side).
//
// The fix for whack-a-mole layout: place POSITION-CONSTRAINED parts first and
// LOCK them, then legalize the rest around the locked set — so a satellite pass
// can never push an edge connector off its edge. Tiers (highest priority first):
//
//	Tier 1  mounting holes            — passed in as obstacles (placed via `pcb slot`), never moved.
//	Tier 2  edge-constrained parts    — connectors (USB / terminal / card socket / IPEX) + RF
//	                                     modules → snapped flush to their NEAREST board edge, fixed.
//	Tier 3  main chips + crystals     — anchors, kept where they are (fixed).
//	Tier 4  satellites + user-facing  — legalized (spiral) around the fixed set, avoiding holes.
//
// Classification is by footprint/designator pattern — the categories are the ones
// the circuit-block library flags with `placement.board_edge=true` / `user-facing`
// (see internal/blocks/data/*.json). A board that was block-assembled or built from
// the schematic both work: we read what's placed, not how it got there.

type cpClass int

const (
	cpSatellite  cpClass = iota // tier 4 (default)
	cpUserFacing                // tier 4, but wants to stay near an edge / visible (LED, button)
	cpMainChip                  // tier 3
	cpEdgeMust                  // tier 2 — MUST sit at a board edge (connector / module / IPEX)
)

func (k cpClass) String() string {
	switch k {
	case cpEdgeMust:
		return "edge"
	case cpMainChip:
		return "main"
	case cpUserFacing:
		return "user-facing"
	default:
		return "satellite"
	}
}

// Footprint / designator patterns for the position-constrained categories. Derived
// from the block library's placement hints (must-edge = connectors + RF module;
// user-facing = buttons + LED). Matched case-insensitively against footprint name.
var (
	cpReEdgeConn = regexp.MustCompile(`(?i)usb|type-?c|micro-?sd|tf[-_ ]?card|sd[-_]?card|push-?push|ipex|u\.?fl|sma|ufl|kf301|kf128|kf2edg|terminal|screw|hdr|header|pin-?header|conn`)
	cpReModule   = regexp.MustCompile(`(?i)wroom|wrover|esp32.*(module|wifi|smd)`)
	cpReSwitch   = regexp.MustCompile(`(?i)tact|switch|\bkey\b|button|sw-?smd`)
	cpReLED      = regexp.MustCompile(`(?i)\bled\b`)
	cpReCrystal  = regexp.MustCompile(`(?i)xtal|crystal|osc|3225|3215|2016|2520|smd-?4p`)
)

// cpComp carries a placed component's device NAME (PCB components expose no
// footprint-name string, so we pattern-match the device name — e.g.
// "esp32-s3-wroom-1u-n8" — for module detection; connectors are caught by the
// J* designator prefix anyway) + copper layer (TOP=1 / BOTTOM=2).
type cpComp struct {
	apComp
	footprint string // device name, used for module/connector pattern matching
	layer     int
}

// classify decides the placement tier from footprint + designator + pin count.
func classifyCP(c cpComp, mainPins int) cpClass {
	fp := c.footprint
	des := strings.ToUpper(c.designator)
	// A connector/module footprint, OR a Jxx designator that isn't a plain header
	// resistor — treat as edge-must.
	if cpReModule.MatchString(fp) {
		return cpEdgeMust
	}
	// Jxx connectors are edge-must, but NOT JPxx (a jumper/link — belongs by its net,
	// not at a board edge). A J-prefix with a connector-ish footprint also qualifies.
	if cpReEdgeConn.MatchString(fp) || (strings.HasPrefix(des, "J") && !strings.HasPrefix(des, "JP")) {
		return cpEdgeMust
	}
	if cpReSwitch.MatchString(fp) || strings.HasPrefix(des, "SW") {
		return cpUserFacing
	}
	if cpReLED.MatchString(fp) || strings.HasPrefix(des, "LED") {
		return cpUserFacing
	}
	// Main chip by distinct-pin count (crystals are few-pin but anchor near their IC
	// — fold them into main so they stay put in tier 3).
	if c.distinctPins() >= mainPins || cpReCrystal.MatchString(fp) {
		return cpMainChip
	}
	return cpSatellite
}

type cpHole struct{ x, y, r float64 }

type cpOptions struct {
	mainPins   int
	edgeMargin float64 // gap between an edge part's bbox and the board edge
	partGap    float64 // clearance between any two parts / part-to-hole
}

func defaultCpOptions() cpOptions {
	return cpOptions{mainPins: 8, edgeMargin: 45, partGap: 14}
}

type cpRect struct{ x0, y0, x1, y1 float64 }

func (r cpRect) overlaps(o cpRect) bool {
	return !(r.x1 <= o.x0 || o.x1 <= r.x0 || r.y1 <= o.y0 || o.y1 <= r.y0)
}

// planConstrainedPlace runs the tiered placement over a snapshot of components +
// mounting holes, returning the anchor moves. Pure: no I/O, so it unit-tests.
func planConstrainedPlace(comps []cpComp, holes []cpHole, opt cpOptions) ([]apMove, []apDiag) {
	var moves []apMove
	var diags []apDiag
	if len(comps) == 0 {
		return moves, diags
	}
	// Board rect = current placed extent (outline-fit already sized it).
	bx0, by0 := math.Inf(1), math.Inf(1)
	bx1, by1 := math.Inf(-1), math.Inf(-1)
	for _, c := range comps {
		if !c.hasBBox {
			continue
		}
		bx0, by0 = math.Min(bx0, c.minX), math.Min(by0, c.minY)
		bx1, by1 = math.Max(bx1, c.maxX), math.Max(by1, c.maxY)
	}
	m := opt.partGap
	// placed holds the FIXED rects (edge parts + mains + holes) satellites avoid.
	var placed []cpRect
	for _, h := range holes {
		placed = append(placed, cpRect{h.x - h.r, h.y - h.r, h.x + h.r, h.y + h.r})
	}
	// Layer-aware: a satellite only clashes with same-layer fixed parts. Track layer per rect.
	type lrect struct {
		cpRect
		layer int
	}
	var lplaced []lrect
	addFixed := func(r cpRect, layer int) {
		placed = append(placed, r)
		lplaced = append(lplaced, lrect{r, layer})
	}

	// Classify.
	kinds := make([]cpClass, len(comps))
	for i, c := range comps {
		kinds[i] = classifyCP(c, opt.mainPins)
	}

	// ── Tier 2: edge-must → snap to nearest board edge, fix. ──────────────────
	// Order: biggest first (big connectors claim edge space before small ones).
	edgeIdx := []int{}
	for i := range comps {
		if kinds[i] == cpEdgeMust {
			edgeIdx = append(edgeIdx, i)
		}
	}
	sort.Slice(edgeIdx, func(a, b int) bool {
		ca, cb := comps[edgeIdx[a]], comps[edgeIdx[b]]
		return ca.width()*ca.height() > cb.width()*cb.height()
	})
	for _, i := range edgeIdx {
		c := comps[i]
		if !c.hasBBox {
			continue
		}
		cx, cy := c.bboxCenter()
		// nearest edge by bbox-center distance
		dL, dR, dB, dT := cx-bx0, bx1-cx, cy-by0, by1-cy
		best := dL
		edge := edgeLeft
		if dR < best {
			best, edge = dR, edgeRight
		}
		if dB < best {
			best, edge = dB, edgeBottom
		}
		if dT < best {
			best, edge = dT, edgeTop
		}
		var nx, ny float64 = c.x, c.y
		switch edge {
		case edgeLeft:
			nx = c.x + (bx0 + opt.edgeMargin - c.minX)
		case edgeRight:
			nx = c.x + (bx1 - opt.edgeMargin - c.maxX)
		case edgeBottom:
			ny = c.y + (by0 + opt.edgeMargin - c.minY)
		case edgeTop:
			ny = c.y + (by1 - opt.edgeMargin - c.maxY)
		}
		// New bbox after the snap.
		dx, dy := nx-c.x, ny-c.y
		nr := cpRect{c.minX + dx - m, c.minY + dy - m, c.maxX + dx + m, c.maxY + dy + m}
		addFixed(nr, c.layer)
		if math.Abs(dx) > 1 || math.Abs(dy) > 1 {
			moves = append(moves, apMove{ID: c.id, Designator: c.designator, NewX: round1(nx), NewY: round1(ny), Edge: edge.String()})
		}
		diags = append(diags, apDiag{Designator: c.designator, Reason: "edge:" + edge.String()})
	}

	// ── Tier 3: main chips + crystals → keep where they are, fix. ─────────────
	for i, c := range comps {
		if kinds[i] != cpMainChip || !c.hasBBox {
			continue
		}
		addFixed(cpRect{c.minX - m, c.minY - m, c.maxX + m, c.maxY + m}, c.layer)
		diags = append(diags, apDiag{Designator: c.designator, Reason: "main:fixed"})
	}

	// ── Tier 4: satellites + user-facing → legalize (spiral) around fixed. ────
	satIdx := []int{}
	for i := range comps {
		if kinds[i] == cpSatellite || kinds[i] == cpUserFacing {
			satIdx = append(satIdx, i)
		}
	}
	// Biggest satellites first (they need the most room).
	sort.Slice(satIdx, func(a, b int) bool {
		ca, cb := comps[satIdx[a]], comps[satIdx[b]]
		return ca.width()*ca.height() > cb.width()*cb.height()
	})
	clashFixed := func(r cpRect, layer int) bool {
		for _, h := range holes { // holes cut every layer
			if (cpRect{h.x - h.r, h.y - h.r, h.x + h.r, h.y + h.r}).overlaps(r) {
				return true
			}
		}
		for _, lr := range lplaced {
			if lr.layer == layer && lr.cpRect.overlaps(r) {
				return true
			}
		}
		return false
	}
	for _, i := range satIdx {
		c := comps[i]
		if !c.hasBBox {
			continue
		}
		cx0, cy0 := c.bboxCenter()
		hw, hh := c.width()/2, c.height()/2
		var best *[2]float64
		for rad := 0.0; rad <= 2200 && best == nil; rad += 25 {
			steps := 1
			if rad > 0 {
				steps = 24
			}
			for s := 0; s < steps; s++ {
				ang := float64(s) * math.Pi / 12
				px, py := cx0+rad*math.Cos(ang), cy0+rad*math.Sin(ang)
				r := cpRect{px - hw - m, py - hh - m, px + hw + m, py + hh + m}
				if r.x0 < bx0-20 || r.y0 < by0-20 || r.x1 > bx1+20 || r.y1 > by1+20 {
					continue
				}
				if !clashFixed(r, c.layer) {
					best = &[2]float64{px, py}
					break
				}
			}
		}
		if best == nil {
			diags = append(diags, apDiag{Designator: c.designator, Reason: "satellite:no-fit"})
			continue
		}
		px, py := best[0], best[1]
		addFixed(cpRect{px - hw - m, py - hh - m, px + hw + m, py + hh + m}, c.layer)
		dx, dy := px-cx0, py-cy0
		if math.Abs(dx) > 1 || math.Abs(dy) > 1 {
			moves = append(moves, apMove{ID: c.id, Designator: c.designator, NewX: round1(c.x + dx), NewY: round1(c.y + dy), Edge: kinds[i].String()})
		}
	}
	return moves, diags
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// parseCpComps parses pcb.components.list into cpComps (apComp + device name +
// layer). A PCB component's identifying string is its device `name` (no footprint
// name is exposed); the layer is TOP=1 / BOTTOM=2.
func parseCpComps(result map[string]any) []cpComp {
	base := parseApComps(result)
	byID := map[string]cpComp{}
	raw, _ := result["components"].([]any)
	for _, ri := range raw {
		cm, ok := ri.(map[string]any)
		if !ok {
			continue
		}
		id := asString(cm["primitiveId"])
		layer := int(asFloat(cm["layer"]))
		if layer == 0 {
			layer = 1
		}
		byID[id] = cpComp{footprint: asString(cm["name"]), layer: layer}
	}
	out := make([]cpComp, 0, len(base))
	for _, b := range base {
		extra := byID[b.id]
		out = append(out, cpComp{apComp: b, footprint: extra.footprint, layer: extra.layer})
	}
	return out
}

// readCpHoles reads mounting-hole cutouts (fills on the MULTI layer, id 12 — where
// `pcb slot` puts board cutouts) and reduces each to a center + radius obstacle.
// Best-effort: a fill without readable points is skipped.
func readCpHoles(cfg *appConfig, window string) []cpHole {
	res, err := requestAction(cfg, "pcb.fill.list", window, nil)
	if err != nil || res == nil {
		return nil
	}
	fills, _ := res.Result["fills"].([]any)
	var out []cpHole
	for _, fi := range fills {
		fm, ok := fi.(map[string]any)
		if !ok || int(asFloat(fm["layer"])) != 12 {
			continue
		}
		pts, _ := fm["points"].([]any)
		if len(pts) < 3 {
			continue
		}
		minX, minY := math.Inf(1), math.Inf(1)
		maxX, maxY := math.Inf(-1), math.Inf(-1)
		for _, pi := range pts {
			p, ok := pi.([]any)
			if !ok || len(p) < 2 {
				continue
			}
			x, y := asFloat(p[0]), asFloat(p[1])
			minX, minY = math.Min(minX, x), math.Min(minY, y)
			maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
		}
		if math.IsInf(minX, 1) {
			continue
		}
		cx, cy := (minX+maxX)/2, (minY+maxY)/2
		// clearance radius = hole radius + washer margin (M3 head ≈ R118 mil)
		r := math.Max((maxX-minX)/2, (maxY-minY)/2) + 60
		out = append(out, cpHole{x: cx, y: cy, r: r})
	}
	return out
}
