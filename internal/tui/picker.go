package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// PickerResult holds the outcome of the profile picker.
type PickerResult struct {
	Project   *model.Project // selected project, nil if bare launch
	Manage    bool           // true if user chose "Manage profiles..."
	Configure bool           // true if user chose to configure a project
}

// pickerItem implements list.DefaultItem.
type pickerItem struct {
	name string
	desc string
	kind pickerItemKind
	proj *model.Project
}

type pickerItemKind int

const (
	pickerProject pickerItemKind = iota
	pickerBare
	pickerManage
	pickerConfigure
)

func (i pickerItem) Title() string       { return i.name }
func (i pickerItem) Description() string { return i.desc }
func (i pickerItem) FilterValue() string { return i.name }

var pickerTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205")).
	MarginBottom(1)

// PickerModel is a lightweight Bubble Tea model for selecting a project profile.
type PickerModel struct {
	list     list.Model
	result   *PickerResult
	width    int
	height   int
	showLogo bool
	version  string
}

// NewPickerModel creates a picker from the service's project list.
func NewPickerModel(svc *service.Service, version string) PickerModel {
	projects := svc.ListProjects()

	items := make([]list.Item, 0, len(projects)+3)
	for i := range projects {
		p := projects[i]
		mcpCount := svc.CountAssignedServers(p)
		skillCount := len(p.Skills)
		hookCount := len(p.Hooks)

		// Build title with profile indicator.
		title := p.Name
		if !p.Launched {
			title += " (new)"
		} else if p.ActiveProfile != "" {
			title += fmt.Sprintf(" [%s]", p.ActiveProfile)
		}

		var counts []string
		if mcpCount > 0 {
			counts = append(counts, fmt.Sprintf("%d MCPs", mcpCount))
		}
		if skillCount > 0 {
			counts = append(counts, fmt.Sprintf("%d skills", skillCount))
		}
		if hookCount > 0 {
			counts = append(counts, fmt.Sprintf("%d hooks", hookCount))
		}

		desc := p.Path
		if len(counts) > 0 {
			desc += " ("
			for j, c := range counts {
				if j > 0 {
					desc += ", "
				}
				desc += c
			}
			desc += ")"
		}

		items = append(items, pickerItem{
			name: title,
			desc: desc,
			kind: pickerProject,
			proj: &projects[i],
		})
	}

	// Sentinel items
	items = append(items, pickerItem{
		name: "Launch without profile",
		desc: "Run claude in the current directory",
		kind: pickerBare,
	})
	items = append(items, pickerItem{
		name: "Configure...",
		desc: "Open the launch wizard for the selected project",
		kind: pickerConfigure,
	})
	items = append(items, pickerItem{
		name: "Manage...",
		desc: "Open the full management TUI",
		kind: pickerManage,
	})

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "hystak"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return PickerModel{list: l, showLogo: true, version: version}
}

// Result returns the picker result after the program exits, or nil if cancelled.
func (m PickerModel) Result() *PickerResult {
	return m.result
}

func (m PickerModel) Init() tea.Cmd {
	return nil
}

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listHeight := msg.Height
		if m.showLogo {
			listHeight -= logoHeight()
		}
		// Account for status bar
		listHeight -= 1
		m.list.SetSize(msg.Width, max(1, listHeight))
		return m, tea.ClearScreen

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "enter":
			item, ok := m.list.SelectedItem().(pickerItem)
			if !ok {
				return m, nil
			}
			switch item.kind {
			case pickerProject:
				m.result = &PickerResult{Project: item.proj}
			case pickerBare:
				m.result = &PickerResult{}
			case pickerManage:
				m.result = &PickerResult{Manage: true}
			case pickerConfigure:
				// Configure operates on the most recently highlighted project.
				// Walk backward from the current index to find the nearest project.
				proj := findNearestProject(m.list)
				if proj != nil {
					m.result = &PickerResult{Project: proj, Configure: true}
				} else {
					// No project available, fall back to manage TUI.
					m.result = &PickerResult{Manage: true}
				}
			}
			return m, tea.Quit

		case "c":
			// Configure shortcut: configure the currently highlighted project.
			item, ok := m.list.SelectedItem().(pickerItem)
			if ok && item.kind == pickerProject {
				m.result = &PickerResult{Project: item.proj, Configure: true}
				return m, tea.Quit
			}
			// Fallback: open management TUI.
			m.result = &PickerResult{Manage: true}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m PickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	var parts []string
	if m.showLogo {
		parts = append(parts, RenderLogo(m.width))
	}
	parts = append(parts, m.list.View())
	hintText := "enter: launch | c: configure | q: quit"
	if m.version != "" {
		hintText = fmt.Sprintf("%s  (%s)", hintText, m.version)
	}
	hint := statusBarStyle.Render(hintText)
	parts = append(parts, hint)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// findNearestProject walks the list items backward from the current index
// to find the nearest project item.
func findNearestProject(l list.Model) *model.Project {
	items := l.Items()
	idx := l.Index()
	for i := idx; i >= 0; i-- {
		if pi, ok := items[i].(pickerItem); ok && pi.kind == pickerProject {
			return pi.proj
		}
	}
	return nil
}

// logoHeight returns the number of lines consumed by the logo.
func logoHeight() int {
	return lipgloss.Height(RenderLogo(0))
}
