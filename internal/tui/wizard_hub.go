package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

// hubCategory identifies a section in the wizard hub sidebar.
type hubCategory int

const (
	hubMCPs hubCategory = iota
	hubSkills
	hubPermissions
	hubHooks
	hubClaudeMD
	hubPrompts
	hubEnvVars
	hubIsolation
	hubReview
	hubCategoryCount
)

var hubCategoryNames = [hubCategoryCount]string{
	"MCPs", "Skills", "Permissions", "Hooks",
	"CLAUDE.md", "Prompts", "Env Vars", "Isolation", "Review",
}

// wizardHubModel implements the hub/reconfiguration mode (S-061).
// Sidebar menu with categories + selection counts, right pane toggle list.
type wizardHubModel struct {
	keys        KeyMap
	svc         *service.Service
	projectName string
	profileName string
	category    hubCategory
	cursor      int // cursor within the right pane

	// Selections (same structure as sequential wizard)
	allMCPs       []string
	allSkills     []string
	allPerms      []string
	allHooks      []string
	allTemplates  []string
	allPrompts    []string
	selectedMCP   map[string]bool
	selectedSkill map[string]bool
	selectedPerm  map[string]bool
	selectedHook  map[string]bool
	selectedTmpl  string
	selectedPrmpt map[string]bool
	envVars       map[string]string
	isolation     model.IsolationStrategy

	// Env editor sub-model
	envEditor envEditorModel
	envActive bool
}

func newWizardHubModel(keys KeyMap, svc *service.Service, projectName, profileName string) wizardHubModel {
	h := wizardHubModel{
		keys:          keys,
		svc:           svc,
		projectName:   projectName,
		profileName:   profileName,
		selectedMCP:   make(map[string]bool),
		selectedSkill: make(map[string]bool),
		selectedPerm:  make(map[string]bool),
		selectedHook:  make(map[string]bool),
		selectedPrmpt: make(map[string]bool),
		envVars:       make(map[string]string),
		isolation:     model.IsolationNone,
	}

	for _, s := range svc.ListServers() {
		h.allMCPs = append(h.allMCPs, s.Name)
	}
	for _, s := range svc.ListSkills() {
		h.allSkills = append(h.allSkills, s.Name)
	}
	for _, p := range svc.ListPermissions() {
		h.allPerms = append(h.allPerms, p.Name)
	}
	for _, hook := range svc.ListHooks() {
		h.allHooks = append(h.allHooks, hook.Name)
	}
	for _, t := range svc.ListTemplates() {
		h.allTemplates = append(h.allTemplates, t.Name)
	}
	for _, p := range svc.ListPrompts() {
		h.allPrompts = append(h.allPrompts, p.Name)
	}

	// Pre-load from existing profile
	if prof, err := svc.LoadProfile(profileName); err == nil {
		for _, a := range prof.MCPs {
			h.selectedMCP[a.Name] = true
		}
		for _, n := range prof.Skills {
			h.selectedSkill[n] = true
		}
		for _, n := range prof.Permissions {
			h.selectedPerm[n] = true
		}
		for _, n := range prof.Hooks {
			h.selectedHook[n] = true
		}
		h.selectedTmpl = prof.Template
		for _, n := range prof.Prompts {
			h.selectedPrmpt[n] = true
		}
		for k, v := range prof.Env {
			h.envVars[k] = v
		}
		h.isolation = prof.Isolation
	}

	return h
}

func (h wizardHubModel) helpKeys() []HelpEntry {
	if h.envActive {
		return h.envEditor.helpKeys()
	}
	if h.category == hubReview {
		return []HelpEntry{{"Enter", "Save"}, {"Esc", "Cancel"}}
	}
	return []HelpEntry{{"Up/Down", "Navigate"}, {"Space", "Toggle"}, {"Esc", "Cancel"}}
}

func (h wizardHubModel) update(msg tea.KeyMsg) (wizardHubModel, tea.Cmd) {
	if h.envActive {
		var cmd tea.Cmd
		h.envEditor, cmd = h.envEditor.update(msg)
		if h.envEditor.done {
			h.envVars = h.envEditor.toMap()
			h.envActive = false
		}
		return h, cmd
	}

	switch {
	case msg.String() == "esc":
		return h, func() tea.Msg { return toolsWizardDoneMsg{} }

	case key.Matches(msg, h.keys.ListUp):
		if h.cursor > 0 {
			h.cursor--
		} else if h.category > 0 {
			h.category--
			h.cursor = 0
		}

	case key.Matches(msg, h.keys.ListDown):
		maxItems := h.currentItemCount()
		if h.cursor < maxItems-1 {
			h.cursor++
		} else if h.category < hubCategoryCount-1 {
			h.category++
			h.cursor = 0
		}

	case msg.String() == "tab" || msg.String() == "right":
		if h.category < hubCategoryCount-1 {
			h.category++
			h.cursor = 0
		}

	case msg.String() == "shift+tab" || msg.String() == "left":
		if h.category > 0 {
			h.category--
			h.cursor = 0
		}

	case key.Matches(msg, h.keys.Select):
		h.toggleCurrent()

	case key.Matches(msg, h.keys.Confirm):
		if h.category == hubReview {
			return h, h.saveProfile()
		}
		if h.category == hubEnvVars {
			h.envEditor = newEnvEditorModel(h.keys, h.envVars)
			h.envActive = true
			return h, nil
		}
	}
	return h, nil
}

func (h *wizardHubModel) toggleCurrent() {
	switch h.category {
	case hubMCPs:
		if h.cursor < len(h.allMCPs) {
			name := h.allMCPs[h.cursor]
			h.selectedMCP[name] = !h.selectedMCP[name]
		}
	case hubSkills:
		if h.cursor < len(h.allSkills) {
			name := h.allSkills[h.cursor]
			h.selectedSkill[name] = !h.selectedSkill[name]
		}
	case hubPermissions:
		if h.cursor < len(h.allPerms) {
			name := h.allPerms[h.cursor]
			h.selectedPerm[name] = !h.selectedPerm[name]
		}
	case hubHooks:
		if h.cursor < len(h.allHooks) {
			name := h.allHooks[h.cursor]
			h.selectedHook[name] = !h.selectedHook[name]
		}
	case hubClaudeMD:
		if h.cursor < len(h.allTemplates) {
			name := h.allTemplates[h.cursor]
			if h.selectedTmpl == name {
				h.selectedTmpl = ""
			} else {
				h.selectedTmpl = name
			}
		}
	case hubPrompts:
		if h.cursor < len(h.allPrompts) {
			name := h.allPrompts[h.cursor]
			h.selectedPrmpt[name] = !h.selectedPrmpt[name]
		}
	case hubIsolation:
		strategies := []model.IsolationStrategy{model.IsolationNone, model.IsolationWorktree, model.IsolationLock}
		if h.cursor < len(strategies) {
			h.isolation = strategies[h.cursor]
		}
	}
}

func (h wizardHubModel) currentItemCount() int {
	switch h.category {
	case hubMCPs:
		return len(h.allMCPs)
	case hubSkills:
		return len(h.allSkills)
	case hubPermissions:
		return len(h.allPerms)
	case hubHooks:
		return len(h.allHooks)
	case hubClaudeMD:
		return len(h.allTemplates)
	case hubPrompts:
		return len(h.allPrompts)
	case hubEnvVars:
		return len(h.envVars)
	case hubIsolation:
		return 3 // none, worktree, lock
	case hubReview:
		return 0
	}
	return 0
}

func (h wizardHubModel) saveProfile() tea.Cmd {
	svc := h.svc
	profileName := h.profileName
	selMCP := copyBoolMap(h.selectedMCP)
	selSkill := copyBoolMap(h.selectedSkill)
	selPerm := copyBoolMap(h.selectedPerm)
	selHook := copyBoolMap(h.selectedHook)
	selPrmpt := copyBoolMap(h.selectedPrmpt)
	tmpl := h.selectedTmpl
	env := make(map[string]string, len(h.envVars))
	for k, v := range h.envVars {
		env[k] = v
	}
	isolation := h.isolation

	return func() tea.Msg {
		prof, err := svc.LoadProfile(profileName)
		if err != nil {
			prof = model.ProjectProfile{Name: profileName}
		}

		prof.MCPs = nil
		for name, sel := range selMCP {
			if sel {
				prof.MCPs = append(prof.MCPs, model.MCPAssignment{Name: name})
			}
		}
		prof.Skills = selectedNames(selSkill)
		prof.Permissions = selectedNames(selPerm)
		prof.Hooks = selectedNames(selHook)
		prof.Prompts = selectedNames(selPrmpt)
		prof.Template = tmpl
		prof.Env = env
		prof.Isolation = isolation

		sort.Slice(prof.MCPs, func(i, j int) bool { return prof.MCPs[i].Name < prof.MCPs[j].Name })
		sort.Strings(prof.Skills)
		sort.Strings(prof.Permissions)
		sort.Strings(prof.Hooks)
		sort.Strings(prof.Prompts)

		if err := svc.SaveProfile(prof); err != nil {
			return toolsLaunchDoneMsg{err: err}
		}
		return toolsWizardDoneMsg{}
	}
}

func selectedNames(m map[string]bool) []string {
	var names []string
	for name, sel := range m {
		if sel {
			names = append(names, name)
		}
	}
	return names
}

func (h wizardHubModel) view(width, height int) string {
	if h.envActive {
		return h.envEditor.view(width, height)
	}

	var b strings.Builder
	b.WriteString("  Configure: " + h.projectName + "\n\n")

	// Two-pane: sidebar + content
	sidebar := h.viewSidebar()
	content := h.viewContent()

	sideLines := strings.Split(sidebar, "\n")
	contentLines := strings.Split(content, "\n")
	maxLines := max(len(sideLines), len(contentLines))

	sideWidth := 22
	for i := 0; i < maxLines; i++ {
		s := ""
		if i < len(sideLines) {
			s = sideLines[i]
		}
		c := ""
		if i < len(contentLines) {
			c = contentLines[i]
		}
		padded := s + strings.Repeat(" ", max(0, sideWidth-len([]rune(s))))
		b.WriteString("  " + padded + " | " + c + "\n")
	}

	b.WriteString("\n  ")
	b.WriteString(styleHelpKey.Render("Tab") + styleHelpDesc.Render(":Category  "))
	b.WriteString(styleHelpKey.Render("Space") + styleHelpDesc.Render(":Toggle  "))
	b.WriteString(styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	box := styleOverlayBorder.Width(min(width-4, 70)).Render(b.String())
	return centerOverlay(box, width, height)
}

func (h wizardHubModel) viewSidebar() string {
	var b strings.Builder
	for i := hubCategory(0); i < hubCategoryCount; i++ {
		name := hubCategoryNames[i]
		count := ""
		switch i {
		case hubMCPs:
			count = fmt.Sprintf(" (%d)", countTrue(h.selectedMCP))
		case hubSkills:
			count = fmt.Sprintf(" (%d)", countTrue(h.selectedSkill))
		case hubPermissions:
			count = fmt.Sprintf(" (%d)", countTrue(h.selectedPerm))
		case hubHooks:
			count = fmt.Sprintf(" (%d)", countTrue(h.selectedHook))
		case hubPrompts:
			count = fmt.Sprintf(" (%d)", countTrue(h.selectedPrmpt))
		case hubEnvVars:
			count = fmt.Sprintf(" (%d)", len(h.envVars))
		}

		line := name + count
		if i == h.category {
			b.WriteString(styleListSelected.Render(">" + line))
		} else {
			b.WriteString(" " + line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (h wizardHubModel) viewContent() string {
	var b strings.Builder

	switch h.category {
	case hubMCPs:
		h.renderToggleList(&b, h.allMCPs, h.selectedMCP)
	case hubSkills:
		h.renderToggleList(&b, h.allSkills, h.selectedSkill)
	case hubPermissions:
		h.renderToggleList(&b, h.allPerms, h.selectedPerm)
	case hubHooks:
		h.renderToggleList(&b, h.allHooks, h.selectedHook)
	case hubClaudeMD:
		for i, name := range h.allTemplates {
			marker := "○ "
			if name == h.selectedTmpl {
				marker = styleSynced.Render("● ")
			}
			line := marker + name
			if i == h.cursor {
				b.WriteString(styleListSelected.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	case hubPrompts:
		h.renderToggleList(&b, h.allPrompts, h.selectedPrmpt)
	case hubEnvVars:
		if len(h.envVars) == 0 {
			b.WriteString("(no env vars)\n")
			b.WriteString("Press Enter to edit\n")
		} else {
			for k, v := range h.envVars {
				b.WriteString(k + "=" + v + "\n")
			}
			b.WriteString("\nPress Enter to edit\n")
		}
	case hubIsolation:
		strategies := []model.IsolationStrategy{model.IsolationNone, model.IsolationWorktree, model.IsolationLock}
		for i, s := range strategies {
			marker := "○ "
			if s == h.isolation {
				marker = styleSynced.Render("● ")
			}
			line := marker + string(s)
			if i == h.cursor {
				b.WriteString(styleListSelected.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	case hubReview:
		b.WriteString(fmt.Sprintf("MCPs:        %d\n", countTrue(h.selectedMCP)))
		b.WriteString(fmt.Sprintf("Skills:      %d\n", countTrue(h.selectedSkill)))
		b.WriteString(fmt.Sprintf("Permissions: %d\n", countTrue(h.selectedPerm)))
		b.WriteString(fmt.Sprintf("Hooks:       %d\n", countTrue(h.selectedHook)))
		b.WriteString(fmt.Sprintf("Template:    %s\n", h.selectedTmpl))
		b.WriteString(fmt.Sprintf("Prompts:     %d\n", countTrue(h.selectedPrmpt)))
		b.WriteString(fmt.Sprintf("Env Vars:    %d\n", len(h.envVars)))
		b.WriteString(fmt.Sprintf("Isolation:   %s\n", h.isolation))
		b.WriteString("\nPress Enter to save.\n")
	}

	return b.String()
}

func (h wizardHubModel) renderToggleList(b *strings.Builder, items []string, sel map[string]bool) {
	if len(items) == 0 {
		b.WriteString("(none registered)\n")
		return
	}
	for i, name := range items {
		marker := "  "
		if sel[name] {
			marker = styleSynced.Render("x ")
		}
		line := marker + name
		if i == h.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
}
