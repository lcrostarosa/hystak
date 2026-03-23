package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
	"github.com/spf13/cobra"
)

var diffAll bool

var diffCmd = &cobra.Command{
	Use:   "diff <project>",
	Short: "Show drift between registry and deployed configs",
	Long:  "Compare expected MCP server configurations against what is deployed, using semantic comparison.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffAll, "all", false, "show drift for all projects")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	if !diffAll && len(args) == 0 {
		return fmt.Errorf("project name is required (or use --all)")
	}

	svc, err := buildServiceReadOnly()
	if err != nil {
		return err
	}

	if diffAll {
		return runDiffAll(cmd, svc)
	}

	return runDiffProject(cmd, svc, args[0])
}

func runDiffProject(cmd *cobra.Command, svc *service.Service, name string) error {
	results, err := svc.DiffProject(name)
	if err != nil {
		return err
	}

	if allSynced(results) {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No drift detected."); err != nil {
			return err
		}
		return nil
	}

	return printDiffResults(cmd, results)
}

func runDiffAll(cmd *cobra.Command, svc *service.Service) error {
	projects := svc.ListProjects()
	if len(projects) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "No projects registered."); err != nil {
			return err
		}
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "PROJECT\tSTATUS"); err != nil {
		return err
	}

	for _, p := range projects {
		results, err := svc.DiffProject(p.Name)
		if err != nil {
			if _, err := fmt.Fprintf(w, "%s\terror: %v\n", p.Name, err); err != nil {
				return err
			}
			continue
		}

		status := summarizeDrift(results)
		if _, err := fmt.Fprintf(w, "%s\t%s\n", p.Name, status); err != nil {
			return err
		}
	}

	return w.Flush()
}

func printDiffResults(cmd *cobra.Command, results []service.DiffResult) error {
	// Sort by server name for deterministic output
	sort.Slice(results, func(i, j int) bool {
		return results[i].ServerName < results[j].ServerName
	})

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "SERVER\tSTATUS\tDETAILS"); err != nil {
		return err
	}

	for _, r := range results {
		detail := diffDetail(r)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", r.ServerName, r.Status, detail); err != nil {
			return err
		}
	}

	return w.Flush()
}

func diffDetail(r service.DiffResult) string {
	switch r.Status {
	case model.DriftDrifted:
		var parts []string
		if r.Expected.Command != r.Deployed.Command {
			parts = append(parts, fmt.Sprintf("command: %q -> %q", r.Deployed.Command, r.Expected.Command))
		}
		if r.Expected.URL != r.Deployed.URL {
			parts = append(parts, fmt.Sprintf("url: %q -> %q", r.Deployed.URL, r.Expected.URL))
		}
		if r.Expected.Transport != r.Deployed.Transport {
			parts = append(parts, fmt.Sprintf("transport: %s -> %s", r.Deployed.Transport, r.Expected.Transport))
		}
		if !slicesEqualNilStr(r.Expected.Args, r.Deployed.Args) {
			parts = append(parts, "args changed")
		}
		if !mapsEqualNilStr(r.Expected.Env, r.Deployed.Env) {
			parts = append(parts, "env changed")
		}
		if !mapsEqualNilStr(r.Expected.Headers, r.Deployed.Headers) {
			parts = append(parts, "headers changed")
		}
		if len(parts) == 0 {
			return "fields differ"
		}
		return strings.Join(parts, "; ")
	case model.DriftMissing:
		return "not deployed"
	case model.DriftUnmanaged:
		return "not in registry"
	default:
		return ""
	}
}

func allSynced(results []service.DiffResult) bool {
	for _, r := range results {
		if r.Status != model.DriftSynced {
			return false
		}
	}
	return true
}

func summarizeDrift(results []service.DiffResult) string {
	if allSynced(results) {
		return "synced"
	}
	counts := make(map[model.DriftStatus]int)
	for _, r := range results {
		counts[r.Status]++
	}
	var parts []string
	for _, s := range []model.DriftStatus{model.DriftDrifted, model.DriftMissing, model.DriftUnmanaged} {
		if c := counts[s]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, s))
		}
	}
	return strings.Join(parts, ", ")
}

// slicesEqualNilStr compares two string slices treating nil and empty as equal.
func slicesEqualNilStr(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// mapsEqualNilStr compares two string maps treating nil and empty as equal.
func mapsEqualNilStr(a, b map[string]string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
