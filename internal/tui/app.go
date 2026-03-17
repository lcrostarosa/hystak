package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/service"
)

// Tab identifies which tab is active.
type Tab int

const (
	ServersTab Tab = iota
	ProjectsTab
	tabCount
)

// Mode represents the current UI mode.
type Mode int

const (
	ModeBrowse Mode = iota
	ModeForm
	ModeConfirm
	ModeDiff
	ModeImport
)

var tabLabels = []string{"Servers", "Projects"}

// AppModel is the root Bubble Tea model for the hystak TUI.
type AppModel struct {
	service   *service.Service
	activeTab Tab
	mode      Mode
	keys      KeyMap
	width     int
	height    int
	err       error

	servers  ServersModel
	projects ProjectsModel
}

// NewApp creates a new TUI application model.
func NewApp(svc *service.Service) AppModel {
	return AppModel{
		service:   svc,
		activeTab: ServersTab,
		mode:      ModeBrowse,
		keys:      newKeyMap(),
		servers:  NewServersModel(svc),
		projects: NewProjectsModel(svc),
	}
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateContentSize()
		return m, nil

	case tea.KeyMsg:
		// In overlay modes, only handle escape to return to browse.
		if m.mode != ModeBrowse {
			if msg.String() == "esc" {
				m.mode = ModeBrowse
			}
			return m, nil
		}

		// If the active tab is consuming input (filtering, confirming),
		// skip global key handling and route directly to the tab.
		if !m.activeTabConsuming() {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit

			case key.Matches(msg, m.keys.TabNext):
				m.activeTab = (m.activeTab + 1) % tabCount
				return m, nil

			case key.Matches(msg, m.keys.TabPrev):
				m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
				return m, nil
			}
		}
	}

	// Route remaining messages to the active tab.
	return m.updateActiveTab(msg)
}

// activeTabConsuming returns true if the active tab is handling its own input.
func (m AppModel) activeTabConsuming() bool {
	switch m.activeTab {
	case ServersTab:
		return m.servers.IsConsuming()
	case ProjectsTab:
		return m.projects.IsConsuming()
	}
	return false
}

// updateActiveTab forwards a message to the currently active tab.
func (m AppModel) updateActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case ServersTab:
		var cmd tea.Cmd
		m.servers, cmd = m.servers.Update(msg)
		return m, cmd
	case ProjectsTab:
		var cmd tea.Cmd
		m.projects, cmd = m.projects.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateContentSize recalculates and propagates dimensions to sub-models.
func (m *AppModel) updateContentSize() {
	contentHeight := m.contentHeight()
	m.servers.SetSize(m.width, contentHeight)
	m.projects.SetSize(m.width, contentHeight)
}

// contentHeight returns the height available for tab content.
func (m AppModel) contentHeight() int {
	// Tab bar is 1 line + border = 2 lines, status bar is 1 line.
	// contentStyle has padding(1,2) = 2 vertical lines.
	overhead := 2 + 1 + 2
	h := m.height - overhead
	if h < 0 {
		h = 0
	}
	return h
}

// View implements tea.Model.
func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	tabBar := m.renderTabBar()

	var content string
	switch m.activeTab {
	case ServersTab:
		content = m.servers.View()
	case ProjectsTab:
		content = m.projects.View()
	}

	statusBar := m.renderStatusBar()

	// Calculate content height: total - tab bar - status bar - padding
	tabBarHeight := lipgloss.Height(tabBar)
	statusBarHeight := lipgloss.Height(statusBar)
	contentHeight := m.height - tabBarHeight - statusBarHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	styledContent := contentStyle.
		Width(m.width).
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, styledContent, statusBar)
}

func (m AppModel) renderTabBar() string {
	var tabs []string
	for i, label := range tabLabels {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Fill remaining width with gap
	gap := tabGapStyle.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(row))))
	return lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
}

func (m AppModel) renderStatusBar() string {
	var help string
	switch m.activeTab {
	case ServersTab:
		help = m.servers.StatusHelp()
	case ProjectsTab:
		help = m.projects.StatusHelp()
	default:
		help = "tab: switch tabs | q: quit"
	}
	return statusBarStyle.Render(help)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
