package daemon

import (
	"fmt"
	"strings"
	"sync"

	"github.com/zhoushoujianwork/easyeda-agent/internal/protocol"
)

// Daemon-level stale-read advisory (SKILL iron rule 5 made mechanical).
//
// After a PCB mutation (rip-up / route / delete / via / track / pour edits) the
// per-document engine state serves STALE data to list/DRC reads until the
// document is reloaded (`easyeda doc reload`) — observed repeatedly on real
// boards. Until now this was enforced only by the agent remembering rule 5.
// This guard makes the system speak up: the daemon tracks the last un-reloaded
// PCB mutation per window and ANNOTATES (never blocks) subsequent PCB reads
// with a top-level `staleRisk` field the CLI surfaces on stderr.
//
// State machine (per windowId, in-memory):
//   SET    — a PCB-domain action with Mutates=true succeeds (catalog-driven,
//            same source of truth as autosave), except the exempt set below.
//   CLEAR  — a `doc reload` completes. Reload is a CLI composite (save → close
//            via debug.exec_js closeDocument → reopen), so the daemon keys on
//            its unique discriminator: a successful debug.exec_js whose code
//            calls closeDocument (a real close resets the per-doc engine
//            state; a mere `doc switch`/document.open does NOT and must not
//            clear). pcb.pour.rebuild also clears — it recomputes the pour
//            connectivity that goes stale (pour-mediated Connection Errors).
//   WARN   — a PCB-domain read (Mutates=false) arrives while the flag is set.
//
// Exemptions (never SET the flag):
//   - pcb.save: saving changes no copper; the daemon's debounced autosave also
//     bypasses /action entirely (dispatchSave), so neither path false-flags.
//   - pcb.pour.rebuild: it is the FIX for stale pour connectivity, not a new
//     hazard — it clears instead.
//
// windowIds churn on reconnect; a reconnected window starts clean (a window
// reload re-reads the saved document, which is exactly the stale-fix), so
// per-window in-memory state is the right lifetime.

// staleExemptActions never mark the window stale even though the catalog says
// Mutates=true (see package comment).
var staleExemptActions = map[string]bool{
	"pcb.save":         true,
	"pcb.pour.rebuild": true,
}

// pcbStaleMarks reports whether a successful `action` should mark the window's
// PCB engine state as possibly stale: any PCB-domain mutating action (catalog
// Mutates, same map autosave uses) minus the exempt set.
func pcbStaleMarks(action string) bool {
	return docTypeForAction(action) == "pcb" && mutatesAction[action] && !staleExemptActions[action]
}

// pcbStaleRead reports whether `action` is a PCB-domain read that can return
// stale data (any non-mutating pcb.* action: lists, DRC, report, snapshot …).
func pcbStaleRead(action string) bool {
	return docTypeForAction(action) == "pcb" && !mutatesAction[action]
}

// pcbStaleClears reports whether a successful request resets the stale flag.
// `doc reload` has no single typed action — its unique step is the
// debug.exec_js closeDocument call (see package comment); pcb.pour.rebuild
// clears because rebuilding pours is the documented stale-connectivity fix.
func pcbStaleClears(req *protocol.Request) bool {
	switch req.Action {
	case "pcb.pour.rebuild":
		return true
	case "debug.exec_js":
		code, _ := req.Payload["code"].(string)
		return strings.Contains(code, "closeDocument")
	}
	return false
}

// staleGuard is the per-window stale-read state machine. Methods are safe for
// concurrent use.
type staleGuard struct {
	mu sync.Mutex
	// last maps windowId → the name of the last successful PCB mutation not yet
	// followed by a reload ("" / absent = no stale risk).
	last map[string]string
}

func newStaleGuard() *staleGuard {
	return &staleGuard{last: map[string]string{}}
}

// observe applies one completed action to the state machine: it may annotate
// resp with a staleRisk advisory (reads while stale) and updates the per-window
// flag (successful mutations set it, reload/pour-rebuild clear it). Call it
// with the connector's response before writing it to the caller.
func (g *staleGuard) observe(req *protocol.Request, resp *protocol.Response) {
	if g == nil || req == nil || resp == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	// Annotate reads first: the read itself never changes the state.
	if pcbStaleRead(req.Action) {
		if mutation := g.last[req.WindowID]; mutation != "" {
			resp.StaleRisk = staleRiskMessage(mutation, req.Action)
		}
		return
	}

	// Only successful actions move the state machine.
	if !resp.OK {
		return
	}
	if pcbStaleClears(req) {
		delete(g.last, req.WindowID)
		return
	}
	if pcbStaleMarks(req.Action) {
		g.last[req.WindowID] = req.Action
	}
}

// staleRiskMessage builds the advisory. Deliberately free of timestamps so the
// CLI can deduplicate identical warnings within one composite command.
func staleRiskMessage(mutation, read string) string {
	return fmt.Sprintf(
		"PCB was mutated by %s since the last reload — %s (and DRC) may read stale engine state; run `easyeda doc reload` first (SKILL 铁律 5)",
		mutation, read)
}
