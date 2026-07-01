package app

// pcb_shortroute_avoid.go — obstacle-aware routing for route-short (#23).
//
// v1 route-short blindly drew a horizontal-first L for every hop, so tracks crossed
// each other and cut through other nets' pads (27 Safe-Spacing violations on ceshi).
// This picks, per hop, the L ORIENTATION (horizontal-first vs vertical-first) that
// hits the fewest OTHER-NET obstacles — already-placed segments it would cross, and
// other-net pads it would run too close to. Greedy + order-dependent (a hop only
// sees hops planned before it), but it removes most of the naive tangle at ~zero
// cost. NOT a maze router: no push-shove, no vias, no rip-up — a hop with no clean
// orientation still emits its lower-cost L (and DRC/the maze tier handles the rest).

import "math"

// obPad is a pad as an obstacle: its net (a same-net pad is never an obstacle) and
// center (mil). Pads have no size in the router's input, so proximity is judged
// against the safe-spacing clearance plus a nominal pad half-extent.
type obPad struct {
	net  string
	x, y float64
}

// nominalPadHalf is a stand-in pad half-extent (mil) added to the clearance when
// judging "a track runs too close to a pad" — real pad sizes aren't in the router
// input, and a typical 0402/SMD pad is ~15-25mil across.
const nominalPadHalf = 12

// routeWithAvoid returns the hop's segments in whichever L orientation hits fewer
// other-net obstacles. With opt.avoid off, or a straight (aligned) hop, it's just
// the default horizontal-first route.
func routeWithAvoid(net string, a, b rtPad, w float64, opt rtOptions, placed []rtSeg, obstacles []obPad) []rtSeg {
	if !opt.avoid || a.x == b.x || a.y == b.y {
		return routeHop(net, a, b, w, opt, true)
	}
	h := routeHop(net, a, b, w, opt, true)  // horizontal-first
	v := routeHop(net, a, b, w, opt, false) // vertical-first
	clr := opt.clearance + nominalPadHalf
	if hopCost(v, net, a, b, placed, obstacles, clr) < hopCost(h, net, a, b, placed, obstacles, clr) {
		return v
	}
	return h // ties keep the conventional horizontal-first
}

// hopCost scores a candidate route by how many other-net obstacles it hits: each
// crossing with an already-placed other-net segment is heavily weighted (a hard
// short-in-waiting); each other-net pad within clr of a segment is a lighter
// penalty. This hop's own endpoint pads (a, b) are never counted.
func hopCost(cand []rtSeg, net string, a, b rtPad, placed []rtSeg, obstacles []obPad, clr float64) int {
	cost := 0
	for _, s := range cand {
		for _, p := range placed {
			if p.Net == net {
				continue
			}
			if segSegCross(s.X1, s.Y1, s.X2, s.Y2, p.X1, p.Y1, p.X2, p.Y2) {
				cost += 10
			}
		}
		for _, pd := range obstacles {
			if pd.net == net || pd.net == "" {
				continue
			}
			if samePoint(pd.x, pd.y, a.x, a.y) || samePoint(pd.x, pd.y, b.x, b.y) {
				continue // this hop's own endpoints
			}
			if segPointDist(s.X1, s.Y1, s.X2, s.Y2, pd.x, pd.y) < clr {
				cost += 4
			}
		}
	}
	return cost
}

func samePoint(ax, ay, bx, by float64) bool {
	return math.Abs(ax-bx) < 0.01 && math.Abs(ay-by) < 0.01
}

// segSegCross reports whether two segments properly cross (interior intersection).
// Shared/near endpoints do NOT count.
func segSegCross(ax, ay, bx, by, cx, cy, dx, dy float64) bool {
	d := (bx-ax)*(dy-cy) - (by-ay)*(dx-cx)
	if math.Abs(d) < 1e-9 {
		return false // parallel / collinear
	}
	t := ((cx-ax)*(dy-cy) - (cy-ay)*(dx-cx)) / d
	u := ((cx-ax)*(by-ay) - (cy-ay)*(bx-ax)) / d
	const eps = 1e-6
	return t > eps && t < 1-eps && u > eps && u < 1-eps
}

// segPointDist is the shortest distance from point (px,py) to segment (ax,ay)-(bx,by).
func segPointDist(ax, ay, bx, by, px, py float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / l2
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}
