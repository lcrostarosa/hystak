package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// paneFocus identifies which pane has keyboard focus in the projects tab.
type paneFocus int

const (
	focusLeft paneFocus = iota
	focusRight
)

// projectItem implements list.DefaultItem for the project list.
type projectItem struct {
	project     model.Project
	serverCount int
}

func (i projectItem) Title() string {
	if i.serverCount > 0 {
		return fmt.Sprintf("%s [%d]", i.project.Name, i.serverCount)
	}
	return i.project.Name
}

func (i projectItem) Description() string {
	if i.project.Path != "" {
		return i.project.Path
	}
	return "no path"
}

func (i projectItem) FilterValue() string { return i.project.Name }

// ProjectDeletedMsg is sent when a project has been deleted.
type ProjectDeletedMsg struct{ Name string }

// ProjectsModel is the sub-model for the Projects tab.
type ProjectsModel struct {
	list         list.Model
	service      *service.Service
	width        int
	height       int
	confirming   bool
	err          error
	syncMsg      string

	focus        paneFocus
	serverCursor int
	allServers   []string // all registry server names, sorted
}

// NewProjectsModel creates a new ProjectsModel.
func NewProjectsModel(svc *service.Service) ProjectsModel {
	items := buildProjectItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return ProjectsModel{
		list:       l,
		service:    svc,
		allServers: buildAllServerNames(svc),
	}
}

func buildProjectItems(svc *service.Service) []list.Item {
	if svc == nil || svc.Projects == nil {
		return nil
	}

	projects := svc.Projects.List()
	items := make([]list.Item, len(projects))
	for i, proj := range projects {
		items[i] = projectItem{
			project:     proj,
			serverCount: countAssignedServers(svc, proj),
		}
	}
	return items
}

func countAssignedServers(svc *service.Service, proj model.Project) int {
	if svc.Registry == nil {
		return len(proj.MCPs)
	}
	seen := make(map[string]bool)
	for _, tag := range proj.Tags {
		if names, err := svc.Registry.ExpandTag(tag); err == nil {
			for _, name := range names {
				seen[name] = true
			}
		}
	}
	for _, mcp := range proj.MCPs {
		seen[mcp.Name] = true
	}
	return len(seen)
}

func buildAllServerNames(svc *service.Service) []string {
	if svc == nil || svc.Registry == nil {
		return nil
	}
	servers := svc.Registry.List()
	names := make([]string, len(servers))
	for i, srv := range servers {
		names[i] = srv.Name
	}
	sort.Strings(names)
	return names
}

func (m ProjectsModel) selectedProject() (model.Project, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.Project{}, false
	}
	pi, ok := item.(projectItem)
	if !ok {
		return model.Project{}, false
	}
	return pi.project, true
}

// isServerAssigned checks if a server is assigned to the given project
// (either directly via MCPs or via tag expansion).
func (m ProjectsModel) isServerAssigned(proj model.Project, serverName string) bool {
	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			return true
		}
	}
	if m.service != nil && m.service.Registry != nil {
		for _, tag := range proj.Tags {
			if names, err := m.service.Registry.ExpandTag(tag); err == nil {
				for _, name := range names {
					if name == serverName {
						return true
					}
				}
			}
		}
	}
	return false
}

// isServerFromTag checks if a server's assignment comes only from tag expansion
// (not from a direct MCP entry).
func (m ProjectsModel) isServerFromTag(proj model.Project, serverName string) bool {
	for _, mcp := range proj.MCPs {
		if mcp.Name == serverName {
			return false
		}
	}
	if m.service != nil && m.service.Registry != nil {
		for _, tag := range proj.Tags {
			if names, err := m.service.Registry.ExpandTag(tag); err == nil {
				for _, name := range names {
					if name == serverName {
						return true
					}
				}
			}
		}
	}
	return false
}

// IsConsuming returns true when the model handles its own input
// (e.g., filtering, confirming, or navigating the right pane).
func (m ProjectsModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming || m.focus == focusRight
}

// SetSize updates the dimensions available to the projects tab.
func (m *ProjectsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m ProjectsModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	if m.focus == focusRight {
		return "space: toggle | s: sync | D: diff | esc: back"
	}
	return "enter: servers | d: delete | s: sync | D: diff | /: filter | tab: switch tabs | q: quit"
}

// Update handles messages for the projects tab.
func (m ProjectsModel) Update(msg tea.Msg) (ProjectsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if proj, ok := m.selectedProject(); ok {
					if err := m.service.Projects.Remove(proj.Name); err != nil {
						m.err = err
					} else {
						_ = m.service.SaveProjects()
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
				m.serverCursor = moveCursor(m.serverCursor, -1, len(m.allServers))
				return m, nil
			case "down", "j":
				m.serverCursor = moveCursor(m.serverCursor, 1, len(m.allServers))
				return m, nil
			case " ":
				m.toggleServerAssignment()
				return m, nil
			case "s":
				m.syncSelectedProject()
				return m, nil
			case "D":
				if proj, ok := m.selectedProject(); ok {
					return m, func() tea.Msg { return RequestDiffMsg{ProjectName: proj.Name} }
				}
				return m, nil
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

		switch msg.String() {
		case "d":
			if _, ok := m.selectedProject(); ok {
				m.confirming = true
				m.err = nil
				m.syncMsg = ""
			}
			return m, nil
		case "enter", "l":
			if _, ok := m.selectedProject(); ok && len(m.allServers) > 0 {
				m.focus = focusRight
				m.serverCursor = 0
				m.syncMsg = ""
			}
			return m, nil
		case "s":
			m.syncSelectedProject()
			return m, nil
		case "D":
			if proj, ok := m.selectedProject(); ok {
				return m, func() tea.Msg { return RequestDiffMsg{ProjectName: proj.Name} }
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ProjectsModel) toggleServerAssignment() {
	proj, ok := m.selectedProject()
	if !ok || m.serverCursor >= len(m.allServers) {
		return
	}

	serverName := m.allServers[m.serverCursor]

	if m.isServerFromTag(proj, serverName) {
		m.err = fmt.Errorf("%q assigned via tag (remove from tag instead)", serverName)
		return
	}

	m.err = nil
	if m.isServerAssigned(proj, serverName) {
		if err := m.service.Projects.Unassign(proj.Name, serverName); err != nil {
			m.err = err
			return
		}
	} else {
		if err := m.service.Projects.Assign(proj.Name, serverName); err != nil {
			m.err = err
			return
		}
	}
	_ = m.service.SaveProjects()
	m.refreshList()
}

func (m *ProjectsModel) syncSelectedProject() {
	proj, ok := m.selectedProject()
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

// View renders the projects tab as a horizontal split: list + detail.
func (m ProjectsModel) View() string {
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

func (m ProjectsModel) renderDetail(width, height int) string {
	proj, ok := m.selectedProject()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No project selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(proj.Name))
	b.WriteString("\n")

	if proj.Path != "" {
		fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Path:"), proj.Path)
	}

	if len(proj.Tags) > 0 {
		fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Tags:"), strings.Join(proj.Tags, ", "))
	}

	if len(proj.Clients) > 0 {
		clients := make([]string, len(proj.Clients))
		for i, c := range proj.Clients {
			clients[i] = string(c)
		}
		fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Clients:"), strings.Join(clients, ", "))
	}

	b.WriteString("\n")
	b.WriteString(detailLabelStyle.Render("Servers:"))
	b.WriteString("\n")

	if len(m.allServers) == 0 {
		b.WriteString("  (no servers in registry)\n")
	} else {
		for i, name := range m.allServers {
			assigned := m.isServerAssigned(proj, name)
			fromTag := m.isServerFromTag(proj, name)

			cursor := "  "
			if m.focus == focusRight && i == m.serverCursor {
				cursor = "\u25b8 " // ▸
			}

			checkbox := "[ ]"
			if assigned {
				if fromTag {
					checkbox = "[t]"
				} else {
					checkbox = "[x]"
				}
			}

			fmt.Fprintf(&b, "%s%s %s\n", cursor, checkbox, name)
		}
	}

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

func (m *ProjectsModel) refreshList() {
	items := buildProjectItems(m.service)
	m.list.SetItems(items)
	m.allServers = buildAllServerNames(m.service)
}
