package protocol

import "testing"

func TestPhase1ActionsHaveStableNames(t *testing.T) {
	actions := Phase1Actions()
	if len(actions) == 0 {
		t.Fatal("expected phase 1 actions")
	}

	seen := map[string]bool{}
	for _, action := range actions {
		if action.Name == "" {
			t.Fatalf("action has empty name: %#v", action)
		}
		if seen[action.Name] {
			t.Fatalf("duplicate action name: %s", action.Name)
		}
		seen[action.Name] = true
		if action.Phase != 1 {
			t.Fatalf("expected phase 1 action, got phase %d for %s", action.Phase, action.Name)
		}
	}

	for _, required := range []string{
		"system.health",
		"schematic.components.list",
		"schematic.component.place",
		"schematic.wire.create",
		"schematic.drc.check",
		"schematic.export.bom",
	} {
		if !seen[required] {
			t.Fatalf("missing required action: %s", required)
		}
	}
}
