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

// wizardStep identifies the current step in the sequential launch wizard.
type wizardStep int

const (
	wizardStepMCPs    wizardStep = iota // Step 1: Select MCPs
	wizardStepOptions                   // Step 2: Quick options (skills, permissions, hooks)
	wizardStepReview                    // Step 3: Review & launch
	wizardStepCount
)

// wizardModel implements the sequential launch wizard (S-060).
type wizardModel struct {
	keys        KeyMap
	svc         *service.Service
	projectName string
	profileName string
	step        wizardStep
	cursor      int

	// Step 1: MCPs
	allMCPs     []string
	selectedMCP map[string]bool

	// Step 2: Options
	allSkills     []string
	allPerms      []string
	allHooks      []string
	selectedSkill map[string]bool
	selectedPerm  map[string]bool
	selectedHook  map[string]bool
}

func newWizardModel(keys KeyMap, svc *service.Service, projectName, profileName string) wizardModel {
	w := wizardModel{
		keys:          keys,
		svc:           svc,
		projectName:   projectName,
		profileName:   profileName,
		selectedMCP:   make(map[string]bool),
		selectedSkill: make(map[string]bool),
		selectedPerm:  make(map[string]bool),
		selectedHook:  make(map[string]bool),
	}

	// Pre-load registry items
	for _, s := range svc.ListServers() {
		w.allMCPs = append(w.allMCPs, s.Name)
	}
	for _, s := range svc.ListSkills() {
		w.allSkills = append(w.allSkills, s.Name)
	}
	for _, p := range svc.ListPermissions() {
		w.allPerms = append(w.allPerms, p.Name)
	}
	for _, h := range svc.ListHooks() {
		w.allHooks = append(w.allHooks, h.Name)
	}

	// Pre-select from existing profile
	if prof, err := svc.LoadProfile(profileName); err == nil {
		for _, a := range prof.MCPs {
			w.selectedMCP[a.Name] = true
		}
		for _, n := range prof.Skills {
			w.selectedSkill[n] = true
		}
		for _, n := range prof.Permissions {
			w.selectedPerm[n] = true
		}
		for _, n := range prof.Hooks {
			w.selectedHook[n] = true
		}
	}

	return w
}

func (w wizardModel) helpKeys() []HelpEntry {
	switch w.step {
	case wizardStepReview:
		return []HelpEntry{{"Enter", "Launch"}, {"Esc", "Cancel"}}
	default:
		return []HelpEntry{{"Space", "Toggle"}, {"Enter", "Next"}, {"Esc", "Cancel"}}
	}
}

func (w wizardModel) update(msg tea.KeyMsg) (wizardModel, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		return w, func() tea.Msg { return toolsWizardDoneMsg{} }
	case key.Matches(msg, w.keys.ListUp):
		if w.cursor > 0 {
			w.cursor--
		}
	case key.Matches(msg, w.keys.ListDown):
		max := w.currentListLen() - 1
		if w.cursor < max {
			w.cursor++
		}
	case key.Matches(msg, w.keys.Select):
		w.toggleCurrent()
	case key.Matches(msg, w.keys.Confirm):
		if w.step < wizardStepReview {
			w.step++
			w.cursor = 0
		} else {
			return w, w.saveAndFinish()
		}
	case msg.String() == "backspace" || msg.String() == "left":
		if w.step > wizardStepMCPs {
			w.step--
			w.cursor = 0
		}
	}
	return w, nil
}

func (w *wizardModel) toggleCurrent() {
	switch w.step {
	case wizardStepMCPs:
		if w.cursor < len(w.allMCPs) {
			name := w.allMCPs[w.cursor]
			w.selectedMCP[name] = !w.selectedMCP[name]
		}
	case wizardStepOptions:
		all, sel := w.currentOptionList()
		if w.cursor < len(all) {
			name := all[w.cursor]
			sel[name] = !sel[name]
		}
	}
}

func (w wizardModel) currentListLen() int {
	switch w.step {
	case wizardStepMCPs:
		return len(w.allMCPs)
	case wizardStepOptions:
		all, _ := w.currentOptionList()
		return len(all)
	case wizardStepReview:
		return 0
	}
	return 0
}

func (w wizardModel) currentOptionList() ([]string, map[string]bool) {
	// Flatten all option sections into one list for now
	all := make([]string, 0, len(w.allSkills)+len(w.allPerms)+len(w.allHooks))
	all = append(all, w.allSkills...)
	all = append(all, w.allPerms...)
	all = append(all, w.allHooks...)

	merged := make(map[string]bool)
	for k, v := range w.selectedSkill {
		merged[k] = v
	}
	for k, v := range w.selectedPerm {
		merged[k] = v
	}
	for k, v := range w.selectedHook {
		merged[k] = v
	}
	return all, merged
}

func (w wizardModel) saveAndFinish() tea.Cmd {
	svc := w.svc
	profileName := w.profileName
	selMCP := copyBoolMap(w.selectedMCP)
	selSkill := copyBoolMap(w.selectedSkill)
	selPerm := copyBoolMap(w.selectedPerm)
	selHook := copyBoolMap(w.selectedHook)

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
		prof.Skills = nil
		for name, sel := range selSkill {
			if sel {
				prof.Skills = append(prof.Skills, name)
			}
		}
		prof.Permissions = nil
		for name, sel := range selPerm {
			if sel {
				prof.Permissions = append(prof.Permissions, name)
			}
		}
		prof.Hooks = nil
		for name, sel := range selHook {
			if sel {
				prof.Hooks = append(prof.Hooks, name)
			}
		}

		// Sort for deterministic YAML output (CS-10)
		sort.Slice(prof.MCPs, func(i, j int) bool { return prof.MCPs[i].Name < prof.MCPs[j].Name })
		sort.Strings(prof.Skills)
		sort.Strings(prof.Permissions)
		sort.Strings(prof.Hooks)

		if err := svc.SaveProfile(prof); err != nil {
			return toolsLaunchDoneMsg{err: err}
		}
		return toolsWizardDoneMsg{}
	}
}

func (w wizardModel) view(width, height int) string {
	var b strings.Builder

	// Step indicator
	steps := [wizardStepCount]string{"MCPs", "Options", "Review"}
	b.WriteString("  Launch Wizard — Step " + fmt.Sprintf("%d", w.step+1) + " of 3: " + steps[w.step] + "\n\n")

	switch w.step {
	case wizardStepMCPs:
		b.WriteString("  Select MCP servers:\n\n")
		for i, name := range w.allMCPs {
			marker := "  "
			if w.selectedMCP[name] {
				marker = styleSynced.Render("x ")
			}
			line := "  " + marker + name
			if i == w.cursor {
				b.WriteString(styleListSelected.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
		if len(w.allMCPs) == 0 {
			b.WriteString("  (no servers in registry)\n")
		}

	case wizardStepOptions:
		sections := []struct {
			title string
			items []string
			sel   map[string]bool
		}{
			{"Skills", w.allSkills, w.selectedSkill},
			{"Permissions", w.allPerms, w.selectedPerm},
			{"Hooks", w.allHooks, w.selectedHook},
		}
		idx := 0
		for _, sec := range sections {
			count := 0
			for _, n := range sec.items {
				if sec.sel[n] {
					count++
				}
			}
			b.WriteString(styleListHeader.Render(fmt.Sprintf("  %s (%d selected)", sec.title, count)) + "\n")
			for _, name := range sec.items {
				marker := "  "
				if sec.sel[name] {
					marker = styleSynced.Render("x ")
				}
				line := "    " + marker + name
				if idx == w.cursor {
					b.WriteString(styleListSelected.Render(line))
				} else {
					b.WriteString(line)
				}
				b.WriteString("\n")
				idx++
			}
			if len(sec.items) == 0 {
				b.WriteString("    (none)\n")
			}
			b.WriteString("\n")
		}

	case wizardStepReview:
		b.WriteString("  Profile: " + w.profileName + "\n")
		b.WriteString("  Project: " + w.projectName + "\n\n")

		mcpCount := countTrue(w.selectedMCP)
		skillCount := countTrue(w.selectedSkill)
		permCount := countTrue(w.selectedPerm)
		hookCount := countTrue(w.selectedHook)

		fmt.Fprintf(&b, "  MCPs          %d\n", mcpCount)
		fmt.Fprintf(&b, "  Skills        %d\n", skillCount)
		fmt.Fprintf(&b, "  Permissions   %d\n", permCount)
		fmt.Fprintf(&b, "  Hooks         %d\n", hookCount)
		b.WriteString("\n  Ready to save profile and launch.\n")
	}

	b.WriteString("\n  ")
	if w.step > 0 {
		b.WriteString(styleHelpKey.Render("Left") + styleHelpDesc.Render(":Back  "))
	}
	if w.step < wizardStepReview {
		b.WriteString(styleHelpKey.Render("Space") + styleHelpDesc.Render(":Toggle  "))
		b.WriteString(styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Next  "))
	} else {
		b.WriteString(styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Launch  "))
	}
	b.WriteString(styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	content := b.String()
	box := styleOverlayBorder.Width(min(width-4, 60)).Render(content)
	return centerOverlay(box, width, height)
}

func countTrue(m map[string]bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

func copyBoolMap(m map[string]bool) map[string]bool {
	cp := make(map[string]bool, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
