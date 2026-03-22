package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
	"github.com/hystak/hystak/internal/service"
)

// SubNav identifies the active resource type within the Registry tab.
type SubNav int

const (
	SubNavMCPs SubNav = iota
	SubNavSkills
	SubNavHooks
	SubNavPermissions
	SubNavTemplates
	SubNavPrompts
	subNavCount
)

var subNavNames = [subNavCount]string{
	"MCPs", "Skills", "Hooks", "Permissions", "Templates", "Prompts",
}

// registryTab is the Registry tab — shows all managed resources with sub-navigation.
type registryTab struct {
	keys    KeyMap
	svc     *service.Service
	sub     SubNav
	servers []model.ServerDef
	cursor  int
	width   int
	height  int
}

func newRegistryTab(keys KeyMap, svc *service.Service) *registryTab {
	return &registryTab{
		keys: keys,
		svc:  svc,
	}
}

func (t *registryTab) Title() string { return "Registry" }

func (t *registryTab) HelpKeys() []HelpEntry {
	return []HelpEntry{
		{"A", "Add"},
		{"E", "Edit"},
		{"D", "Delete"},
		{"/", "Filter"},
		{"I", "Import"},
	}
}

// registryLoadedMsg is sent when registry data has been loaded asynchronously.
type registryLoadedMsg struct {
	servers []model.ServerDef
}

func (t *registryTab) Init() tea.Cmd {
	return t.loadData
}

func (t *registryTab) loadData() tea.Msg {
	return registryLoadedMsg{
		servers: t.svc.ListServers(),
	}
}

func (t *registryTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case registryLoadedMsg:
		t.servers = msg.servers
		t.cursor = 0
		return t, nil

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t *registryTab) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, t.keys.ListUp):
		if t.cursor > 0 {
			t.cursor--
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.cursor < len(t.servers)-1 {
			t.cursor++
		}
	case msg.String() == "left" || msg.String() == "shift+tab":
		if t.sub > 0 {
			t.sub--
			return t, t.loadData
		}
	case msg.String() == "right" || msg.String() == "tab":
		if t.sub < subNavCount-1 {
			t.sub++
			return t, t.loadData
		}
	}
	return t, nil
}

func (t *registryTab) View() string {
	var b strings.Builder

	// Sub-nav bar
	for i := SubNav(0); i < subNavCount; i++ {
		name := subNavNames[i]
		if i == t.sub {
			b.WriteString(styleTabActive.Render("[" + name + "]"))
		} else {
			b.WriteString(styleTabInactive.Render(" " + name + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	// Currently only MCPs sub-nav is fully implemented;
	// others show a placeholder.
	if t.sub != SubNavMCPs {
		b.WriteString(styleListHeader.Render(
			fmt.Sprintf("  %s — coming soon", subNavNames[t.sub]),
		))
		b.WriteString("\n")
		return b.String()
	}

	// Column header
	b.WriteString(styleListHeader.Render(
		fmt.Sprintf("  %-20s  %-10s  %s", "NAME", "TRANSPORT", "COMMAND/URL"),
	))
	b.WriteString("\n")

	if len(t.servers) == 0 {
		b.WriteString("  (no servers registered)\n")
		return b.String()
	}

	for i, s := range t.servers {
		endpoint := s.Command
		if s.Transport == model.TransportSSE || s.Transport == model.TransportHTTP {
			endpoint = s.URL
		}
		line := fmt.Sprintf("  %-20s  %-10s  %s", truncate(s.Name, 20), s.Transport, truncate(endpoint, 40))
		if i == t.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(styleListNormal.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// truncate shortens a string to max length, appending "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
