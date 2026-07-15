package daemon

import (
	"context"
	"sync"
	"time"

	"github.com/zhoushoujianwork/easyeda-agent/internal/protocol"
)

// Daemon-level debounced autosave.
//
// `place`/`wire`/`modify` only mutate the in-memory EasyEDA document; without a
// save they never hit disk, so a window reload / daemon restart / EasyEDA crash
// silently loses the work (observed live: placed parts vanished after the daemon
// hot-reloaded). This is the infrastructure safety net — after any content-
// changing action the daemon arms a trailing-debounce timer and saves once the
// edits quiesce, so the agent doesn't have to remember to. Opt-in via
// Options.AutosaveDebounce (0 = off).

// mutatesAction maps each action name to whether it mutates the document, so the
// daemon fires an autosave after content-changing actions only.
var mutatesAction = func() map[string]bool {
	m := map[string]bool{}
	for _, a := range protocol.AllActions() {
		m[a.Name] = a.Mutates
	}
	return m
}()

// dryRunPayloadField is the payload key every dry-runnable action uses to mark a
// request as a PREVIEW. It is a project-wide convention, not a per-action one:
// the actions that forward a preview flag to the connector (pcb.page.clear,
// schematic.page.clear, pcb.beautify) all send exactly `dryRun`, and every other
// `--dry-run` CLI flag (mount-holes / power-planes / pour-fit / route-short /
// autoconnect …) short-circuits inside the CLI and never dispatches a mutating
// action at all. Keeping one key means the daemon needs no per-action catalog
// entry to tell a preview from a write (issue #112).
const dryRunPayloadField = "dryRun"

// isDryRunRequest reports whether a request is a preview that changes nothing.
// Strictly a JSON `true` — an unparseable/absent flag counts as a real write,
// which is the safe direction to err (a missed preview only costs a redundant
// save; a misread write would lose the safety net entirely).
func isDryRunRequest(req *protocol.Request) bool {
	if req == nil {
		return false
	}
	v, ok := req.Payload[dryRunPayloadField]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

// requestMutates reports whether a request actually changes the document: the
// catalog's Mutates flag (action-name granularity) MINUS dry-run previews, which
// the catalog cannot see. `pcb.page.clear --dry-run` only enumerates, so it must
// neither arm autosave nor the stale-read guard (issue #112).
func requestMutates(req *protocol.Request) bool {
	return req != nil && mutatesAction[req.Action] && !isDryRunRequest(req)
}

// saveActionForDocType returns the typed save action for a documentType, or ""
// when none exists. Schematic and PCB both have a typed save; a PCB-mutating
// action therefore arms a debounced pcb.save the same way a schematic edit arms
// schematic.save.
func saveActionForDocType(docType string) string {
	switch docType {
	case "schematic":
		return "schematic.save"
	case "pcb":
		return "pcb.save"
	}
	return ""
}

// autosaver debounces per-window saves: a burst of edits on one window collapses
// into a single save fired `debounce` after the LAST edit (trailing debounce).
type autosaver struct {
	mu       sync.Mutex
	debounce time.Duration
	timers   map[string]*time.Timer
	save     func(windowID, saveAction string)
}

func newAutosaver(debounce time.Duration, save func(windowID, saveAction string)) *autosaver {
	return &autosaver{
		debounce: debounce,
		timers:   map[string]*time.Timer{},
		save:     save,
	}
}

// schedule (re)arms the trailing-debounce timer for windowID. Each call resets
// the timer, so N rapid mutations coalesce into one save `debounce` after the
// last. nil/zero-debounce receiver is a no-op (autosave disabled).
func (a *autosaver) schedule(windowID, saveAction string) {
	if a == nil || a.debounce <= 0 || windowID == "" || saveAction == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if t, ok := a.timers[windowID]; ok {
		t.Stop()
	}
	a.timers[windowID] = time.AfterFunc(a.debounce, func() {
		a.mu.Lock()
		delete(a.timers, windowID)
		a.mu.Unlock()
		a.save(windowID, saveAction)
	})
}

// stop cancels all pending timers (daemon shutdown). Pending edits are not force-
// flushed — flushing would race shutdown; the next session saves on first edit.
func (a *autosaver) stop() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, t := range a.timers {
		t.Stop()
	}
	a.timers = map[string]*time.Timer{}
}

// maybeAutosave arms an autosave after a successful mutating action. It EXCLUDES
// the save action itself (schematic.save is Mutates=true) so a save never arms
// another save — no recursion.
func (s *Server) maybeAutosave(req *protocol.Request) {
	if s.autosave == nil || req == nil {
		return
	}
	if !requestMutates(req) {
		return
	}
	saveAction := saveActionForDocType(docTypeForAction(req.Action))
	if saveAction == "" || req.Action == saveAction {
		return
	}
	s.autosave.schedule(req.WindowID, saveAction)
}

// dispatchSave forwards the debounced save to the window's connector. Best-effort
// and fired from a timer (no HTTP caller): logged + audited, never surfaced.
func (s *Server) dispatchSave(windowID, saveAction string) {
	target, ok := s.hub.target(windowID)
	if !ok {
		return // window disconnected before the timer fired
	}
	req := protocol.Request{
		Envelope: protocol.Envelope{
			ID:        s.nextRequestID(),
			Type:      protocol.TypeRequest,
			Version:   "v1",
			WindowID:  windowID,
			CreatedAt: time.Now().UTC(),
		},
		Action: saveAction,
	}
	ctx, cancel := context.WithTimeout(s.connCtx, dispatchTimeout)
	defer cancel()
	started := time.Now().UTC()
	resp, err := target.dispatch(ctx, req)
	if err != nil {
		s.logf("autosave: %s on %s failed: %v", saveAction, windowID, err)
		return
	}
	s.audit.Append(fromResponse(started, &req, resp))
	s.logf("autosave: %s on %s (ok=%v)", saveAction, windowID, resp.OK)
}
