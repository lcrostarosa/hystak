package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rbbydotdev/hystak/internal/service"
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
}

// NewApp creates a new TUI application model.
func NewApp(svc *service.Service) AppModel {
	return AppModel{
		service:   svc,
		activeTab: ServersTab,
		mode:      ModeBrowse,
		keys:      newKeyMap(),
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
		return m, nil

	case tea.KeyMsg:
		// In overlay modes, only handle escape to return to browse.
		if m.mode != ModeBrowse {
			if msg.String() == "esc" {
				m.mode = ModeBrowse
			}
			return m, nil
		}

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

	return m, nil
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
		content = m.renderServersPlaceholder()
	case ProjectsTab:
		content = m.renderProjectsPlaceholder()
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
	help := "tab: switch tabs | q: quit"
	return statusBarStyle.Render(help)
}

func (m AppModel) renderServersPlaceholder() string {
	return "Servers tab — content coming in Step 9"
}

func (m AppModel) renderProjectsPlaceholder() string {
	return "Projects tab — content coming in Step 10"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
