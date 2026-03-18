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

// serverItem implements list.DefaultItem for the server list.
type serverItem struct {
	server       model.ServerDef
	projectCount int
}

func (i serverItem) Title() string {
	if i.projectCount > 0 {
		return fmt.Sprintf("%s ⌂%d", i.server.Name, i.projectCount)
	}
	return i.server.Name
}

func (i serverItem) Description() string {
	if i.server.Description != "" {
		return i.server.Description
	}
	return string(i.server.Transport)
}

func (i serverItem) FilterValue() string { return i.server.Name }

// ServerDeletedMsg is sent when a server has been deleted.
type ServerDeletedMsg struct{ Name string }

// ServersModel is the sub-model for the Servers tab.
type ServersModel struct {
	list       list.Model
	service    *service.Service
	width      int
	height     int
	confirming bool
	err        error
}

// NewServersModel creates a new ServersModel.
func NewServersModel(svc *service.Service) ServersModel {
	items := buildServerItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return ServersModel{
		list:    l,
		service: svc,
	}
}

func buildServerItems(svc *service.Service) []list.Item {
	if svc == nil || svc.Registry == nil {
		return nil
	}

	projectCounts := countProjectRefs(svc)
	servers := svc.Registry.List()
	items := make([]list.Item, len(servers))
	for i, srv := range servers {
		items[i] = serverItem{
			server:       srv,
			projectCount: projectCounts[srv.Name],
		}
	}
	return items
}

func countProjectRefs(svc *service.Service) map[string]int {
	counts := make(map[string]int)
	if svc.Projects == nil || svc.Registry == nil {
		return counts
	}

	for _, proj := range svc.Projects.List() {
		seen := make(map[string]bool)
		for _, tag := range proj.Tags {
			if names, err := svc.Registry.ExpandTag(tag); err == nil {
				for _, name := range names {
					if !seen[name] {
						seen[name] = true
						counts[name]++
					}
				}
			}
		}
		for _, mcp := range proj.MCPs {
			if !seen[mcp.Name] {
				seen[mcp.Name] = true
				counts[mcp.Name]++
			}
		}
	}
	return counts
}

func (m ServersModel) selectedServer() (model.ServerDef, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.ServerDef{}, false
	}
	si, ok := item.(serverItem)
	if !ok {
		return model.ServerDef{}, false
	}
	return si.server, true
}

// IsConsuming returns true when the model handles its own input
// (e.g., filtering or confirming a delete).
func (m ServersModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming
}

// SetSize updates the dimensions available to the servers tab.
func (m *ServersModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m ServersModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	return "a: add | e: edit | d: delete | i: import | /: filter | tab: switch tabs | q: quit"
}

// Update handles messages for the servers tab.
func (m ServersModel) Update(msg tea.Msg) (ServersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if srv, ok := m.selectedServer(); ok {
					if err := m.service.Registry.Delete(srv.Name); err != nil {
						m.err = err
					} else {
						_ = m.service.SaveRegistry()
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

		// Don't handle shortcut keys when filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "a":
			return m, func() tea.Msg { return RequestFormMsg{} }
		case "e":
			if srv, ok := m.selectedServer(); ok {
				return m, func() tea.Msg { return RequestFormMsg{EditServer: &srv} }
			}
			return m, nil
		case "d":
			if _, ok := m.selectedServer(); ok {
				m.confirming = true
				m.err = nil
			}
			return m, nil
		case "i":
			return m, func() tea.Msg { return RequestImportMsg{} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the servers tab as a horizontal split: list + detail.
func (m ServersModel) View() string {
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

func (m ServersModel) renderDetail(width, height int) string {
	srv, ok := m.selectedServer()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No server selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(srv.Name))
	b.WriteString("\n")

	if srv.Description != "" {
		b.WriteString(srv.Description)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	writeServerFields(&b, srv, detailLabelStyle)

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", srv.Name)))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

func (m *ServersModel) refreshList() {
	items := buildServerItems(m.service)
	m.list.SetItems(items)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
