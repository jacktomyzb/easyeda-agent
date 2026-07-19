package app

// pcb_drc_rules.go — payload builders for the per-net / per-net-pair / per-region
// DRC sub-rule write actions (`pcb.drc.net_rules.set` / `net_by_net_rules.set`
// / `region_rules.set`). Kept pure (no `cfg`, no `dispatch`, no `cobra`) so they
// can be unit-tested without spinning up a CLI; cmd_pcb.go wires them to flags.
//
// The three rule kinds share the same three input shapes the connector accepts:
//   • mode=replace (default) — `netRules` / `netByNetRules` / `regionRules` array
//     (a full overwrite; read first, mutate, write back).
//   • mode=merge — `upserts` array; each entry is matched by its key (net / net
//     pair / region id) and recursively deep-merged into the existing entry,
//     appending new ones.
//   • structured single-value form — `patches` array `[{key, patch:{...}}]`;
//     the CLI's `--track-width`/`--clearance`/etc convenience flags build one
//     such patch entry. The connector deep-merges the patch into the matched
//     entry, so the CLI only describes the fields it wants to change.
//
// `--file` reads a JSON document from disk; `--rules` takes an inline JSON
// string. Either yields the `[]entry` array for replace/merge mode. The two
// are mutually exclusive on the CLI (enforced there, not here — builders stay
// permissive so a future caller can mix them).

import (
	"encoding/json"
	"fmt"
	"os"
)

// f64Ptr returns &v when `set` is true, else nil — a Go-friendly substitute
// for the `set ? &v : nil` ternary the Cobra RunE closures need when feeding
// optional --track-width / --clearance / --via-* flags into the *float64
// parameters of the patch builders below.
func f64Ptr(v float64, set bool) *float64 {
	if !set {
		return nil
	}
	return &v
}

// parseRulesJSON parses a JSON document (inline string or @file path) into a
// `[]map[string]any`. Used by the `-set` commands for `--rules`/`--file`.
// Accepts either a JSON array or a JSON object with a single top-level array
// field (e.g. `{"netRules": [...]}`) — common when users save a verbatim read
// result back to disk.
func parseRulesJSON(inline, file string) ([]map[string]any, error) {
	if inline == "" && file == "" {
		return nil, fmt.Errorf("either --rules (inline JSON) or --file (path to a JSON document) is required")
	}
	if inline != "" && file != "" {
		return nil, fmt.Errorf("--rules and --file are mutually exclusive")
	}

	var raw []byte
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read --file %q: %w", file, err)
		}
		raw = b
	} else {
		raw = []byte(inline)
	}

	// Try array first; if that fails, try a single-object-with-array-field shape.
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("JSON must be an array of entries or an object with a single array field: %w", err)
	}
	// Pick the first array-valued field — tolerate any of the known wrapper keys.
	for _, key := range []string{"netRules", "netByNetRules", "regionRules", "rules"} {
		if v, ok := obj[key]; ok {
			if err := json.Unmarshal(v, &arr); err == nil {
				return arr, nil
			}
		}
	}
	// Fall back: any single field that unmarshals as an array.
	for _, v := range obj {
		if err := json.Unmarshal(v, &arr); err == nil {
			return arr, nil
		}
	}
	return nil, fmt.Errorf("no array field found in JSON object")
}

// buildNetRulesSetPayload builds the `pcb.drc.net_rules.set` payload from the
// CLI flags of `pcb net-rules-set`. Exactly one of {rulesJSON, file, patches,
// removeNets} must be non-empty (enforced by the caller via required-flag
// checks; here we only validate cross-flag sanity so the builder stays usable
// from non-CLI callers).
//
//	--mode=replace (default) + --rules/--file  → replace shape
//	--mode=merge     + --rules/--file          → upserts shape
//	--patches (always mode=merge)              → patches shape (single net)
//	--remove-nets (mode=merge only)            → merge + removeNets
func buildNetRulesSetPayload(
	mode string,
	rulesJSON, file string,
	patches []map[string]any,
	removeNets []string,
) (map[string]any, error) {
	if mode != "replace" && mode != "merge" {
		return nil, fmt.Errorf("mode must be 'replace' or 'merge', got %q", mode)
	}

	// Structured-flag form: build patches and force merge mode.
	if len(patches) > 0 {
		if rulesJSON != "" || file != "" {
			return nil, fmt.Errorf("--track-width / --clearance / --via-* / --net are mutually exclusive with --rules / --file")
		}
		if mode == "replace" {
			return nil, fmt.Errorf("structured flags (--track-width etc.) require --mode=merge; replace mode would clobber every other net")
		}
		return map[string]any{
			"mode":    "merge",
			"patches": patches,
		}, nil
	}

	// removeNets-only form (merge mode).
	if rulesJSON == "" && file == "" {
		if mode == "replace" {
			return nil, fmt.Errorf("--mode=replace requires --rules or --file")
		}
		if len(removeNets) == 0 {
			return nil, fmt.Errorf("at least one of --rules, --file, --remove-nets, or structured flags (--net + --track-width etc.) is required")
		}
		return map[string]any{
			"mode":       "merge",
			"removeNets": removeNets,
		}, nil
	}

	entries, err := parseRulesJSON(rulesJSON, file)
	if err != nil {
		return nil, err
	}

	if mode == "replace" {
		return map[string]any{
			"mode":     "replace",
			"netRules": entries,
		}, nil
	}
	return map[string]any{
		"mode":     "merge",
		"upserts":  entries,
		"removeNets": removeNets,
	}, nil
}

// buildNetRuleSinglePatch builds a single `{net, patch:{...}}` entry for the
// `pcb net-rule` convenience command. Only fields the user explicitly set are
// included in the patch, so the connector's deep-merge leaves everything else
// untouched.
func buildNetRuleSinglePatch(
	net string,
	trackWidth, clearance, viaDrill, viaDiameter *float64,
) (map[string]any, error) {
	if net == "" {
		return nil, fmt.Errorf("--net is required (the net name to patch)")
	}
	patch := map[string]any{}
	if trackWidth != nil {
		patch["trackWidth"] = *trackWidth
	}
	if clearance != nil {
		patch["clearance"] = *clearance
	}
	if viaDrill != nil {
		patch["viaDrill"] = *viaDrill
	}
	if viaDiameter != nil {
		patch["viaDiameter"] = *viaDiameter
	}
	if len(patch) == 0 {
		return nil, fmt.Errorf("at least one of --track-width / --clearance / --via-drill / --via-diameter is required")
	}
	return map[string]any{
		"net":   net,
		"patch": patch,
	}, nil
}

// buildNetByNetRulesSetPayload — same shape as buildNetRulesSetPayload but for
// `pcb.drc.net_by_net_rules.set`. `entries` (from --rules/--file) feed either
// `netByNetRules` (replace) or `upserts` (merge); `removePairs` is a slice of
// {netA, netB} objects.
func buildNetByNetRulesSetPayload(
	mode string,
	rulesJSON, file string,
	patches []map[string]any,
	removePairs []map[string]string,
) (map[string]any, error) {
	if mode != "replace" && mode != "merge" {
		return nil, fmt.Errorf("mode must be 'replace' or 'merge', got %q", mode)
	}

	if len(patches) > 0 {
		if rulesJSON != "" || file != "" {
			return nil, fmt.Errorf("--clearance / --net-a / --net-b are mutually exclusive with --rules / --file")
		}
		if mode == "replace" {
			return nil, fmt.Errorf("structured flags require --mode=merge; replace mode would clobber every other net pair")
		}
		return map[string]any{
			"mode":    "merge",
			"patches": patches,
		}, nil
	}

	if rulesJSON == "" && file == "" {
		if mode == "replace" {
			return nil, fmt.Errorf("--mode=replace requires --rules or --file")
		}
		if len(removePairs) == 0 {
			return nil, fmt.Errorf("at least one of --rules, --file, --remove-pairs, or structured flags (--net-a + --net-b + --clearance) is required")
		}
		rp := make([]any, 0, len(removePairs))
		for _, p := range removePairs {
			rp = append(rp, p)
		}
		return map[string]any{
			"mode":        "merge",
			"removePairs": rp,
		}, nil
	}

	entries, err := parseRulesJSON(rulesJSON, file)
	if err != nil {
		return nil, err
	}

	if mode == "replace" {
		return map[string]any{
			"mode":          "replace",
			"netByNetRules": entries,
		}, nil
	}
	rp := make([]any, 0, len(removePairs))
	for _, p := range removePairs {
		rp = append(rp, p)
	}
	return map[string]any{
		"mode":        "merge",
		"upserts":     entries,
		"removePairs": rp,
	}, nil
}

// buildNetByNetRuleSinglePatch — single {netA, netB, patch:{clearance}} entry
// for `pcb net-by-net-rule`.
func buildNetByNetRuleSinglePatch(netA, netB string, clearance *float64) (map[string]any, error) {
	if netA == "" || netB == "" {
		return nil, fmt.Errorf("--net-a and --net-b are both required (the net pair)")
	}
	if netA == netB {
		return nil, fmt.Errorf("--net-a and --net-b must be different nets")
	}
	if clearance == nil {
		return nil, fmt.Errorf("--clearance is required (mil)")
	}
	return map[string]any{
		"netA": netA,
		"netB": netB,
		"patch": map[string]any{
			"clearance": *clearance,
		},
	}, nil
}

// buildRegionRulesSetPayload — same shape for `pcb.drc.region_rules.set`.
func buildRegionRulesSetPayload(
	mode string,
	rulesJSON, file string,
	patches []map[string]any,
	removeIds []string,
) (map[string]any, error) {
	if mode != "replace" && mode != "merge" {
		return nil, fmt.Errorf("mode must be 'replace' or 'merge', got %q", mode)
	}

	if len(patches) > 0 {
		if rulesJSON != "" || file != "" {
			return nil, fmt.Errorf("--clearance / --track-width / --region-id are mutually exclusive with --rules / --file")
		}
		if mode == "replace" {
			return nil, fmt.Errorf("structured flags require --mode=merge; replace mode would clobber every other region")
		}
		return map[string]any{
			"mode":    "merge",
			"patches": patches,
		}, nil
	}

	if rulesJSON == "" && file == "" {
		if mode == "replace" {
			return nil, fmt.Errorf("--mode=replace requires --rules or --file")
		}
		if len(removeIds) == 0 {
			return nil, fmt.Errorf("at least one of --rules, --file, --remove-ids, or structured flags (--region-id + --clearance etc.) is required")
		}
		return map[string]any{
			"mode":      "merge",
			"removeIds": removeIds,
		}, nil
	}

	entries, err := parseRulesJSON(rulesJSON, file)
	if err != nil {
		return nil, err
	}

	if mode == "replace" {
		return map[string]any{
			"mode":        "replace",
			"regionRules": entries,
		}, nil
	}
	return map[string]any{
		"mode":      "merge",
		"upserts":   entries,
		"removeIds": removeIds,
	}, nil
}

// buildRegionRuleSinglePatch — single {regionId, patch:{...}} entry for
// `pcb region-rule`.
func buildRegionRuleSinglePatch(
	regionId string,
	clearance, trackWidth, viaDrill, viaDiameter *float64,
) (map[string]any, error) {
	if regionId == "" {
		return nil, fmt.Errorf("--region-id is required")
	}
	patch := map[string]any{}
	if clearance != nil {
		patch["clearance"] = *clearance
	}
	if trackWidth != nil {
		patch["trackWidth"] = *trackWidth
	}
	if viaDrill != nil {
		patch["viaDrill"] = *viaDrill
	}
	if viaDiameter != nil {
		patch["viaDiameter"] = *viaDiameter
	}
	if len(patch) == 0 {
		return nil, fmt.Errorf("at least one of --clearance / --track-width / --via-drill / --via-diameter is required")
	}
	return map[string]any{
		"regionId": regionId,
		"patch":    patch,
	}, nil
}
