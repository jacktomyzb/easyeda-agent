package app

// idlist.go — one shared parser for every delete-by-id CLI flag (issue #109).
// History: `fill delete` / `pour-delete` / `region delete` / `pcb delete` took a
// JSON array payload while `track-delete` / `via-delete` took CSV — the same
// agent flip-flopped between the two formats mid-session and got it wrong both
// ways. parseIDList accepts BOTH: a JSON array of strings is tried first (input
// starting with '['), anything else is split as CSV with per-item trimming.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// parseIDList normalizes an --ids flag value into a []string of primitive ids.
// Accepted forms:
//
//	CSV:        id1,id2       (spaces around commas ok, empty items dropped)
//	JSON array: ["id1","id2"] (numbers tolerated and stringified)
//
// An input that LOOKS like JSON (leading '[') but fails to parse is an error —
// it is never silently re-interpreted as CSV, so a typo'd JSON array cannot
// half-delete the wrong ids.
func parseIDList(s string) ([]string, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil, fmt.Errorf("empty id list — pass CSV (id1,id2) or a JSON array ([\"id1\",\"id2\"])")
	}
	if strings.HasPrefix(t, "[") {
		var raw []any
		if err := json.Unmarshal([]byte(t), &raw); err != nil {
			return nil, fmt.Errorf("--ids looks like a JSON array but does not parse (expected [\"id1\",\"id2\"]): %w", err)
		}
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			switch x := v.(type) {
			case string:
				if id := strings.TrimSpace(x); id != "" {
					out = append(out, id)
				}
			case float64:
				out = append(out, strconv.FormatFloat(x, 'f', -1, 64))
			default:
				return nil, fmt.Errorf("--ids JSON array items must be strings, got %T (%v)", v, v)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("--ids JSON array contains no ids")
		}
		return out, nil
	}
	var out []string
	for _, p := range strings.Split(t, ",") {
		if id := strings.TrimSpace(p); id != "" {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ids found in %q — pass CSV (id1,id2) or a JSON array", s)
	}
	return out, nil
}
