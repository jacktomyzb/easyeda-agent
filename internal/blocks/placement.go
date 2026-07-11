package blocks

import (
	"encoding/json"
	"strings"
)

// PlacementHint is a block's placement.<REF> declaration, projected for the PCB
// placer to consume. The circuit-block library (data/*.json) is the single
// source-of-truth for placement roles; the placer reverse-maps a PLACED
// component (which carries no block link) back to its declared role via the
// part device id or the designator prefix.
type PlacementHint struct {
	BoardEdge   bool   `json:"board_edge"`
	Edge        string `json:"edge"`
	Side        string `json:"side"`
	Orientation string `json:"orientation"`
	Severity    string `json:"severity"`
	Device      string `json:"-"` // library part id (parts[ref].part) this hint applies to
	Ref         string `json:"-"` // block-internal designator the hint was declared under
}

// PlacementIndex is the two-level reverse-map the placer consults before falling
// back to hardcoded regex: device-name → hint and designator-prefix → hint.
//
// ByRefPrefix is deliberately restricted to DISTINCTIVE prefixes (2+ letters
// such as JP / SW / LED / ANT). Generic single-letter prefixes (U / C / R / X)
// are excluded: on a real board they denote a whole component class, so blanket
// prefix-mapping them would misfile (e.g. snapping every U* IC to a board edge).
// Those are handled by the precise ByDevice layer or the regex fallback.
type PlacementIndex struct {
	ByDevice    map[string]PlacementHint // lower-cased library part id → hint
	ByRefPrefix map[string]PlacementHint // upper-cased alpha designator prefix → hint
}

// blockPlacementRaw mirrors just the fields of a block JSON the index needs.
type blockPlacementRaw struct {
	Parts map[string]struct {
		Part string `json:"part"`
	} `json:"parts"`
	Placement map[string]PlacementHint `json:"placement"`
}

// refPrefix returns the leading alphabetic run of a designator, upper-cased
// ("JP701" → "JP", "SW_BOOT" → "SW", "J4" → "J"). Empty if it doesn't start
// with a letter.
func refPrefix(ref string) string {
	ref = strings.ToUpper(strings.TrimSpace(ref))
	i := 0
	for i < len(ref) && ref[i] >= 'A' && ref[i] <= 'Z' {
		i++
	}
	return ref[:i]
}

// LoadPlacementIndex builds the reverse-map from every embedded block's
// placement.* declarations. It reads the raw JSON directly (not the Block
// projection, which drops `placement`) so the block library stays the sole
// source-of-truth. A malformed single block is skipped, not fatal.
func LoadPlacementIndex() (PlacementIndex, error) {
	idx := PlacementIndex{
		ByDevice:    map[string]PlacementHint{},
		ByRefPrefix: map[string]PlacementHint{},
	}
	all, err := Load()
	if err != nil {
		return idx, err
	}
	// A prefix that resolves to conflicting board_edge across blocks is
	// ambiguous → drop it, let device-name / regex decide instead.
	prefixConflict := map[string]bool{}
	for _, b := range all {
		var raw blockPlacementRaw
		if json.Unmarshal(b.Raw, &raw) != nil {
			continue
		}
		for ref, hint := range raw.Placement {
			hint.Ref = ref
			if p, ok := raw.Parts[ref]; ok {
				hint.Device = p.Part
			}
			if hint.Device != "" {
				key := strings.ToLower(hint.Device)
				if _, seen := idx.ByDevice[key]; !seen {
					idx.ByDevice[key] = hint
				}
			}
			prefix := refPrefix(ref)
			// Skip generic single-letter prefixes (see PlacementIndex doc).
			if len(prefix) < 2 || prefixConflict[prefix] {
				continue
			}
			if prev, seen := idx.ByRefPrefix[prefix]; seen {
				if prev.BoardEdge != hint.BoardEdge {
					delete(idx.ByRefPrefix, prefix)
					prefixConflict[prefix] = true
				}
				continue
			}
			idx.ByRefPrefix[prefix] = hint
		}
	}
	return idx, nil
}
