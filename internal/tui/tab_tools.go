package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/service"
)

// toolAction identifies which tool action is selected.
type toolAction int

const (
	toolSync toolAction = iota
	toolDiff
	toolDiscover
	toolLaunch
	toolCount
)

var toolDefs = [toolCount]struct {
	name string
	desc string
}{
	{"Sync", "Deploy project configs"},
	{"Diff", "Show config drift"},
	{"Discover", "Scan for new MCPs"},
	{"Launch", "Sync + run Claude Code"},
}

// toolsMode tracks the current state of the tools tab.
type toolsMode int

const (
	toolsModeGrid     toolsMode = iota
	toolsModePicker             // project picker before action
	toolsModeDiff               // showing diff results
	toolsModeDiscover           // showing discovery results
	toolsModeWizard             // launch wizard
)

// toolsTab is the Tools tab — action grid with overlays.
type toolsTab struct {
	keys   KeyMap
	svc    *service.Service
	cursor int
	width  int
	height int
	mode   toolsMode
	err    string

	// Project picker
	projects      []string
	projectCursor int
	pendingAction toolAction

	// Diff view
	diffView diffViewModel

	// Discovery view
	discoverView discoveryModel

	// Wizard
	wizard wizardModel
}

func newToolsTab(keys KeyMap, svc *service.Service) *toolsTab {
	return &toolsTab{keys: keys, svc: svc}
}

func (t *toolsTab) Title() string { return "Tools" }

func (t *toolsTab) HelpKeys() []HelpEntry {
	switch t.mode {
	case toolsModePicker:
		return []HelpEntry{{"Enter", "Select"}, {"Esc", "Cancel"}}
	case toolsModeDiff:
		return t.diffView.helpKeys()
	case toolsModeDiscover:
		return t.discoverView.helpKeys()
	case toolsModeWizard:
		return t.wizard.helpKeys()
	default:
		return []HelpEntry{{"Enter", "Select"}, {"Arrow", "Navigate"}}
	}
}

// --- Messages ---

type toolsProjectListMsg struct {
	projects []string
}
type toolsSyncDoneMsg struct {
	results []service.SyncResult
	err     error
}
type toolsDiffDoneMsg struct {
	results []service.DiffResult
	err     error
}
type toolsDiscoverDoneMsg struct {
	imported int
	err      error
}
type toolsLaunchDoneMsg struct{ err error }
type toolsWizardDoneMsg struct{}

// --- Init / Update ---

func (t *toolsTab) Init() tea.Cmd { return nil }

func (t *toolsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case toolsProjectListMsg:
		t.projects = msg.projects
		t.projectCursor = 0
		return t, nil

	case toolsSyncDoneMsg:
		if msg.err != nil {
			t.err = msg.err.Error()
			t.mode = toolsModeGrid
		} else {
			t.err = fmt.Sprintf("Sync complete: %d server(s) processed", len(msg.results))
			t.mode = toolsModeGrid
		}
		return t, nil

	case toolsDiffDoneMsg:
		if msg.err != nil {
			t.err = msg.err.Error()
			t.mode = toolsModeGrid
		} else {
			t.diffView = newDiffViewModel(msg.results, t.keys)
			t.mode = toolsModeDiff
		}
		return t, nil

	case toolsDiscoverDoneMsg:
		if msg.err != nil {
			t.err = msg.err.Error()
			t.mode = toolsModeGrid
		} else {
			t.err = fmt.Sprintf("Discovery complete: %d new server(s) imported", msg.imported)
			t.mode = toolsModeGrid
		}
		return t, nil

	case toolsLaunchDoneMsg:
		if msg.err != nil {
			t.err = msg.err.Error()
		}
		t.mode = toolsModeGrid
		return t, nil

	case toolsWizardDoneMsg:
		t.mode = toolsModeGrid
		return t, nil

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		switch t.mode {
		case toolsModePicker:
			return t.handlePickerKey(msg)
		case toolsModeDiff:
			return t.handleDiffKey(msg)
		case toolsModeDiscover:
			return t.handleDiscoverKey(msg)
		case toolsModeWizard:
			return t.handleWizardKey(msg)
		default:
			return t.handleGridKey(msg)
		}
	}
	return t, nil
}

// --- Grid mode ---

func (t *toolsTab) handleGridKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cols := 2
	switch {
	case key.Matches(msg, t.keys.ListUp):
		if t.cursor >= cols {
			t.cursor -= cols
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.cursor+cols < int(toolCount) {
			t.cursor += cols
		}
	case msg.String() == "left":
		if t.cursor%cols > 0 {
			t.cursor--
		}
	case msg.String() == "right":
		if t.cursor%cols < cols-1 && t.cursor+1 < int(toolCount) {
			t.cursor++
		}
	case key.Matches(msg, t.keys.Confirm):
		return t, t.activateTool(toolAction(t.cursor))
	}
	return t, nil
}

func (t *toolsTab) activateTool(action toolAction) tea.Cmd {
	t.pendingAction = action
	t.mode = toolsModePicker
	svc := t.svc
	return func() tea.Msg {
		projects := svc.ListProjects()
		names := make([]string, len(projects))
		for i, p := range projects {
			names[i] = p.Name
		}
		return toolsProjectListMsg{projects: names}
	}
}

// --- Project picker ---

func (t *toolsTab) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		t.mode = toolsModeGrid
	case key.Matches(msg, t.keys.ListUp):
		if t.projectCursor > 0 {
			t.projectCursor--
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.projectCursor < len(t.projects)-1 {
			t.projectCursor++
		}
	case key.Matches(msg, t.keys.Confirm):
		if len(t.projects) > 0 {
			return t, t.executeAction(t.projects[t.projectCursor])
		}
	}
	return t, nil
}

func (t *toolsTab) executeAction(projectName string) tea.Cmd {
	svc := t.svc
	action := t.pendingAction

	switch action {
	case toolSync:
		return func() tea.Msg {
			results, err := svc.SyncProject(projectName)
			return toolsSyncDoneMsg{results: results, err: err}
		}
	case toolDiff:
		return func() tea.Msg {
			results, err := svc.DiffProject(projectName)
			return toolsDiffDoneMsg{results: results, err: err}
		}
	case toolDiscover:
		return func() tea.Msg {
			proj, ok := svc.GetProject(projectName)
			if !ok {
				return toolsDiscoverDoneMsg{err: fmt.Errorf("project %q not found", projectName)}
			}
			imported, err := svc.AutoDiscover(proj.Path)
			return toolsDiscoverDoneMsg{imported: len(imported), err: err}
		}
	case toolLaunch:
		return func() tea.Msg {
			// Sync before launch
			if _, err := svc.SyncProject(projectName); err != nil {
				return toolsLaunchDoneMsg{err: fmt.Errorf("sync failed: %w", err)}
			}
			// Actual launch requires exiting alt screen; direct user to CLI
			return toolsLaunchDoneMsg{err: fmt.Errorf("launch requires exiting TUI — use 'hystak run %s' from CLI", projectName)}
		}
	}
	return nil
}

// --- Diff overlay ---

func (t *toolsTab) handleDiffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		t.mode = toolsModeGrid
		return t, nil
	}
	t.diffView = t.diffView.update(msg)
	return t, nil
}

// --- Discover overlay ---

func (t *toolsTab) handleDiscoverKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		t.mode = toolsModeGrid
		return t, nil
	}
	return t, nil
}

// --- Wizard ---

func (t *toolsTab) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		t.mode = toolsModeGrid
		return t, nil
	}
	return t, nil
}

// --- View ---

func (t *toolsTab) View() string {
	switch t.mode {
	case toolsModePicker:
		return t.viewPicker()
	case toolsModeDiff:
		return t.diffView.view(t.width, t.height)
	}

	var b strings.Builder
	cols := 2
	colWidth := 28

	if t.err != "" {
		b.WriteString("  " + styleError.Render(t.err) + "\n\n")
	}

	for row := 0; row < (int(toolCount)+1)/cols; row++ {
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= int(toolCount) {
				break
			}
			name := fmt.Sprintf("  %-*s", colWidth-2, toolDefs[idx].name)
			if idx == t.cursor {
				b.WriteString(styleListSelected.Render(name))
			} else {
				b.WriteString(styleTitle.Render(name))
			}
			b.WriteString("  ")
		}
		b.WriteString("\n")

		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= int(toolCount) {
				break
			}
			desc := fmt.Sprintf("  %-*s", colWidth-2, toolDefs[idx].desc)
			if idx == t.cursor {
				b.WriteString(styleListSelected.Render(desc))
			} else {
				b.WriteString(styleHelpDesc.Render(desc))
			}
			b.WriteString("  ")
		}
		b.WriteString("\n\n")
	}

	return b.String()
}

func (t *toolsTab) viewPicker() string {
	var b strings.Builder
	action := toolDefs[t.pendingAction].name
	b.WriteString("Select project for " + action + "\n\n")

	if len(t.projects) == 0 {
		b.WriteString("  (no projects registered)\n")
	}
	for i, name := range t.projects {
		line := "  " + name
		if i == t.projectCursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n  " +
		styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Select  ") +
		styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	content := b.String()
	box := styleOverlayBorder.Width(min(t.width-4, 50)).Render(content)
	return centerOverlay(box, t.width, t.height)
}
