package daemon

import (
	"strings"
	"testing"

	"github.com/zhoushoujianwork/easyeda-agent/internal/protocol"
)

// staleReq builds a minimal request for the stale-read state machine.
func staleReq(action, windowID string, payload map[string]any) *protocol.Request {
	return &protocol.Request{
		Envelope: protocol.Envelope{WindowID: windowID},
		Action:   action,
		Payload:  payload,
	}
}

// runStale feeds one action (with the given outcome) through the guard and
// returns the response so callers can assert on StaleRisk.
func runStale(g *staleGuard, action, windowID string, ok bool, payload map[string]any) *protocol.Response {
	resp := &protocol.Response{OK: ok}
	g.observe(staleReq(action, windowID, payload), resp)
	return resp
}

func TestStaleGuard_MutationThenReadWarns(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.route.rip_up", "w1", true, nil)

	for _, read := range []string{"pcb.line.list", "pcb.via.list", "pcb.components.list", "pcb.drc.check", "pcb.pour.list"} {
		resp := runStale(g, read, "w1", true, nil)
		if resp.StaleRisk == "" {
			t.Errorf("%s after pcb.route.rip_up: want staleRisk advisory, got none", read)
			continue
		}
		if !strings.Contains(resp.StaleRisk, "pcb.route.rip_up") {
			t.Errorf("%s advisory should name the mutation, got %q", read, resp.StaleRisk)
		}
		if !strings.Contains(resp.StaleRisk, "doc reload") {
			t.Errorf("%s advisory should tell the fix (doc reload), got %q", read, resp.StaleRisk)
		}
	}
}

func TestStaleGuard_ReloadClears(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.via.create", "w1", true, nil)

	// `doc reload` is a CLI composite; its daemon-visible discriminator is the
	// debug.exec_js closeDocument step (a doc switch/document.open must NOT clear).
	runStale(g, "debug.exec_js", "w1", true, map[string]any{
		"code": `return await eda.dmt_EditorControl.closeDocument("tab-1")`,
	})

	if resp := runStale(g, "pcb.drc.check", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("read after reload: want no staleRisk, got %q", resp.StaleRisk)
	}
}

func TestStaleGuard_PourRebuildClears(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.route.delete", "w1", true, nil)
	runStale(g, "pcb.pour.rebuild", "w1", true, nil)

	if resp := runStale(g, "pcb.line.list", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("read after pour-rebuild: want no staleRisk, got %q", resp.StaleRisk)
	}
	// pour.rebuild is also exempt from marking: it must never re-arm the flag.
	if resp := runStale(g, "pcb.drc.check", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("pour-rebuild must not itself mark stale, got %q", resp.StaleRisk)
	}
}

func TestStaleGuard_FailedMutationDoesNotMark(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.route.rip_up", "w1", false, nil)

	if resp := runStale(g, "pcb.line.list", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("failed mutation must not mark stale, got %q", resp.StaleRisk)
	}
}

func TestStaleGuard_SaveDoesNotMarkOrClear(t *testing.T) {
	g := newStaleGuard()

	// pcb.save alone (e.g. explicit save, or a raw caller) never marks …
	runStale(g, "pcb.save", "w1", true, nil)
	if resp := runStale(g, "pcb.line.list", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("pcb.save must not mark stale, got %q", resp.StaleRisk)
	}

	// … and does not clear an existing mark either (saving fixes nothing).
	runStale(g, "pcb.line.create", "w1", true, nil)
	runStale(g, "pcb.save", "w1", true, nil)
	if resp := runStale(g, "pcb.line.list", "w1", true, nil); resp.StaleRisk == "" {
		t.Error("pcb.save must not clear the stale mark")
	}
}

func TestStaleGuard_DocSwitchDoesNotClear(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.clear_routing", "w1", true, nil)

	// A foreground tab switch (document.open) does not reload engine state.
	runStale(g, "document.open", "w1", true, map[string]any{"uuid": "abc"})

	if resp := runStale(g, "pcb.drc.check", "w1", true, nil); resp.StaleRisk == "" {
		t.Error("document.open (doc switch) must NOT clear the stale mark — only a real reload does")
	}
}

func TestStaleGuard_SchematicMutationDoesNotMarkPcb(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "schematic.wire.create", "w1", true, nil)

	if resp := runStale(g, "pcb.line.list", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("schematic mutation must not mark PCB stale, got %q", resp.StaleRisk)
	}
}

func TestStaleGuard_PerWindowIsolation(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.route.rip_up", "w1", true, nil)

	if resp := runStale(g, "pcb.line.list", "w2", true, nil); resp.StaleRisk != "" {
		t.Errorf("mutation on w1 must not flag reads on w2, got %q", resp.StaleRisk)
	}
	if resp := runStale(g, "pcb.line.list", "w1", true, nil); resp.StaleRisk == "" {
		t.Error("mutation on w1 should still flag reads on w1")
	}
}

func TestStaleGuard_MutatingActionsNotAnnotated(t *testing.T) {
	g := newStaleGuard()
	runStale(g, "pcb.route.rip_up", "w1", true, nil)

	// A follow-up mutation is about to change the board anyway — no advisory.
	if resp := runStale(g, "pcb.line.create", "w1", true, nil); resp.StaleRisk != "" {
		t.Errorf("mutating action should not carry staleRisk, got %q", resp.StaleRisk)
	}
}

// TestStaleGuard_CatalogClassification pins the catalog-driven classification
// for the load-bearing copper mutations named by iron rule 5.
func TestStaleGuard_CatalogClassification(t *testing.T) {
	marks := []string{
		"pcb.route.rip_up", "pcb.route.delete", "pcb.route.via_hop",
		"pcb.clear_routing", "pcb.line.create", "pcb.via.create",
		"pcb.pour.create", "pcb.pour.delete", "pcb.import_autoroute",
	}
	for _, a := range marks {
		if !pcbStaleMarks(a) {
			t.Errorf("pcbStaleMarks(%q) = false, want true", a)
		}
	}
	noMarks := []string{
		"pcb.save", "pcb.pour.rebuild", // exempt
		"pcb.line.list", "pcb.drc.check", // reads
		"schematic.wire.create", "document.open", // other domains
	}
	for _, a := range noMarks {
		if pcbStaleMarks(a) {
			t.Errorf("pcbStaleMarks(%q) = true, want false", a)
		}
	}
	reads := []string{"pcb.line.list", "pcb.via.list", "pcb.pour.list", "pcb.components.list", "pcb.drc.check", "pcb.nets.list", "pcb.report", "pcb.board.info"}
	for _, a := range reads {
		if !pcbStaleRead(a) {
			t.Errorf("pcbStaleRead(%q) = false, want true", a)
		}
	}
	if pcbStaleRead("schematic.components.list") {
		t.Error("schematic reads must not be classified as PCB stale reads")
	}
}
