package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/service"
)

// toolAction identifies an action in the Tools tab.
type toolAction int

const (
	toolSync toolAction = iota
	toolDiff
	toolDiscover
	toolLaunch
	toolActionCount
)

var toolActionLabels = []string{"Sync", "Diff", "Discover", "Launch"}
var toolActionDescs = []string{
	"Deploy config to client files",
	"Show diff between registry and deployed",
	"Auto-discover skills in project directory",
	"Launch Claude Code with this profile",
}

// ProfileSelectionChangedMsg is emitted when the profiles tab selection changes.
type ProfileSelectionChangedMsg struct {
	ProfileName string
	ProjectPath string
}

// ToolsModel is the sub-model for the Tools tab.
type ToolsModel struct {
	service     *service.Service
	keys        KeyMap
	width       int
	height      int
	cursor      int
	profileName string
	profilePath string
	err         error
	statusMsg   string
}

// NewToolsModel creates a new ToolsModel.
func NewToolsModel(svc *service.Service, keys KeyMap) ToolsModel {
	return ToolsModel{
		service: svc,
		keys:    keys,
	}
}

// SetProfile updates the target profile for tools actions.
func (m *ToolsModel) SetProfile(name, path string) {
	m.profileName = name
	m.profilePath = path
	m.err = nil
	m.statusMsg = ""
}

// IsConsuming returns false; the tools tab never consumes global input.
func (m ToolsModel) IsConsuming() bool { return false }

// SetSize updates the dimensions.
func (m *ToolsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// StatusHelp returns help text for the status bar.
func (m ToolsModel) StatusHelp() string {
	return fmt.Sprintf("%s: run | up/down: select | %s",
		m.keys.ToolsExecute.Help().Key, m.keys.tabNavHelp())
}

// Update handles messages for the Tools tab.
func (m ToolsModel) Update(msg tea.Msg) (ToolsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "up" || msg.String() == "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case msg.String() == "down" || msg.String() == "j":
			if m.cursor < int(toolActionCount)-1 {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.ToolsExecute):
			return m.executeAction()
		}
	}
	return m, nil
}

func (m ToolsModel) executeAction() (ToolsModel, tea.Cmd) {
	if m.profileName == "" {
		m.err = fmt.Errorf("no profile selected")
		return m, nil
	}
	m.err = nil
	m.statusMsg = ""

	switch toolAction(m.cursor) {
	case toolSync:
		results, err := m.service.SyncProject(m.profileName)
		if err != nil {
			m.err = err
		} else {
			m.statusMsg = fmt.Sprintf("Synced %s: %d servers", m.profileName, len(results))
		}
		return m, nil

	case toolDiff:
		name := m.profileName
		return m, func() tea.Msg { return RequestDiffMsg{ProjectName: name} }

	case toolDiscover:
		name := m.profileName
		path := m.profilePath
		return m, func() tea.Msg {
			return RequestDiscoveryMsg{ProjectName: name, ProjectPath: path}
		}

	case toolLaunch:
		name := m.profileName
		return m, func() tea.Msg { return RequestLaunchMsg{ProfileName: name} }
	}

	return m, nil
}

// View renders the Tools tab.
func (m ToolsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	if m.profileName == "" {
		b.WriteString(sectionDimStyle.Render("No profile selected -- select one on the Profiles tab"))
		return detailPaneStyle.Width(m.width).Height(m.height).Render(b.String())
	}

	b.WriteString(detailTitleStyle.Render("Tools: " + m.profileName))
	b.WriteString("\n")
	if m.profilePath != "" {
		b.WriteString(sectionDimStyle.Render(m.profilePath))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	for i := 0; i < int(toolActionCount); i++ {
		cursor := "  "
		if i == m.cursor {
			cursor = "\u25b8 " // ▸
		}

		label := toolActionLabels[i]
		desc := toolActionDescs[i]
		if i == m.cursor {
			b.WriteString(cursor + sectionActiveStyle.Render(label))
		} else {
			b.WriteString(cursor + sectionHeaderStyle.Render(label))
		}
		b.WriteString("  " + sectionDimStyle.Render(desc))
		b.WriteString("\n")
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(syncMsgStyle.Render(m.statusMsg))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return lipgloss.NewStyle().Width(m.width).Height(m.height).Padding(1, 2).Render(b.String())
}
