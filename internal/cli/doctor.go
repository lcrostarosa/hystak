package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/profile"
	"github.com/hystak/hystak/internal/registry"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate registry and projects",
	Long:  "Check for dangling references, missing files, and configuration issues.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type doctorIssue struct {
	severity string // "error" or "warning"
	message  string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	reg, err := registry.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	profMgr := profile.NewDefaultManager()
	var issues []doctorIssue

	// Check registry counts
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Checking registry..."); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d MCP server(s)\n", reg.Servers.Len()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d skill(s)\n", reg.Skills.Len()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d hook(s)\n", reg.Hooks.Len()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d permission(s)\n", reg.Permissions.Len()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d template(s)\n", reg.Templates.Len()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d prompt(s)\n", reg.Prompts.Len()); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// Check tags for dangling references
	for name, members := range reg.ListTags() {
		for _, member := range members {
			if _, ok := reg.Servers.Get(member); !ok {
				issues = append(issues, doctorIssue{
					severity: "error",
					message:  fmt.Sprintf("tag %q references non-existent server %q", name, member),
				})
			}
		}
	}

	// Check profiles for dangling references
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "\nChecking profiles..."); err != nil {
		return err
	}
	profNames, err := profMgr.List()
	if err != nil {
		if _, wErr := fmt.Fprintf(cmd.ErrOrStderr(), "  warning: listing profiles: %v\n", err); wErr != nil {
			return wErr
		}
	} else {
		for _, profName := range profNames {
			prof, err := profMgr.Load(profName)
			if err != nil {
				issues = append(issues, doctorIssue{
					severity: "warning",
					message:  fmt.Sprintf("profile %q: load error: %v", profName, err),
				})
				continue
			}
			issues = append(issues, checkProfileRefs(reg, prof)...)
		}
	}

	// Print issues
	if len(issues) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "\nNo issues found."); err != nil {
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%d issue(s) found:\n", len(issues)); err != nil {
		return err
	}
	errors, warnings := 0, 0
	for _, issue := range issues {
		var prefix string
		if issue.severity == "error" {
			prefix = "  ERROR: "
			errors++
		} else {
			prefix = "  WARNING: "
			warnings++
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), prefix+issue.message); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\n%d error(s), %d warning(s)\n", errors, warnings); err != nil {
		return err
	}

	if errors > 0 {
		return fmt.Errorf("%d error(s) found", errors)
	}
	return nil
}

func checkProfileRefs(reg *registry.Registry, prof model.ProjectProfile) []doctorIssue {
	var issues []doctorIssue
	for _, a := range prof.MCPs {
		if _, ok := reg.Servers.Get(a.Name); !ok {
			issues = append(issues, doctorIssue{
				severity: "error",
				message:  fmt.Sprintf("profile %q references missing server %q", prof.Name, a.Name),
			})
		}
	}
	for _, name := range prof.Skills {
		if _, ok := reg.Skills.Get(name); !ok {
			issues = append(issues, doctorIssue{
				severity: "warning",
				message:  fmt.Sprintf("profile %q references missing skill %q", prof.Name, name),
			})
		}
	}
	for _, name := range prof.Hooks {
		if _, ok := reg.Hooks.Get(name); !ok {
			issues = append(issues, doctorIssue{
				severity: "warning",
				message:  fmt.Sprintf("profile %q references missing hook %q", prof.Name, name),
			})
		}
	}
	for _, name := range prof.Permissions {
		if _, ok := reg.Permissions.Get(name); !ok {
			issues = append(issues, doctorIssue{
				severity: "warning",
				message:  fmt.Sprintf("profile %q references missing permission %q", prof.Name, name),
			})
		}
	}
	if prof.Template != "" {
		if _, ok := reg.Templates.Get(prof.Template); !ok {
			issues = append(issues, doctorIssue{
				severity: "warning",
				message:  fmt.Sprintf("profile %q references missing template %q", prof.Name, prof.Template),
			})
		}
	}
	return issues
}
