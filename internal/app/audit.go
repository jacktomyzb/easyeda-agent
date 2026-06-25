package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func runAudit(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "audit: subcommand required (tail)")
		return 2
	}
	switch args[0] {
	case "tail":
		return runAuditTail(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "audit: unknown subcommand %q (expected: tail)\n", args[0])
		return 2
	}
}

func runAuditTail(args []string, stdout io.Writer, stderr io.Writer) int {
	n := 20
	dir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "audit tail: -n requires a value")
				return 2
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v <= 0 {
				fmt.Fprintf(stderr, "audit tail: invalid -n %q\n", args[i])
				return 2
			}
			n = v
		case "--dir":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "audit tail: --dir requires a value")
				return 2
			}
			dir = args[i]
		default:
			fmt.Fprintf(stderr, "audit tail: unknown flag %q\n", args[i])
			return 2
		}
	}

	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv("HOME")
		}
		dir = filepath.Join(home, ".easyeda-agent", "audit")
	}

	lines, err := readLastLines(dir, n)
	if err != nil {
		fmt.Fprintf(stderr, "audit tail: %v\n", err)
		return 1
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
	return 0
}

// readLastLines walks the audit directory in reverse chronological order
// (newest day first) and accumulates up to n lines from the most recent files.
func readLastLines(dir string, n int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("no audit log directory at %s (run the daemon first)", dir)
		}
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var collected []string
	for _, name := range files {
		if len(collected) >= n {
			break
		}
		path := filepath.Join(dir, name)
		lines, err := readFileLines(path)
		if err != nil {
			return nil, err
		}
		need := n - len(collected)
		if len(lines) > need {
			lines = lines[len(lines)-need:]
		}
		// Older lines first within the file.
		collected = append(lines, collected...)
	}
	return collected, nil
}

func readFileLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// 1 MiB per line is enough for the largest action result payloads we expect.
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var out []string
	for scanner.Scan() {
		out = append(out, scanner.Text())
	}
	return out, scanner.Err()
}

// (unused but available for a future `audit since <duration>` subcommand)
var _ = time.Now
