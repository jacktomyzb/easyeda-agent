package blocks

import (
	"strings"
	"testing"
)

func TestLoadPlacementIndex(t *testing.T) {
	idx, err := LoadPlacementIndex()
	if err != nil {
		t.Fatalf("LoadPlacementIndex: %v", err)
	}

	// JP701 (RS485 120R terminator jumper) is board_edge=false + anchor=true in
	// the RS485 block — the exact misfire issue #95 fixes. Its DISTINCTIVE "JP"
	// prefix must resolve to that anchored, non-edge hint (a placed part never
	// exposes the block role-id, so the prefix is what the placer matches on).
	jp, ok := idx.ByRefPrefix["JP"]
	if !ok {
		t.Fatal("JP prefix (JP701) missing from ByRefPrefix index")
	}
	if jp.BoardEdge {
		t.Errorf("JP701 hint should be board_edge=false; got true")
	}
	if !jp.Anchor {
		t.Errorf("JP701 hint should be anchor=true (deliberate non-edge); got false")
	}

	// ANT (RF u.FL) is a board-edge part; SW (tactile buttons) are user-facing.
	if ant, ok := idx.ByRefPrefix["ANT"]; !ok || !ant.BoardEdge {
		t.Errorf("ANT prefix should be indexed board_edge=true; ok=%v edge=%v", ok, ant.BoardEdge)
	}
	if sw, ok := idx.ByRefPrefix["SW"]; !ok || !strings.EqualFold(strings.TrimSpace(sw.Edge), "user-facing") {
		t.Errorf("SW prefix should be user-facing; ok=%v edge=%q", ok, sw.Edge)
	}

	// Generic single-letter prefixes must NOT be indexed (would misfile whole
	// component classes — e.g. snapping every U* IC to a board edge).
	for _, p := range []string{"U", "C", "R", "X", "J"} {
		if _, ok := idx.ByRefPrefix[p]; ok {
			t.Errorf("generic prefix %q must not be in ByRefPrefix", p)
		}
	}
}

func TestRefPrefix(t *testing.T) {
	cases := map[string]string{"JP701": "JP", "J4": "J", "SW_BOOT": "SW", "LED1": "LED", "701": "", "": ""}
	for in, want := range cases {
		if got := refPrefix(in); got != want {
			t.Errorf("refPrefix(%q)=%q, want %q", in, got, want)
		}
	}
}
