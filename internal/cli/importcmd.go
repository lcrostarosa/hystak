package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lcrostarosa/hystak/internal/service"
	"github.com/spf13/cobra"
)

func (a *cliApp) newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Import servers from a client config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			candidates, err := a.svc.ImportFromFile(args[0])
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				fmt.Fprintln(out, "No servers found to import.")
				return nil
			}

			// Show discovered servers.
			fmt.Fprintf(out, "Found %d server(s):\n", len(candidates))
			for _, c := range candidates {
				conflict := ""
				if c.Conflict {
					conflict = " (conflict)"
				}
				fmt.Fprintf(out, "  %s [%s]%s\n", c.Name, c.Server.Transport, conflict)
			}

			// Resolve conflicts interactively.
			reader := bufio.NewReader(cmd.InOrStdin())
			for i, c := range candidates {
				if !c.Conflict {
					continue
				}
				fmt.Fprintf(out, "\nConflict: %q already exists in registry.\n", c.Name)
				fmt.Fprintln(out, "  [k]eep existing  [r]eplace  [n]ame (rename)  [s]kip")
				fmt.Fprint(out, "  Choice: ")

				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				input = strings.TrimSpace(strings.ToLower(input))

				switch input {
				case "k", "keep":
					candidates[i].Resolution = service.ImportKeep
				case "r", "replace":
					candidates[i].Resolution = service.ImportReplace
				case "n", "name", "rename":
					fmt.Fprint(out, "  New name: ")
					name, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("reading input: %w", err)
					}
					candidates[i].Resolution = service.ImportRename
					candidates[i].RenameTo = strings.TrimSpace(name)
				default:
					candidates[i].Resolution = service.ImportSkip
				}
			}

			if err := a.svc.ApplyImport(candidates); err != nil {
				return err
			}

			imported := 0
			for _, c := range candidates {
				if c.WasImported() {
					imported++
				}
			}
			fmt.Fprintf(out, "\nImported %d server(s).\n", imported)
			return nil
		},
	}
}
