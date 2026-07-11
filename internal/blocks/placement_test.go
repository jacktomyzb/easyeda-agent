package blocks

import "testing"

func TestLoadPlacementIndex(t *testing.T) {
	idx, err := LoadPlacementIndex()
	if err != nil {
		t.Fatalf("LoadPlacementIndex: %v", err)
	}

	// JP701's part (conn.sip2_254) is declared board_edge=false in the RS485
	// block — the exact misfire issue #95 fixes.
	jp, ok := idx.ByDevice["conn.sip2_254"]
	if !ok {
		t.Fatal("conn.sip2_254 (JP701) missing from ByDevice index")
	}
	if jp.BoardEdge {
		t.Errorf("JP701 part should be board_edge=false; got true")
	}

	// The 3P terminal is an edge part.
	if j4, ok := idx.ByDevice["conn.terminal_3p_508"]; !ok || !j4.BoardEdge {
		t.Errorf("conn.terminal_3p_508 (J4) should be indexed board_edge=true; ok=%v edge=%v", ok, j4.BoardEdge)
	}

	// Distinctive prefixes are indexed; the JP prefix resolves to a non-edge hint.
	if jpP, ok := idx.ByRefPrefix["JP"]; ok && jpP.BoardEdge {
		t.Errorf("JP prefix should map to a non-edge hint; got board_edge=true")
	}
	// Generic single-letter prefixes must NOT be indexed (would misfile whole
	// component classes).
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
