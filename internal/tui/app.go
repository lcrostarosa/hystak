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

// overlay is the interface for full-screen overlay models.
type overlay interface {
	SetSize(w, h int)
	View() string
}

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
	form     FormModel
	importer ImportModel
	diff     DiffModel
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

// activeOverlay returns the current overlay model, or nil in browse/confirm modes.
func (m *AppModel) activeOverlay() overlay {
	switch m.mode {
	case ModeForm:
		return &m.form
	case ModeImport:
		return &m.importer
	case ModeDiff:
		return &m.diff
	}
	return nil
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
		if ov := m.activeOverlay(); ov != nil {
			ov.SetSize(m.width, m.height)
		}
		return m, nil

	case RequestFormMsg:
		if msg.EditServer != nil {
			m.form = NewEditFormModel(*msg.EditServer)
		} else {
			m.form = NewFormModel()
		}
		m.form.SetSize(m.width, m.height)
		m.mode = ModeForm
		return m, nil

	case FormSubmittedMsg:
		m.mode = ModeBrowse
		m.err = nil
		var err error
		if msg.IsEdit {
			err = m.service.Registry.Update(m.form.editName, msg.Server)
		} else {
			err = m.service.Registry.Add(msg.Server)
		}
		if err != nil {
			m.err = err
		} else {
			_ = m.service.SaveRegistry()
		}
		m.servers.refreshList()
		return m, nil

	case FormCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestImportMsg:
		m.importer = NewImportModel(m.service)
		m.importer.SetSize(m.width, m.height)
		m.mode = ModeImport
		return m, nil

	case ImportCompletedMsg:
		m.mode = ModeBrowse
		m.servers.refreshList()
		return m, nil

	case ImportCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestDiffMsg:
		m.diff = NewDiffModel(m.service, msg.ProjectName)
		m.diff.SetSize(m.width, m.height)
		m.mode = ModeDiff
		return m, nil

	case DiffClosedMsg:
		m.mode = ModeBrowse
		m.projects.refreshList()
		return m, nil

	case tea.KeyMsg:
		// In form mode, route all input to the form.
		if m.mode == ModeForm {
			var cmd tea.Cmd
			m.form, cmd = m.form.Update(msg)
			return m, cmd
		}

		// In import mode, route all input to the importer.
		if m.mode == ModeImport {
			var cmd tea.Cmd
			m.importer, cmd = m.importer.Update(msg)
			return m, cmd
		}

		// In diff mode, route all input to the diff model.
		if m.mode == ModeDiff {
			var cmd tea.Cmd
			m.diff, cmd = m.diff.Update(msg)
			return m, cmd
		}

		// In other overlay modes, handle escape to return to browse.
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
	return max(0, m.height-overhead)
}

// View implements tea.Model.
func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Overlay modes take over the full screen.
	if ov := m.activeOverlay(); ov != nil {
		return ov.View()
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
	contentHeight := max(0, m.height-tabBarHeight-statusBarHeight)

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
