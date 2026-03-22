package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hystak/hystak/internal/service"
)

// App is the root Bubble Tea model for the hystak TUI.
type App struct {
	keys      KeyMap
	svc       *service.Service
	tabs      []Tab
	activeTab TabIndex
	width     int
	height    int

	// Overlay state
	overlay      OverlayKind
	overlayMsg   string
	overlayTitle string
}

// NewApp creates the root TUI model.
func NewApp(svc *service.Service, keys KeyMap, version, commit, buildDate string) App {
	tabs := []Tab{
		newRegistryTab(keys, svc),
		newProjectsTab(keys, svc),
		newToolsTab(keys),
		newHelpTab(keys, version, commit, buildDate),
	}
	return App{
		keys:      keys,
		svc:       svc,
		tabs:      tabs,
		activeTab: TabRegistry,
	}
}

func (a App) Init() tea.Cmd {
	// Kick off each tab's Init
	cmds := make([]tea.Cmd, len(a.tabs))
	for i, tab := range a.tabs {
		cmds[i] = tab.Init()
	}
	return tea.Batch(cmds...)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate to all tabs
		var cmds []tea.Cmd
		for i, tab := range a.tabs {
			updated, cmd := tab.Update(msg)
			a.tabs[i] = updated.(Tab)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return a, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Handle overlay keys first
		if a.overlay != OverlayNone {
			return a.handleOverlayKey(msg)
		}
		// Global keys
		return a.handleGlobalKey(msg)
	}

	// Delegate other messages to active tab
	updated, cmd := a.tabs[a.activeTab].Update(msg)
	a.tabs[a.activeTab] = updated.(Tab)
	return a, cmd
}

func (a App) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		return a, tea.Quit
	case msg.String() == "q" && a.activeTab != TabRegistry:
		return a, tea.Quit
	case key.Matches(msg, a.keys.NextTab):
		a.activeTab = (a.activeTab + 1) % TabIndex(tabCount)
		return a, nil
	case key.Matches(msg, a.keys.PrevTab):
		a.activeTab = (a.activeTab - 1 + TabIndex(tabCount)) % TabIndex(tabCount)
		return a, nil
	case msg.String() == "1":
		a.activeTab = TabRegistry
		return a, nil
	case msg.String() == "2":
		a.activeTab = TabProjects
		return a, nil
	case msg.String() == "3":
		a.activeTab = TabTools
		return a, nil
	case msg.String() == "4":
		a.activeTab = TabHelp
		return a, nil
	}

	// Delegate to active tab
	updated, cmd := a.tabs[a.activeTab].Update(msg)
	a.tabs[a.activeTab] = updated.(Tab)
	return a, cmd
}

func (a App) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.overlay {
	case OverlayConfirm:
		switch msg.String() {
		case "y", "Y", "enter":
			a.overlay = OverlayNone
			return a, nil
		case "n", "N", "esc":
			a.overlay = OverlayNone
			return a, nil
		}
	default:
		if msg.String() == "esc" {
			a.overlay = OverlayNone
			return a, nil
		}
	}
	return a, nil
}

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title bar
	title := styleTitle.Render(" hystak ")
	b.WriteString(title)
	b.WriteString("\n")

	// Tab bar
	b.WriteString(a.renderTabBar())
	b.WriteString("\n")

	// Active tab content
	tabContent := a.tabs[a.activeTab].View()
	b.WriteString(tabContent)

	// Footer / help bar
	b.WriteString("\n")
	b.WriteString(a.renderFooter())

	// If overlay active, render it on top
	if a.overlay != OverlayNone {
		base := b.String()
		overlay := a.renderOverlay()
		return a.compositeOverlay(base, overlay)
	}

	return b.String()
}

func (a App) renderTabBar() string {
	var parts []string
	for i, tab := range a.tabs {
		name := tab.Title()
		if TabIndex(i) == a.activeTab {
			parts = append(parts, styleTabActive.Render(" "+name+" "))
		} else {
			parts = append(parts, styleTabInactive.Render(" "+name+" "))
		}
	}
	return styleTabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
}

func (a App) renderFooter() string {
	entries := a.tabs[a.activeTab].HelpKeys()

	var parts []string
	for _, e := range entries {
		parts = append(parts,
			styleHelpKey.Render(e.Key)+styleHelpDesc.Render(":"+e.Desc),
		)
	}
	parts = append(parts,
		styleHelpKey.Render("Tab")+styleHelpDesc.Render(":Switch tab"),
		styleHelpKey.Render("q")+styleHelpDesc.Render(":Quit"),
	)
	return styleFooter.Render(strings.Join(parts, "  "))
}

func (a App) renderOverlay() string {
	switch a.overlay {
	case OverlayConfirm:
		return confirmOverlay(a.overlayTitle, a.overlayMsg, a.width, a.height)
	default:
		return ""
	}
}

// compositeOverlay layers the overlay on top of the base content.
func (a App) compositeOverlay(base, overlay string) string {
	_ = base
	return overlay
}
