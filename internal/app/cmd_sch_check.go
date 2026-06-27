package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ── sch check: reconstructed per-item design check ──────────────────────────
//
// The EDA schematic DRC API (eda.sch_Drc.check) returns only an aggregate
// {count,type} — the per-item detail the UI panel shows is not exposed by any
// public API. schematic.check reconstructs the actionable findings from the
// primitives directly (connector side). Rule 1: floating pins via geometric
// connectivity. This file renders that report and (with --strict) gates on it.
// Output is by designator + pin number — feed it straight into `sch no-connect`.

type checkFinding struct {
	Type       string   `json:"type"`
	Level      string   `json:"level"`
	Designator string   `json:"designator,omitempty"`
	Pins       []string `json:"pins,omitempty"`
	Count      int      `json:"count,omitempty"`
	Message    string   `json:"message,omitempty"`
}

type checkSummary struct {
	FloatingPins           int `json:"floatingPins"`
	ComponentsWithFloating int `json:"componentsWithFloating"`
	Total                  int `json:"total"`
}

type checkReport struct {
	Passed   bool           `json:"passed"`
	Summary  checkSummary   `json:"summary"`
	Findings []checkFinding `json:"findings"`
}

// runSchCheck runs the reconstructed design check, renders it, and (only with
// strict) returns a non-zero exit when there are findings. By default it is
// informational — floating IO pins are normal on an MCU board until NC-marked.
func runSchCheck(cfg *appConfig, window string, allPages, strict, asJSON bool, stdout, stderr io.Writer) error {
	payload := map[string]any{}
	if allPages {
		payload["allPages"] = true
	}
	res, err := requestAction(cfg, "schematic.check", window, payload)
	if err != nil {
		return err
	}

	rep, perr := parseCheckReport(res.Result)
	if perr != nil {
		if b, mErr := json.MarshalIndent(res.Result, "", "  "); mErr == nil {
			_, _ = stdout.Write(b)
			fmt.Fprintln(stdout)
		}
		return perr
	}

	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			return err
		}
	} else {
		renderCheckReport(rep, stdout)
	}

	if strict && len(rep.Findings) > 0 {
		return fmt.Errorf("sch check: %d finding(s) (--strict)", len(rep.Findings))
	}
	return nil
}

func parseCheckReport(result map[string]any) (checkReport, error) {
	var rep checkReport
	if result == nil {
		return rep, fmt.Errorf("empty check result")
	}
	b, err := json.Marshal(result)
	if err != nil {
		return rep, err
	}
	if err := json.Unmarshal(b, &rep); err != nil {
		return rep, fmt.Errorf("unexpected check result shape: %w", err)
	}
	return rep, nil
}

func checkLevelTag(level string) string {
	switch strings.ToLower(level) {
	case "fatal":
		return "FATAL"
	case "error":
		return "ERROR"
	case "warn":
		return "WARN"
	case "info":
		return "INFO"
	default:
		return "?????"
	}
}

func renderCheckReport(rep checkReport, w io.Writer) {
	s := rep.Summary
	fmt.Fprintf(w, "sch check: %d finding(s) — %d floating pin(s) across %d component(s)\n",
		s.Total, s.FloatingPins, s.ComponentsWithFloating)

	for _, f := range rep.Findings {
		tag := checkLevelTag(f.Level)
		msg := f.Message
		if msg == "" {
			msg = f.Type
		}
		line := fmt.Sprintf("  %-5s  %-13s  %s  %s", tag, f.Type, f.Designator, msg)
		if len(f.Pins) > 0 {
			line += "  [" + strings.Join(f.Pins, ",") + "]"
		}
		fmt.Fprintln(w, line)
	}

	if rep.Passed {
		fmt.Fprintln(w, "✓ no findings")
	} else if s.FloatingPins > 0 {
		// The floating-pin list is the exact input `sch no-connect` takes.
		fmt.Fprintln(w, "→ fix by wiring the pins, or mark intentional ones: easyeda sch no-connect --designator <D> --pin <n,n,…>")
	}
}
