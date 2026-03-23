package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/profile"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	profileExportOutput string
	profileImportAs     string
	profileListProject  string
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "List, export, and import profiles.",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  "Show all available profiles with name, scope, and description.",
	RunE:  runProfileList,
}

var profileExportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export a profile to YAML",
	Long:  "Serialize a profile to YAML on stdout or to a file with -o.",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileExport,
}

var profileImportCmd = &cobra.Command{
	Use:         "import <file>",
	Short:       "Import a profile from YAML",
	Long:        "Import a profile from a YAML file. Use --as to rename on import.",
	Args:        cobra.ExactArgs(1),
	Annotations: map[string]string{"mutates": "true"},
	RunE:        runProfileImport,
}

func init() {
	profileExportCmd.Flags().StringVarP(&profileExportOutput, "output", "o", "", "write to file instead of stdout")
	profileImportCmd.Flags().StringVar(&profileImportAs, "as", "", "rename profile on import")
	profileListCmd.Flags().StringVar(&profileListProject, "project", "", "filter by project scope")
	profileCmd.AddCommand(profileListCmd, profileExportCmd, profileImportCmd)
	rootCmd.AddCommand(profileCmd)
}

// S-032: List profiles
func runProfileList(cmd *cobra.Command, args []string) error {
	mgr := profile.NewDefaultManager()

	profiles, err := mgr.LoadAll()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No profiles found."); err != nil {
			return err
		}
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tSCOPE\tDESCRIPTION"); err != nil {
		return err
	}

	for _, p := range profiles {
		if profileListProject != "" && p.Project != profileListProject && p.Scope != "global" {
			continue
		}
		scope := p.Scope
		if scope == "" {
			scope = "global"
		}
		desc := truncateStr(p.Description, 50)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, scope, desc); err != nil {
			return err
		}
	}
	return w.Flush()
}

// S-030: Export profile
func runProfileExport(cmd *cobra.Command, args []string) error {
	mgr := profile.NewDefaultManager()

	prof, err := mgr.Load(args[0])
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(prof)
	if err != nil {
		return err
	}

	if profileExportOutput != "" {
		if err := os.WriteFile(profileExportOutput, data, 0o644); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Exported to %s\n", profileExportOutput); err != nil {
			return err
		}
		return nil
	}

	_, err = cmd.OutOrStdout().Write(data)
	return err
}

// S-031: Import profile
func runProfileImport(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("reading profile file: %w", err)
	}

	var prof profile.ImportedProfile
	if err := yaml.Unmarshal(data, &prof); err != nil {
		return fmt.Errorf("parsing profile: %w", err)
	}

	if profileImportAs != "" {
		prof.Name = profileImportAs
	}
	if prof.Name == "" {
		return fmt.Errorf("profile name is required (use --as to set one)")
	}

	mgr := profile.NewDefaultManager()
	if err := mgr.Save(prof.ToProjectProfile()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Imported profile %q\n", prof.Name); err != nil {
		return err
	}
	return nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
