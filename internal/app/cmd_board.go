package app

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// newBoardCmd returns the "board" subcommand group: manage Boards (板子/组合),
// the schematic↔PCB binding. A Board groups exactly one schematic + one PCB and
// is identified by NAME. All map to eda.dmt_Board.*.
func newBoardCmd(cfg *appConfig, stdout, stderr io.Writer) *cobra.Command {
	var window string

	board := &cobra.Command{
		Use:   "board",
		Short: "Manage Boards (板子/组合) — the schematic↔PCB binding",
		Long: "Manage Boards (板子/组合).\n\n" +
			"A Board groups exactly one schematic + one PCB — this is how the two are\n" +
			"kept together (and what `pcb import_changes` follows). Boards are identified\n" +
			"by NAME, not UUID.",
	}
	board.PersistentFlags().StringVar(&window, "window", "", "EasyEDA window ID")

	// ── list ───────────────────────────────────────────────────────────────
	// board.list
	board.AddCommand(&cobra.Command{
		Use:     "list",
		Short:   "List all Boards in the current project (name + schematic + pcb)",
		Args:    cobra.NoArgs,
		Example: `  easyeda board list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return dispatch(cfg, "board.list", window, nil, stdout, stderr)
		},
	})

	// ── current ────────────────────────────────────────────────────────────
	// board.current
	board.AddCommand(&cobra.Command{
		Use:     "current",
		Short:   "Read the current Board (its bound schematic + PCB)",
		Args:    cobra.NoArgs,
		Example: `  easyeda board current`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return dispatch(cfg, "board.current", window, nil, stdout, stderr)
		},
	})

	// ── create ─────────────────────────────────────────────────────────────
	// board.create
	{
		var schUuid, pcbUuid string
		c := &cobra.Command{
			Use:   "create",
			Short: "Bind a schematic and/or PCB into a new Board (组合)",
			Args:  cobra.NoArgs,
			Example: `  easyeda board create --schematic <schUuid> --pcb <pcbUuid>
  easyeda board create --schematic <schUuid>`,
			RunE: func(cmd *cobra.Command, args []string) error {
				if schUuid == "" && pcbUuid == "" {
					return fmt.Errorf("pass at least one of --schematic / --pcb")
				}
				payload := map[string]any{}
				if schUuid != "" {
					payload["schematicUuid"] = schUuid
				}
				if pcbUuid != "" {
					payload["pcbUuid"] = pcbUuid
				}
				return dispatch(cfg, "board.create", window, payload, stdout, stderr)
			},
		}
		c.Flags().StringVar(&schUuid, "schematic", "", "schematic UUID to bind")
		c.Flags().StringVar(&pcbUuid, "pcb", "", "PCB UUID to bind")
		board.AddCommand(c)
	}

	// ── rename ─────────────────────────────────────────────────────────────
	// board.rename
	{
		var name, newName string
		c := &cobra.Command{
			Use:     "rename",
			Short:   "Rename a Board by its current name",
			Args:    cobra.NoArgs,
			Example: `  easyeda board rename --name "Board1" --new "电源板"`,
			RunE: func(cmd *cobra.Command, args []string) error {
				if name == "" {
					return fmt.Errorf("--name is required")
				}
				if newName == "" {
					return fmt.Errorf("--new is required")
				}
				return dispatch(cfg, "board.rename", window,
					map[string]any{"name": name, "newName": newName}, stdout, stderr)
			},
		}
		c.Flags().StringVar(&name, "name", "", "current board name (required)")
		c.Flags().StringVar(&newName, "new", "", "new board name (required)")
		board.AddCommand(c)
	}

	// ── copy ───────────────────────────────────────────────────────────────
	// board.copy
	{
		var name string
		c := &cobra.Command{
			Use:     "copy",
			Short:   "Copy a Board (its schematic + PCB) into a new Board",
			Args:    cobra.NoArgs,
			Example: `  easyeda board copy --name "Board1"`,
			RunE: func(cmd *cobra.Command, args []string) error {
				if name == "" {
					return fmt.Errorf("--name is required")
				}
				return dispatch(cfg, "board.copy", window,
					map[string]any{"name": name}, stdout, stderr)
			},
		}
		c.Flags().StringVar(&name, "name", "", "source board name (required)")
		board.AddCommand(c)
	}

	// ── delete ─────────────────────────────────────────────────────────────
	// board.delete
	{
		var name string
		c := &cobra.Command{
			Use:     "delete",
			Short:   "Delete a Board by name (no undo)",
			Args:    cobra.NoArgs,
			Example: `  easyeda board delete --name "Board1"`,
			RunE: func(cmd *cobra.Command, args []string) error {
				if name == "" {
					return fmt.Errorf("--name is required")
				}
				return dispatch(cfg, "board.delete", window,
					map[string]any{"name": name}, stdout, stderr)
			},
		}
		c.Flags().StringVar(&name, "name", "", "board name to delete (required)")
		board.AddCommand(c)
	}

	return board
}
