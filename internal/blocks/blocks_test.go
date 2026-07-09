package blocks

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	all, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("no blocks embedded — did `make sync-blocks` run?")
	}
	for _, b := range all {
		if b.ID == "" || b.Desc == "" {
			t.Errorf("block missing id/desc: %+v", b)
		}
		if b.Ready() != (b.Validated != nil && *b.Validated != "") {
			t.Errorf("%s: Ready() disagrees with Validated", b.ID)
		}
	}
}

func TestGetPrefixOptional(t *testing.T) {
	all, err := Load()
	if err != nil || len(all) == 0 {
		t.Skip("no blocks")
	}
	want := all[0].ID // e.g. block.xxx
	bare := want[len("block."):]
	for _, id := range []string{want, bare} {
		b, ok, err := Get(id)
		if err != nil || !ok {
			t.Fatalf("Get(%q): ok=%v err=%v", id, ok, err)
		}
		if b.ID != want {
			t.Errorf("Get(%q) → %s, want %s", id, b.ID, want)
		}
	}
}

// TestFilenameMatchesID enforces the one-block-per-file contract: data/<id>.json
// where <id> is the block id minus the `block.` prefix.
func TestFilenameMatchesID(t *testing.T) {
	entries, err := data.ReadDir("data")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, "_") {
			continue
		}
		raw, _ := data.ReadFile("data/" + name)
		var b Block
		if err := json.Unmarshal(raw, &b); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		want := "block." + strings.TrimSuffix(name, ".json")
		if b.ID != want {
			t.Errorf("%s: id %q, want %q (filename minus block.)", name, b.ID, want)
		}
	}
}

// TestAttributionOnValidated: a validated (non-draft) block must carry author +
// added + updated (permanent, traceable credit).
func TestAttributionOnValidated(t *testing.T) {
	all, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range all {
		if !b.Ready() {
			continue
		}
		if b.Author == "" || b.Added == "" || b.Updated == "" {
			t.Errorf("%s: validated block missing author/added/updated", b.ID)
		}
	}
}

// standardPartsPath is the skill's part library; block parts cross-reference it.
const standardPartsPath = "../../skills/easyeda-agent/references/standard-parts.json"

// TestPartsExistInStandardParts: every block's parts[].part (and alt[]) must be a
// real key in standard-parts.json, so BOM/LCSC stays single-sourced.
func TestPartsExistInStandardParts(t *testing.T) {
	raw, err := os.ReadFile(standardPartsPath)
	if err != nil {
		t.Skipf("standard-parts.json not found (outside repo tree): %v", err)
	}
	var sp struct {
		Parts map[string]any `json:"parts"`
	}
	if err := json.Unmarshal(raw, &sp); err != nil {
		t.Fatalf("parse standard-parts.json: %v", err)
	}
	all, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range all {
		var parts map[string]struct {
			Part string   `json:"part"`
			Alt  []string `json:"alt"`
		}
		if err := json.Unmarshal(b.Raw, &struct {
			Parts *map[string]struct {
				Part string   `json:"part"`
				Alt  []string `json:"alt"`
			} `json:"parts"`
		}{Parts: &parts}); err != nil {
			t.Errorf("%s: parse parts: %v", b.ID, err)
			continue
		}
		for role, p := range parts {
			for _, key := range append([]string{p.Part}, p.Alt...) {
				if key == "" {
					continue
				}
				if _, ok := sp.Parts[key]; !ok {
					t.Errorf("%s role %s: part %q not in standard-parts.json", b.ID, role, key)
				}
			}
		}
	}
}
