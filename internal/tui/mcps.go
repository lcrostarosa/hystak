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

// mcpItem implements list.DefaultItem for the MCP list.
type mcpItem struct {
	server       model.ServerDef
	profileCount int
}

func (i mcpItem) Title() string {
	if i.profileCount > 0 {
		return fmt.Sprintf("%s ⌂%d", i.server.Name, i.profileCount)
	}
	return i.server.Name
}

func (i mcpItem) Description() string {
	if i.server.Description != "" {
		return i.server.Description
	}
	return string(i.server.Transport)
}

func (i mcpItem) FilterValue() string { return i.server.Name }

// MCPDeletedMsg is sent when an MCP has been deleted.
type MCPDeletedMsg struct{ Name string }

// MCPsModel is the sub-model for the MCPs tab.
type MCPsModel struct {
	list       list.Model
	service    *service.Service
	keys       KeyMap
	width      int
	height     int
	confirming bool
	err        error
}

// NewMCPsModel creates a new MCPsModel.
func NewMCPsModel(svc *service.Service, keys KeyMap) MCPsModel {
	items := buildMCPItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return MCPsModel{
		list:    l,
		service: svc,
		keys:    keys,
	}
}

func buildMCPItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}

	profileCounts := svc.CountServerProfileRefs()
	servers := svc.ListServers()
	items := make([]list.Item, len(servers))
	for i, srv := range servers {
		items[i] = mcpItem{
			server:       srv,
			profileCount: profileCounts[srv.Name],
		}
	}
	return items
}

func (m MCPsModel) selectedMCP() (model.ServerDef, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.ServerDef{}, false
	}
	si, ok := item.(mcpItem)
	if !ok {
		return model.ServerDef{}, false
	}
	return si.server, true
}

// IsConsuming returns true when the model handles its own input
// (e.g., filtering or confirming a delete).
func (m MCPsModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming
}

// SetSize updates the dimensions available to the MCPs tab.
func (m *MCPsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m MCPsModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	return fmt.Sprintf("%s: add | %s: edit | %s: delete | %s: import | /: filter | %s | q: quit",
		m.keys.MCPAdd.Help().Key, m.keys.MCPEdit.Help().Key,
		m.keys.MCPDelete.Help().Key, m.keys.MCPImport.Help().Key,
		m.keys.tabNavHelp())
}

// Update handles messages for the MCPs tab.
func (m MCPsModel) Update(msg tea.Msg) (MCPsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if srv, ok := m.selectedMCP(); ok {
					if err := m.service.DeleteServer(srv.Name); err != nil {
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

		// Don't handle shortcut keys when filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.MCPAdd):
			return m, func() tea.Msg { return RequestFormMsg{} }
		case key.Matches(msg, m.keys.MCPEdit):
			if srv, ok := m.selectedMCP(); ok {
				return m, func() tea.Msg { return RequestFormMsg{EditServer: &srv} }
			}
			return m, nil
		case key.Matches(msg, m.keys.MCPDelete):
			if _, ok := m.selectedMCP(); ok {
				m.confirming = true
				m.err = nil
			}
			return m, nil
		case key.Matches(msg, m.keys.MCPImport):
			return m, func() tea.Msg { return RequestImportMsg{} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the MCPs tab as a horizontal split: list + detail.
func (m MCPsModel) View() string {
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

func (m MCPsModel) renderDetail(width, height int) string {
	srv, ok := m.selectedMCP()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No MCP selected")
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

func (m *MCPsModel) refreshList() {
	items := buildMCPItems(m.service)
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
