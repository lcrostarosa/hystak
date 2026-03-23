package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

// projectsPane identifies which pane has focus.
type projectsPane int

const (
	paneProjects projectsPane = iota
	paneDetail
)

// detailSection identifies which section in the detail pane is active.
type detailSection int

const (
	sectionProfiles detailSection = iota
	sectionMCPs
	sectionSkills
	sectionHooks
	sectionPermissions
	sectionTemplate
	detailSectionCount
)

var detailSectionNames = [detailSectionCount]string{
	"Profiles", "MCPs", "Skills", "Hooks", "Permissions", "Template",
}

// projectsTab is the Projects tab — two-pane layout per wireframe.
type projectsTab struct {
	keys   KeyMap
	svc    *service.Service
	width  int
	height int
	err    string

	// Left pane
	projects []model.Project
	cursor   int

	// Right pane
	pane          projectsPane
	section       detailSection
	sectionCursor int

	// Detail data (loaded for the selected project)
	profile      model.ProjectProfile
	profileNames []string
	allMCPs      []string
	allSkills    []string
	allHooks     []string
	allPerms     []string
	allTemplates []string

	// Confirm overlay for profile switch
	mode        projectsMode
	confirm     confirmModel
	profilePick int // cursor in profile picker
}

type projectsMode int

const (
	projectsModeNormal projectsMode = iota
	projectsModeProfilePicker
	projectsModeConfirm
)

func newProjectsTab(keys KeyMap, svc *service.Service) *projectsTab {
	return &projectsTab{
		keys: keys,
		svc:  svc,
	}
}

func (t *projectsTab) Title() string { return "Projects" }

func (t *projectsTab) HelpKeys() []HelpEntry {
	if t.mode == projectsModeProfilePicker {
		return []HelpEntry{{"Enter", "Activate"}, {"Esc", "Cancel"}}
	}
	if t.pane == paneProjects {
		return []HelpEntry{
			{"Enter/Right", "Detail"},
			{"P", "Profile"},
			{"S", "Sync"},
		}
	}
	return []HelpEntry{
		{"Left/Esc", "Back"},
		{"Space", "Toggle"},
		{"P", "Profile"},
	}
}

// --- Messages ---

type projectsLoadedMsg struct {
	projects []model.Project
}

type projectDetailMsg struct {
	profile      model.ProjectProfile
	profileNames []string
	allMCPs      []string
	allSkills    []string
	allHooks     []string
	allPerms     []string
	allTemplates []string
	err          error
}

type profileSwitchedMsg struct{}
type toggledMsg struct{}
type projectsErrorMsg struct{ err error }

// --- Init / Load ---

func (t *projectsTab) Init() tea.Cmd {
	return t.loadProjects
}

func (t *projectsTab) loadProjects() tea.Msg {
	return projectsLoadedMsg{projects: t.svc.ListProjects()}
}

func (t *projectsTab) loadDetail() tea.Cmd {
	if len(t.projects) == 0 {
		return nil
	}
	proj := t.projects[t.cursor]
	svc := t.svc
	profileName := proj.ActiveProfile
	return func() tea.Msg {
		var prof model.ProjectProfile
		var loadErr error
		if profileName != "" {
			p, err := svc.LoadProfile(profileName)
			if err != nil {
				loadErr = fmt.Errorf("loading profile %q: %w", profileName, err)
			} else {
				prof = p
			}
		}
		names, err := svc.ListProfileNames()
		if err != nil {
			loadErr = fmt.Errorf("listing profiles: %w", err)
		}
		return projectDetailMsg{
			profile:      prof,
			profileNames: names,
			allMCPs:      resourceNames(svc.ListServers()),
			allSkills:    resourceNames(svc.ListSkills()),
			allHooks:     resourceNames(svc.ListHooks()),
			allPerms:     resourceNames(svc.ListPermissions()),
			allTemplates: resourceNames(svc.ListTemplates()),
			err:          loadErr,
		}
	}
}

// --- Update ---

func (t *projectsTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsLoadedMsg:
		t.projects = msg.projects
		if t.cursor >= len(t.projects) {
			t.cursor = max(0, len(t.projects)-1)
		}
		t.err = ""
		return t, t.loadDetail()

	case projectDetailMsg:
		t.profile = msg.profile
		t.profileNames = msg.profileNames
		t.allMCPs = msg.allMCPs
		t.allSkills = msg.allSkills
		t.allHooks = msg.allHooks
		t.allPerms = msg.allPerms
		t.allTemplates = msg.allTemplates
		t.sectionCursor = 0
		if msg.err != nil {
			t.err = msg.err.Error()
		}
		return t, nil

	case profileSwitchedMsg:
		t.mode = projectsModeNormal
		return t, t.loadProjects

	case toggledMsg:
		return t, t.loadDetail()

	case projectsErrorMsg:
		t.err = msg.err.Error()
		t.mode = projectsModeNormal
		return t, nil

	case confirmYesMsg:
		return t, t.handleProfileConfirm()

	case confirmNoMsg:
		t.mode = projectsModeNormal
		return t, nil

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		switch t.mode {
		case projectsModeProfilePicker:
			return t.handleProfilePickerKey(msg)
		case projectsModeConfirm:
			var cmd tea.Cmd
			t.confirm, cmd = t.confirm.Update(msg)
			return t, cmd
		default:
			return t.handleKey(msg)
		}
	}
	return t, nil
}

// --- Key handlers ---

func (t *projectsTab) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if t.pane == paneProjects {
		return t.handleProjectsKey(msg)
	}
	return t.handleDetailKey(msg)
}

func (t *projectsTab) handleProjectsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, t.keys.ListUp):
		if t.cursor > 0 {
			t.cursor--
			return t, t.loadDetail()
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.cursor < len(t.projects)-1 {
			t.cursor++
			return t, t.loadDetail()
		}
	case key.Matches(msg, t.keys.Confirm) || msg.String() == "right":
		if len(t.projects) > 0 {
			t.pane = paneDetail
			t.section = sectionProfiles
			t.sectionCursor = 0
		}
	case key.Matches(msg, t.keys.Preview): // P = profile picker
		if len(t.projects) > 0 {
			t.openProfilePicker()
		}
	}
	return t, nil
}

func (t *projectsTab) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "left" || msg.String() == "esc":
		t.pane = paneProjects
		return t, nil
	case key.Matches(msg, t.keys.ListUp):
		if t.sectionCursor > 0 {
			t.sectionCursor--
		} else if t.section > 0 {
			t.section--
			t.sectionCursor = t.sectionItemCount() - 1
			if t.sectionCursor < 0 {
				t.sectionCursor = 0
			}
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.sectionCursor < t.sectionItemCount()-1 {
			t.sectionCursor++
		} else if t.section < detailSectionCount-1 {
			t.section++
			t.sectionCursor = 0
		}
	case key.Matches(msg, t.keys.Select):
		return t, t.toggleCurrent()
	case key.Matches(msg, t.keys.Preview): // P = profile picker
		t.openProfilePicker()
	}
	return t, nil
}

// --- Profile picker ---

func (t *projectsTab) openProfilePicker() {
	t.mode = projectsModeProfilePicker
	t.profilePick = 0
	// Find current active profile in the list
	proj := t.projects[t.cursor]
	for i, name := range t.profileNames {
		if name == proj.ActiveProfile {
			t.profilePick = i
			break
		}
	}
}

func (t *projectsTab) handleProfilePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		t.mode = projectsModeNormal
	case key.Matches(msg, t.keys.ListUp):
		if t.profilePick > 0 {
			t.profilePick--
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.profilePick < len(t.profileNames)-1 {
			t.profilePick++
		}
	case key.Matches(msg, t.keys.Confirm):
		return t, t.handleProfileConfirm()
	}
	return t, nil
}

func (t *projectsTab) handleProfileConfirm() tea.Cmd {
	if len(t.profileNames) == 0 || len(t.projects) == 0 {
		return nil
	}
	projName := t.projects[t.cursor].Name
	profName := t.profileNames[t.profilePick]
	svc := t.svc
	return func() tea.Msg {
		if err := svc.SetActiveProfile(projName, profName); err != nil {
			return projectsErrorMsg{err: err}
		}
		return profileSwitchedMsg{}
	}
}

// --- Toggle resources ---

func (t *projectsTab) toggleCurrent() tea.Cmd {
	if len(t.projects) == 0 {
		return nil
	}
	proj := t.projects[t.cursor]
	if proj.ActiveProfile == "" {
		return nil
	}
	profName := proj.ActiveProfile
	svc := t.svc

	switch t.section {
	case sectionMCPs:
		if t.sectionCursor < len(t.allMCPs) {
			name := t.allMCPs[t.sectionCursor]
			return func() tea.Msg {
				if _, err := svc.ToggleMCP(profName, name); err != nil {
					return projectsErrorMsg{err: err}
				}
				return toggledMsg{}
			}
		}
	case sectionSkills:
		if t.sectionCursor < len(t.allSkills) {
			name := t.allSkills[t.sectionCursor]
			return func() tea.Msg {
				if _, err := svc.ToggleProfileResource(profName, "skills", name); err != nil {
					return projectsErrorMsg{err: err}
				}
				return toggledMsg{}
			}
		}
	case sectionHooks:
		if t.sectionCursor < len(t.allHooks) {
			name := t.allHooks[t.sectionCursor]
			return func() tea.Msg {
				if _, err := svc.ToggleProfileResource(profName, "hooks", name); err != nil {
					return projectsErrorMsg{err: err}
				}
				return toggledMsg{}
			}
		}
	case sectionPermissions:
		if t.sectionCursor < len(t.allPerms) {
			name := t.allPerms[t.sectionCursor]
			return func() tea.Msg {
				if _, err := svc.ToggleProfileResource(profName, "permissions", name); err != nil {
					return projectsErrorMsg{err: err}
				}
				return toggledMsg{}
			}
		}
	case sectionTemplate:
		if t.sectionCursor < len(t.allTemplates) {
			tmplName := t.allTemplates[t.sectionCursor]
			return func() tea.Msg {
				// Toggle: if already set, clear; otherwise set
				prof, err := svc.LoadProfile(profName)
				if err != nil {
					return projectsErrorMsg{err: err}
				}
				newTmpl := tmplName
				if prof.Template == tmplName {
					newTmpl = ""
				}
				if err := svc.SetProfileTemplate(profName, newTmpl); err != nil {
					return projectsErrorMsg{err: err}
				}
				return toggledMsg{}
			}
		}
	}
	return nil
}

func (t *projectsTab) sectionItemCount() int {
	switch t.section {
	case sectionProfiles:
		return len(t.profileNames)
	case sectionMCPs:
		return len(t.allMCPs)
	case sectionSkills:
		return len(t.allSkills)
	case sectionHooks:
		return len(t.allHooks)
	case sectionPermissions:
		return len(t.allPerms)
	case sectionTemplate:
		return len(t.allTemplates)
	}
	return 0
}

// --- View ---

func (t *projectsTab) View() string {
	if t.mode == projectsModeProfilePicker {
		return t.viewProfilePicker()
	}

	if len(t.projects) == 0 {
		return "  No projects registered.\n  Use 'hystak setup' or add a project.\n"
	}

	var b strings.Builder

	if t.err != "" {
		b.WriteString("  " + styleError.Render("Error: "+t.err) + "\n\n")
	}

	leftWidth := 26
	left := t.viewProjectList(leftWidth)
	right := t.viewDetail()

	// Side-by-side layout
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	maxLines := max(len(leftLines), len(rightLines))

	for i := 0; i < maxLines; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		// Pad left column
		padded := l + strings.Repeat(" ", max(0, leftWidth-visibleLen(l)))
		b.WriteString(padded + " | " + r + "\n")
	}

	return b.String()
}

func (t *projectsTab) viewProjectList(width int) string {
	var b strings.Builder
	b.WriteString(styleListHeader.Render("  PROJECTS") + "\n")

	for i, p := range t.projects {
		indicator := "  "
		if p.ActiveProfile != "" {
			indicator = styleSynced.Render("* ")
		}
		name := truncate(p.Name, width-4)
		line := indicator + name
		if i == t.cursor && t.pane == paneProjects {
			b.WriteString(styleListSelected.Render(line))
		} else if i == t.cursor {
			b.WriteString(styleTitle.Render(line))
		} else {
			b.WriteString(styleListNormal.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (t *projectsTab) viewDetail() string {
	if len(t.projects) == 0 {
		return ""
	}

	proj := t.projects[t.cursor]
	var b strings.Builder
	b.WriteString(styleTitle.Render(proj.Name) + "\n")
	b.WriteString(styleHelpDesc.Render("  Path: "+truncate(proj.Path, 50)) + "\n")
	b.WriteString(styleHelpDesc.Render("  Active: "+proj.ActiveProfile) + "\n\n")

	// Profile assignments from the loaded profile
	mcpSet := makeSet(mcpNames(t.profile.MCPs))
	skillSet := makeSet(t.profile.Skills)
	hookSet := makeSet(t.profile.Hooks)
	permSet := makeSet(t.profile.Permissions)

	sections := []struct {
		sec   detailSection
		title string
		all   []string
		sel   map[string]bool
	}{
		{sectionProfiles, "Profiles", t.profileNames, nil},
		{sectionMCPs, "MCPs", t.allMCPs, mcpSet},
		{sectionSkills, "Skills", t.allSkills, skillSet},
		{sectionHooks, "Hooks", t.allHooks, hookSet},
		{sectionPermissions, "Permissions", t.allPerms, permSet},
		{sectionTemplate, "Template", t.allTemplates, nil},
	}

	for _, sec := range sections {
		count := ""
		if sec.sel != nil {
			n := 0
			for _, name := range sec.all {
				if sec.sel[name] {
					n++
				}
			}
			count = fmt.Sprintf(" (%d)", n)
		}
		b.WriteString(styleListHeader.Render("  "+sec.title+count) + "\n")

		if len(sec.all) == 0 {
			b.WriteString("    (none)\n")
		}

		for i, name := range sec.all {
			isActive := t.pane == paneDetail && t.section == sec.sec && t.sectionCursor == i

			var marker string
			switch sec.sec {
			case sectionProfiles:
				if name == proj.ActiveProfile {
					marker = styleSynced.Render("● ")
				} else {
					marker = "○ "
				}
			case sectionTemplate:
				if name == t.profile.Template {
					marker = styleSynced.Render("● ")
				} else {
					marker = "○ "
				}
			default:
				if sec.sel != nil && sec.sel[name] {
					marker = styleSynced.Render("x ")
				} else {
					marker = "  "
				}
			}

			line := "    " + marker + name
			if isActive {
				b.WriteString(styleListSelected.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (t *projectsTab) viewProfilePicker() string {
	if len(t.projects) == 0 {
		return ""
	}
	proj := t.projects[t.cursor]

	var b strings.Builder
	b.WriteString("Select Profile for " + proj.Name + "\n\n")
	for i, name := range t.profileNames {
		marker := "  "
		if name == proj.ActiveProfile {
			marker = styleSynced.Render("● ")
		}
		line := "  " + marker + name
		if i == t.profilePick {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n  " +
		styleHelpKey.Render("Enter") + styleHelpDesc.Render(":Activate  ") +
		styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Cancel"))

	content := b.String()
	box := styleOverlayBorder.Width(min(t.width-4, 50)).Render(content)
	return centerOverlay(box, t.width, t.height)
}

// --- Helpers ---

func mcpNames(mcps []model.MCPAssignment) []string {
	names := make([]string, len(mcps))
	for i, a := range mcps {
		names[i] = a.Name
	}
	return names
}

func makeSet(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// resourceNames extracts names from any slice of Resource-implementing types.
func resourceNames[T any, PT interface {
	*T
	model.Resource
}](items []T) []string {
	names := make([]string, len(items))
	for i := range items {
		p := PT(&items[i])
		names[i] = p.ResourceName()
	}
	return names
}

// visibleLen returns the visible length of a string (ignoring ANSI codes).
// Simplified: counts runes. For styled strings, lipgloss.Width is more accurate
// but this is sufficient for padding.
func visibleLen(s string) int {
	return len([]rune(s))
}
