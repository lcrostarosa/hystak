package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// paneFocus identifies which pane has keyboard focus in the profiles tab.
type paneFocus int

const (
	focusLeft paneFocus = iota
	focusRight
)

// rightSection identifies which section of the right pane is active.
type rightSection int

const (
	sectionMCPs rightSection = iota
	sectionSkills
	sectionHooks
	sectionPermissions
	sectionTemplate
	sectionCount
)

// profileItem implements list.DefaultItem for the profile list.
type profileItem struct {
	project     model.Project
	serverCount int
}

func (i profileItem) Title() string {
	if i.serverCount > 0 {
		return fmt.Sprintf("%s [%d]", i.project.Name, i.serverCount)
	}
	return i.project.Name
}

func (i profileItem) Description() string {
	parts := []string{}
	if n := i.serverCount; n > 0 {
		parts = append(parts, fmt.Sprintf("%d MCPs", n))
	}
	if n := len(i.project.Skills); n > 0 {
		parts = append(parts, fmt.Sprintf("%d skills", n))
	}
	if n := len(i.project.Hooks); n > 0 {
		parts = append(parts, fmt.Sprintf("%d hooks", n))
	}
	summary := strings.Join(parts, ", ")
	if i.project.Path != "" {
		if summary != "" {
			return i.project.Path + " (" + summary + ")"
		}
		return i.project.Path
	}
	if summary != "" {
		return summary
	}
	return "no path"
}

func (i profileItem) FilterValue() string { return i.project.Name }

// ProfileDeletedMsg is sent when a profile has been deleted.
type ProfileDeletedMsg struct{ Name string }

// AutoSyncResultMsg is sent when an auto-sync completes after a toggle.
type AutoSyncResultMsg struct {
	ProjectName string
	Err         error
	Results     []service.SyncResult
}

// RequestLaunchMsg is sent when the user wants to launch a profile.
type RequestLaunchMsg struct {
	ProfileName string
}

// ProfilesModel is the sub-model for the Profiles tab.
type ProfilesModel struct {
	list    list.Model
	service *service.Service
	keys    KeyMap
	width   int
	height  int

	confirming bool
	err        error
	syncMsg    string

	focus          paneFocus
	activeSection  rightSection
	sectionCursors [sectionCount]int

	// All resource names from registry (sorted).
	allMCPs        []string
	allSkills      []string
	allHooks       []string
	allPermissions []string
	allTemplates   []string

	// Track selection for ProfileSelectionChangedMsg.
	lastSelectedName string
}

// NewProfilesModel creates a new ProfilesModel.
func NewProfilesModel(svc *service.Service, keys KeyMap) ProfilesModel {
	items := buildProfileItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return ProfilesModel{
		list:           l,
		service:        svc,
		keys:           keys,
		allMCPs:        buildAllServerNames(svc),
		allSkills:      buildAllSkillNames(svc),
		allHooks:       buildAllHookNames(svc),
		allPermissions: buildAllPermissionNames(svc),
		allTemplates:   buildAllTemplateNames(svc),
	}
}

func buildProfileItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}

	projects := svc.ListProjects()
	items := make([]list.Item, len(projects))
	for i, proj := range projects {
		items[i] = profileItem{
			project:     proj,
			serverCount: svc.CountAssignedServers(proj),
		}
	}
	return items
}

func buildAllServerNames(svc *service.Service) []string {
	if svc == nil {
		return nil
	}
	servers := svc.ListServers()
	names := make([]string, len(servers))
	for i, srv := range servers {
		names[i] = srv.Name
	}
	sort.Strings(names)
	return names
}

func buildAllSkillNames(svc *service.Service) []string {
	if svc == nil {
		return nil
	}
	skills := svc.ListSkills()
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	sort.Strings(names)
	return names
}

func buildAllHookNames(svc *service.Service) []string {
	if svc == nil {
		return nil
	}
	hooks := svc.ListHooks()
	names := make([]string, len(hooks))
	for i, h := range hooks {
		names[i] = h.Name
	}
	sort.Strings(names)
	return names
}

func buildAllPermissionNames(svc *service.Service) []string {
	if svc == nil {
		return nil
	}
	perms := svc.ListPermissions()
	names := make([]string, len(perms))
	for i, p := range perms {
		names[i] = p.Name
	}
	sort.Strings(names)
	return names
}

func buildAllTemplateNames(svc *service.Service) []string {
	if svc == nil {
		return nil
	}
	tmpls := svc.ListTemplates()
	names := make([]string, len(tmpls))
	for i, t := range tmpls {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names
}

func (m ProfilesModel) selectedProfile() (model.Project, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.Project{}, false
	}
	pi, ok := item.(profileItem)
	if !ok {
		return model.Project{}, false
	}
	return pi.project, true
}

// isServerAssigned checks if a server is assigned to the given profile
// (either directly via MCPs or via tag expansion).
func (m ProfilesModel) isServerAssigned(proj model.Project, serverName string) bool {
	if m.service == nil {
		return false
	}
	return m.service.IsServerAssigned(proj, serverName)
}

// isServerFromTag checks if a server's assignment comes only from tag expansion
// (not from a direct MCP entry).
func (m ProfilesModel) isServerFromTag(proj model.Project, serverName string) bool {
	if m.service == nil {
		return false
	}
	return m.service.IsServerFromTag(proj, serverName)
}

// sectionLen returns the number of items in a section for the given project.
func (m ProfilesModel) sectionLen(sec rightSection) int {
	switch sec {
	case sectionMCPs:
		return len(m.allMCPs)
	case sectionSkills:
		return len(m.allSkills)
	case sectionHooks:
		return len(m.allHooks)
	case sectionPermissions:
		return len(m.allPermissions)
	case sectionTemplate:
		return len(m.allTemplates)
	}
	return 0
}

// IsConsuming returns true when the model handles its own input
// (e.g., filtering, confirming, or navigating the right pane).
func (m ProfilesModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming || m.focus == focusRight
}

// SetSize updates the dimensions available to the profiles tab.
func (m *ProfilesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m ProfilesModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	if m.focus == focusRight {
		return "space: toggle | tab/S-tab: section | esc: back"
	}
	return fmt.Sprintf("%s: launch | %s: configure | %s: delete | /: filter | %s | q: quit",
		m.keys.ProfileLaunch.Help().Key,
		m.keys.ProfileConfigure.Help().Key,
		m.keys.ProfileDelete.Help().Key,
		m.keys.tabNavHelp())
}

// autoSync triggers an async sync for the given project.
func (m *ProfilesModel) autoSync(projName string) tea.Cmd {
	svc := m.service
	return func() tea.Msg {
		results, err := svc.SyncProject(projName)
		return AutoSyncResultMsg{ProjectName: projName, Err: err, Results: results}
	}
}

// Update handles messages for the profiles tab.
func (m ProfilesModel) Update(msg tea.Msg) (ProfilesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case AutoSyncResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			// Build a summary: count MCPs, skills, etc. from results.
			mcpCount := 0
			for _, r := range msg.Results {
				if r.Action != service.SyncUnmanaged {
					mcpCount++
				}
			}
			proj, _ := m.service.GetProject(msg.ProjectName)
			parts := []string{}
			if mcpCount > 0 {
				parts = append(parts, fmt.Sprintf("%d MCPs", mcpCount))
			}
			if n := len(proj.Skills); n > 0 {
				parts = append(parts, fmt.Sprintf("%d skills", n))
			}
			if len(parts) > 0 {
				m.syncMsg = "Synced: " + strings.Join(parts, ", ")
			} else {
				m.syncMsg = "Synced"
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if proj, ok := m.selectedProfile(); ok {
					if err := m.service.DeleteProject(proj.Name); err != nil {
						m.err = err
					} else {
						m.refreshList()
					}
				}
				return m, nil
			case "n", "N", "esc":
				m.confirming = false
				return m, nil
			}
			return m, nil
		}

		// Right pane focused.
		if m.focus == focusRight {
			switch msg.String() {
			case "up", "k":
				cur := m.sectionCursors[m.activeSection]
				m.sectionCursors[m.activeSection] = moveCursor(cur, -1, m.sectionLen(m.activeSection))
				return m, nil
			case "down", "j":
				cur := m.sectionCursors[m.activeSection]
				m.sectionCursors[m.activeSection] = moveCursor(cur, 1, m.sectionLen(m.activeSection))
				return m, nil
			case "tab":
				m.activeSection = (m.activeSection + 1) % sectionCount
				return m, nil
			case "shift+tab":
				m.activeSection = (m.activeSection - 1 + sectionCount) % sectionCount
				return m, nil
			case " ":
				cmd := m.toggleAssignment()
				return m, cmd
			case "esc", "h":
				m.focus = focusLeft
				return m, nil
			}
			return m, nil
		}

		// Left pane focused — skip shortcut keys when filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.ProfileDelete):
			if _, ok := m.selectedProfile(); ok {
				m.confirming = true
				m.err = nil
				m.syncMsg = ""
			}
			return m, nil
		case key.Matches(msg, m.keys.ProfileLaunch):
			if proj, ok := m.selectedProfile(); ok {
				return m, func() tea.Msg { return RequestLaunchMsg{ProfileName: proj.Name} }
			}
			return m, nil
		case key.Matches(msg, m.keys.ProfileConfigure):
			if _, ok := m.selectedProfile(); ok {
				m.focus = focusRight
				m.activeSection = sectionMCPs
				m.sectionCursors = [sectionCount]int{}
				m.syncMsg = ""
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	// Emit selection change if the selected profile changed.
	if proj, ok := m.selectedProfile(); ok && proj.Name != m.lastSelectedName {
		m.lastSelectedName = proj.Name
		name := proj.Name
		path := proj.Path
		cmd = tea.Batch(cmd, func() tea.Msg {
			return ProfileSelectionChangedMsg{ProfileName: name, ProjectPath: path}
		})
	}
	return m, cmd
}

// toggleAssignment toggles the assignment of the item at the current cursor
// within the active section. Returns a tea.Cmd for auto-sync on success.
func (m *ProfilesModel) toggleAssignment() tea.Cmd {
	proj, ok := m.selectedProfile()
	if !ok {
		return nil
	}
	m.err = nil

	var toggleErr error

	switch m.activeSection {
	case sectionMCPs:
		if len(m.allMCPs) == 0 {
			return nil
		}
		cursor := m.sectionCursors[sectionMCPs]
		if cursor >= len(m.allMCPs) {
			return nil
		}
		serverName := m.allMCPs[cursor]
		if m.isServerFromTag(proj, serverName) {
			m.err = fmt.Errorf("%q assigned via tag (remove from tag instead)", serverName)
			return nil
		}
		if m.isServerAssigned(proj, serverName) {
			toggleErr = m.service.UnassignServer(proj.Name, serverName)
		} else {
			toggleErr = m.service.AssignServer(proj.Name, serverName)
		}

	case sectionSkills:
		if len(m.allSkills) == 0 {
			return nil
		}
		cursor := m.sectionCursors[sectionSkills]
		if cursor >= len(m.allSkills) {
			return nil
		}
		skillName := m.allSkills[cursor]
		assigned := false
		for _, sk := range proj.Skills {
			if sk == skillName {
				assigned = true
				break
			}
		}
		if assigned {
			toggleErr = m.service.UnassignSkill(proj.Name, skillName)
		} else {
			toggleErr = m.service.AssignSkill(proj.Name, skillName)
		}

	case sectionHooks:
		if len(m.allHooks) == 0 {
			return nil
		}
		cursor := m.sectionCursors[sectionHooks]
		if cursor >= len(m.allHooks) {
			return nil
		}
		hookName := m.allHooks[cursor]
		assigned := false
		for _, h := range proj.Hooks {
			if h == hookName {
				assigned = true
				break
			}
		}
		if assigned {
			toggleErr = m.service.UnassignHook(proj.Name, hookName)
		} else {
			toggleErr = m.service.AssignHook(proj.Name, hookName)
		}

	case sectionPermissions:
		if len(m.allPermissions) == 0 {
			return nil
		}
		cursor := m.sectionCursors[sectionPermissions]
		if cursor >= len(m.allPermissions) {
			return nil
		}
		permName := m.allPermissions[cursor]
		assigned := false
		for _, p := range proj.Permissions {
			if p == permName {
				assigned = true
				break
			}
		}
		if assigned {
			toggleErr = m.service.UnassignPermission(proj.Name, permName)
		} else {
			toggleErr = m.service.AssignPermission(proj.Name, permName)
		}

	case sectionTemplate:
		if len(m.allTemplates) == 0 {
			return nil
		}
		cursor := m.sectionCursors[sectionTemplate]
		if cursor >= len(m.allTemplates) {
			return nil
		}
		templateName := m.allTemplates[cursor]
		if proj.ClaudeMD == templateName {
			toggleErr = m.service.ClearClaudeMDTemplate(proj.Name)
		} else {
			toggleErr = m.service.SetClaudeMDTemplate(proj.Name, templateName)
		}
	}

	if toggleErr != nil {
		m.err = toggleErr
		return nil
	}

	m.refreshList()
	return m.autoSync(proj.Name)
}

func (m *ProfilesModel) syncSelectedProfile() {
	proj, ok := m.selectedProfile()
	if !ok {
		return
	}
	m.err = nil
	m.syncMsg = ""
	results, err := m.service.SyncProject(proj.Name)
	if err != nil {
		m.err = err
		return
	}
	m.syncMsg = fmt.Sprintf("Synced %s: %d servers", proj.Name, len(results))
}

// View renders the profiles tab as a horizontal split: list + detail.
func (m ProfilesModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	listWidth := m.width * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	detailWidth := m.width - listWidth
	if detailWidth < 0 {
		detailWidth = 0
	}

	listView := m.list.View()
	detailView := m.renderDetail(detailWidth, m.height)

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (m ProfilesModel) renderDetail(width, height int) string {
	proj, ok := m.selectedProfile()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No profile selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render("Profile: " + proj.Name))
	b.WriteString("\n")

	if proj.Path != "" {
		fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Path:"), proj.Path)
	}

	if len(proj.Clients) > 0 || len(proj.Tags) > 0 {
		clients := make([]string, len(proj.Clients))
		for i, c := range proj.Clients {
			clients[i] = string(c)
		}
		line := ""
		if len(clients) > 0 {
			line += detailLabelStyle.Render("Clients:") + " " + strings.Join(clients, ", ")
		}
		if len(proj.Tags) > 0 {
			if line != "" {
				line += "   "
			}
			line += detailLabelStyle.Render("Tags:") + " " + strings.Join(proj.Tags, ", ")
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	// MCPs section.
	m.renderSection(&b, proj, sectionMCPs)

	// Skills section.
	m.renderSection(&b, proj, sectionSkills)

	// Hooks section.
	m.renderSection(&b, proj, sectionHooks)

	// Permissions section.
	m.renderSection(&b, proj, sectionPermissions)

	// Template section.
	m.renderSection(&b, proj, sectionTemplate)

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", proj.Name)))
	}

	if m.syncMsg != "" {
		b.WriteString("\n")
		b.WriteString(syncMsgStyle.Render(m.syncMsg))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

// renderSection writes one section block into b.
func (m ProfilesModel) renderSection(b *strings.Builder, proj model.Project, sec rightSection) {
	isActive := m.focus == focusRight && m.activeSection == sec
	cursor := m.sectionCursors[sec]

	var headerLabel string
	var items []string

	switch sec {
	case sectionMCPs:
		headerLabel = fmt.Sprintf("MCPs (%d)", len(m.allMCPs))
		items = m.allMCPs
	case sectionSkills:
		headerLabel = fmt.Sprintf("Skills (%d)", len(m.allSkills))
		items = m.allSkills
	case sectionHooks:
		headerLabel = fmt.Sprintf("Hooks (%d)", len(m.allHooks))
		items = m.allHooks
	case sectionPermissions:
		headerLabel = fmt.Sprintf("Permissions (%d)", len(m.allPermissions))
		items = m.allPermissions
	case sectionTemplate:
		headerLabel = "CLAUDE.md Template"
		items = m.allTemplates
	}

	if isActive {
		b.WriteString(sectionActiveStyle.Render(headerLabel))
	} else {
		b.WriteString(sectionHeaderStyle.Render(headerLabel))
	}
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString(sectionDimStyle.Render("  (none in registry)"))
		b.WriteString("\n")
		return
	}

	for i, name := range items {
		cursorStr := "  "
		if isActive && i == cursor {
			cursorStr = "\u25b8 " // ▸
		}

		var checkbox string
		switch sec {
		case sectionMCPs:
			assigned := m.isServerAssigned(proj, name)
			fromTag := m.isServerFromTag(proj, name)
			if assigned {
				if fromTag {
					checkbox = "[t]"
				} else {
					checkbox = "[x]"
				}
			} else {
				checkbox = "[ ]"
			}
		case sectionSkills:
			assigned := false
			for _, sk := range proj.Skills {
				if sk == name {
					assigned = true
					break
				}
			}
			if assigned {
				checkbox = "[x]"
			} else {
				checkbox = "[ ]"
			}
		case sectionHooks:
			assigned := false
			for _, h := range proj.Hooks {
				if h == name {
					assigned = true
					break
				}
			}
			if assigned {
				checkbox = "[x]"
			} else {
				checkbox = "[ ]"
			}
		case sectionPermissions:
			assigned := false
			for _, p := range proj.Permissions {
				if p == name {
					assigned = true
					break
				}
			}
			if assigned {
				checkbox = "[x]"
			} else {
				checkbox = "[ ]"
			}
		case sectionTemplate:
			if proj.ClaudeMD == name {
				checkbox = "(*)"
			} else {
				checkbox = "( )"
			}
		}

		fmt.Fprintf(b, "%s%s %s\n", cursorStr, checkbox, name)
	}
}

func (m *ProfilesModel) refreshList() {
	items := buildProfileItems(m.service)
	m.list.SetItems(items)
	m.allMCPs = buildAllServerNames(m.service)
	m.allSkills = buildAllSkillNames(m.service)
	m.allHooks = buildAllHookNames(m.service)
	m.allPermissions = buildAllPermissionNames(m.service)
	m.allTemplates = buildAllTemplateNames(m.service)
}
