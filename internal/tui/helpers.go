package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
)

// clamp restricts val to the range [lo, hi].
func clamp(val, lo, hi int) int {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}

// moveCursor shifts cursor by delta, clamping to [0, max-1].
// Returns 0 if max <= 0.
func moveCursor(cursor, delta, max int) int {
	if max <= 0 {
		return 0
	}
	n := cursor + delta
	if n < 0 {
		return 0
	}
	if n >= max {
		return max - 1
	}
	return n
}

// writeServerFields renders the common transport/command/args/url/env/headers
// fields of a ServerDef into a strings.Builder. Used by both the servers tab
// detail pane and the importer conflict view.
func writeServerFields(b *strings.Builder, srv model.ServerDef, labelStyle lipgloss.Style) {
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Transport:"), string(srv.Transport))

	if srv.Command != "" {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Command:"), srv.Command)
	}

	if len(srv.Args) > 0 {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Args:"), strings.Join(srv.Args, " "))
	}

	if srv.URL != "" {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("URL:"), srv.URL)
	}

	if len(srv.Env) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Environment:"))
		b.WriteString("\n")
		for _, k := range sortedKeys(srv.Env) {
			fmt.Fprintf(b, "  %s=%s\n", k, srv.Env[k])
		}
	}

	if len(srv.Headers) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Headers:"))
		b.WriteString("\n")
		for _, k := range sortedKeys(srv.Headers) {
			fmt.Fprintf(b, "  %s: %s\n", k, srv.Headers[k])
		}
	}
}

// writeSkillFields renders the fields of a SkillDef into a strings.Builder.
func writeSkillFields(b *strings.Builder, skill model.SkillDef, labelStyle lipgloss.Style) {
	if skill.Description != "" {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Description:"), skill.Description)
	}
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Source:"), skill.Source)
}

// writeHookFields renders the fields of a HookDef into a strings.Builder.
func writeHookFields(b *strings.Builder, hook model.HookDef, labelStyle lipgloss.Style) {
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Event:"), hook.Event)
	if hook.Matcher != "" {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Matcher:"), hook.Matcher)
	}
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Command:"), hook.Command)
	if hook.Timeout > 0 {
		fmt.Fprintf(b, "%s %dms\n", labelStyle.Render("Timeout:"), hook.Timeout)
	}
}

// writePermissionFields renders the fields of a PermissionRule into a strings.Builder.
func writePermissionFields(b *strings.Builder, perm model.PermissionRule, labelStyle lipgloss.Style) {
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Rule:"), perm.Rule)
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Type:"), perm.EffectiveType())
}

// writeTemplateFields renders the fields of a TemplateDef into a strings.Builder.
func writeTemplateFields(b *strings.Builder, tmpl model.TemplateDef, labelStyle lipgloss.Style) {
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Source:"), tmpl.Source)
}

// writePromptFields renders the fields of a PromptDef into a strings.Builder.
func writePromptFields(b *strings.Builder, prompt model.PromptDef, labelStyle lipgloss.Style) {
	if prompt.Description != "" {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Description:"), prompt.Description)
	}
	fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Source:"), prompt.Source)
	if prompt.Category != "" {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Category:"), prompt.Category)
	}
	if prompt.Order != 0 {
		fmt.Fprintf(b, "%s %d\n", labelStyle.Render("Order:"), prompt.Order)
	}
	if len(prompt.Tags) > 0 {
		fmt.Fprintf(b, "%s %s\n", labelStyle.Render("Tags:"), strings.Join(prompt.Tags, ", "))
	}
}
