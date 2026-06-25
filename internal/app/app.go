package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhoushoujianwork/easyeda-agent/internal/protocol"
	"github.com/zhoushoujianwork/easyeda-agent/internal/version"
)

const (
	defaultHost      = "127.0.0.1"
	defaultPortStart = 49620
	defaultPortEnd   = 49629
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "version":
		fmt.Fprintf(stdout, "%s %s\n", version.Name, version.Version)
		return 0
	case "phase1":
		printPhase1(stdout)
		return 0
	case "actions":
		return printActions(stdout, stderr)
	case "health":
		return health(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `%s

Usage:
  easyeda version
  easyeda phase1
  easyeda actions
  easyeda health [--host 127.0.0.1] [--ports 49620-49629]

`, version.Name)
}

func printPhase1(w io.Writer) {
	fmt.Fprintln(w, "Phase 1: schematic automation")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Goals:")
	fmt.Fprintln(w, "  - connect to an active EasyEDA schematic window")
	fmt.Fprintln(w, "  - inspect project, document, pages, components, wires, and selections")
	fmt.Fprintln(w, "  - place and modify schematic components")
	fmt.Fprintln(w, "  - create wires, net flags, and ports")
	fmt.Fprintln(w, "  - run DRC, save, export netlist/BOM, and capture snapshots")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Out of scope: PCB editing, footprint authoring, manufacturing export beyond schematic BOM/netlist.")
}

func printActions(stdout io.Writer, stderr io.Writer) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(protocol.Phase1Actions()); err != nil {
		fmt.Fprintf(stderr, "encode actions: %v\n", err)
		return 1
	}
	return 0
}

type healthOptions struct {
	host      string
	portStart int
	portEnd   int
}

type healthResult struct {
	Status  string          `json:"status"`
	Host    string          `json:"host"`
	Ports   string          `json:"ports"`
	Found   *daemonHealth   `json:"found,omitempty"`
	Checked []checkedHealth `json:"checked"`
}

type checkedHealth struct {
	Port   int    `json:"port"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type daemonHealth struct {
	Port    int             `json:"port"`
	Service string          `json:"service,omitempty"`
	Raw     json.RawMessage `json:"raw,omitempty"`
}

func health(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseHealthOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "health: %v\n", err)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := scanHealth(ctx, opts)
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(stderr, "encode health: %v\n", err)
		return 1
	}
	if result.Found == nil {
		return 1
	}
	return 0
}

func parseHealthOptions(args []string) (healthOptions, error) {
	opts := healthOptions{
		host:      defaultHost,
		portStart: defaultPortStart,
		portEnd:   defaultPortEnd,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host":
			i++
			if i >= len(args) {
				return opts, errors.New("--host requires a value")
			}
			opts.host = args[i]
		case "--ports":
			i++
			if i >= len(args) {
				return opts, errors.New("--ports requires a value")
			}
			start, end, err := parsePortRange(args[i])
			if err != nil {
				return opts, err
			}
			opts.portStart = start
			opts.portEnd = end
		default:
			return opts, fmt.Errorf("unknown health option: %s", args[i])
		}
	}

	return opts, nil
}

func parsePortRange(raw string) (int, int, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid port range %q, expected start-end", raw)
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start port %q", parts[0])
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end port %q", parts[1])
	}
	if start <= 0 || end <= 0 || start > end {
		return 0, 0, fmt.Errorf("invalid port range %q", raw)
	}
	return start, end, nil
}

func scanHealth(ctx context.Context, opts healthOptions) healthResult {
	result := healthResult{
		Status: "not_found",
		Host:   opts.host,
		Ports:  fmt.Sprintf("%d-%d", opts.portStart, opts.portEnd),
	}

	client := http.Client{Timeout: 700 * time.Millisecond}
	for port := opts.portStart; port <= opts.portEnd; port++ {
		checked := checkedHealth{Port: port, Status: "unreachable"}
		url := fmt.Sprintf("http://%s:%d/health", opts.host, port)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			checked.Error = err.Error()
			result.Checked = append(result.Checked, checked)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			checked.Error = err.Error()
			result.Checked = append(result.Checked, checked)
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		closeErr := resp.Body.Close()
		if readErr != nil {
			checked.Status = "read_error"
			checked.Error = readErr.Error()
			result.Checked = append(result.Checked, checked)
			continue
		}
		if closeErr != nil {
			checked.Status = "close_error"
			checked.Error = closeErr.Error()
			result.Checked = append(result.Checked, checked)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			checked.Status = fmt.Sprintf("http_%d", resp.StatusCode)
			result.Checked = append(result.Checked, checked)
			continue
		}

		service := serviceName(body)
		checked.Status = "ok"
		result.Checked = append(result.Checked, checked)
		if service == "easyeda-agent" {
			raw := append(json.RawMessage(nil), body...)
			result.Status = "found"
			result.Found = &daemonHealth{Port: port, Service: service, Raw: raw}
			return result
		}
	}

	return result
}

func serviceName(body []byte) string {
	var payload struct {
		Service string `json:"service"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Service
}
