package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func (a *cliApp) newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles (list, export, import)",
	}

	cmd.AddCommand(a.newProfileListCmd())
	cmd.AddCommand(a.newProfileExportCmd())
	cmd.AddCommand(a.newProfileImportCmd())

	return cmd
}

func (a *cliApp) newProfileListCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// Global profiles.
			globals, err := a.svc.ListGlobalProfiles()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tSCOPE\tDESCRIPTION")
			for _, p := range globals {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, "global", p.Description)
			}

			// Project-scoped profiles (specific project or all).
			if projectName != "" {
				profiles, err := a.svc.ListProjectProfiles(projectName)
				if err != nil {
					return err
				}
				names := make([]string, 0, len(profiles))
				for n := range profiles {
					names = append(names, n)
				}
				sort.Strings(names)
				for _, n := range names {
					pp := profiles[n]
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", n, projectName, pp.Description)
				}
			} else {
				// Show project profiles for all projects.
				for _, projName := range a.svc.ListProjectNames() {
					profiles, err := a.svc.ListProjectProfiles(projName)
					if err != nil {
						continue
					}
					names := make([]string, 0, len(profiles))
					for n := range profiles {
						names = append(names, n)
					}
					sort.Strings(names)
					for _, n := range names {
						pp := profiles[n]
						_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", n, projName, pp.Description)
					}
				}
			}

			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&projectName, "project", "", "show profiles for a specific project only")

	return cmd
}

func (a *cliApp) newProfileExportCmd() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a global profile to YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			data, err := a.svc.ExportProfile(name)
			if err != nil {
				return err
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0o644); err != nil {
					return fmt.Errorf("writing file: %w", err)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q exported to %s\n", name, outputFile)
				return nil
			}

			// Write to stdout.
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")

	return cmd
}

func (a *cliApp) newProfileImportCmd() *cobra.Command {
	var asName string

	cmd := &cobra.Command{
		Use:   "import <file.yaml>",
		Short: "Import a profile from YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			var imported *importedProfile
			if asName != "" {
				p, err := a.svc.ImportProfileAs(data, asName)
				if err != nil {
					return err
				}
				imported = &importedProfile{name: p.Name}
			} else {
				p, err := a.svc.ImportProfile(data)
				if err != nil {
					return err
				}
				imported = &importedProfile{name: p.Name}
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q imported.\n", imported.name)
			return nil
		},
	}

	cmd.Flags().StringVar(&asName, "as", "", "import under a different name (avoids conflicts)")

	return cmd
}

type importedProfile struct {
	name string
}
