package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// promptItem implements list.DefaultItem for the prompts list.
type promptItem struct {
	prompt       model.PromptDef
	profileCount int
}

func (i promptItem) Title() string {
	title := i.prompt.Name
	if i.profileCount > 0 {
		title = fmt.Sprintf("%s \u2302%d", title, i.profileCount)
	}
	if i.prompt.Category != "" {
		title = fmt.Sprintf("%s [%s]", title, i.prompt.Category)
	}
	return title
}

func (i promptItem) Description() string {
	if i.prompt.Description != "" {
		return i.prompt.Description
	}
	return i.prompt.Source
}

func (i promptItem) FilterValue() string { return i.prompt.Name }

// PromptDeletedMsg is sent when a prompt has been deleted.
type PromptDeletedMsg struct{ Name string }

// RequestPromptFormMsg is sent to open the prompt add/edit overlay.
type RequestPromptFormMsg struct {
	EditPrompt *model.PromptDef // nil for add, non-nil for edit
}

// PromptsModel is the sub-model for the Prompts tab.
type PromptsModel struct {
	list       list.Model
	service    *service.Service
	width      int
	height     int
	confirming bool
	previewing bool
	preview    string
	err        error
}

// NewPromptsModel creates a new PromptsModel.
func NewPromptsModel(svc *service.Service) PromptsModel {
	items := buildPromptItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return PromptsModel{
		list:    l,
		service: svc,
	}
}

func buildPromptItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}
	profileCounts := svc.CountPromptProfileRefs()
	prompts := svc.ListPrompts()
	items := make([]list.Item, len(prompts))
	for i, p := range prompts {
		items[i] = promptItem{
			prompt:       p,
			profileCount: profileCounts[p.Name],
		}
	}
	return items
}

func (m PromptsModel) selectedPrompt() (model.PromptDef, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.PromptDef{}, false
	}
	pi, ok := item.(promptItem)
	if !ok {
		return model.PromptDef{}, false
	}
	return pi.prompt, true
}

// IsConsuming returns true when the model handles its own input.
func (m PromptsModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming || m.previewing
}

// SetSize updates the dimensions available to the Prompts tab.
func (m *PromptsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m PromptsModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	if m.previewing {
		return "esc: close preview"
	}
	return "a: add | e: edit | d: delete | v: preview | /: filter | tab: switch tabs | q: quit"
}

// Update handles messages for the Prompts tab.
func (m PromptsModel) Update(msg tea.Msg) (PromptsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.previewing {
			switch msg.String() {
			case "esc", "v", "q":
				m.previewing = false
				m.preview = ""
				return m, nil
			}
			return m, nil
		}

		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if prompt, ok := m.selectedPrompt(); ok {
					if err := m.service.DeletePrompt(prompt.Name); err != nil {
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

		switch msg.String() {
		case "a":
			return m, func() tea.Msg { return RequestPromptFormMsg{} }
		case "e":
			if prompt, ok := m.selectedPrompt(); ok {
				return m, func() tea.Msg { return RequestPromptFormMsg{EditPrompt: &prompt} }
			}
			return m, nil
		case "d":
			if _, ok := m.selectedPrompt(); ok {
				m.confirming = true
				m.err = nil
			}
			return m, nil
		case "v":
			if prompt, ok := m.selectedPrompt(); ok {
				content, err := os.ReadFile(prompt.Source)
				if err != nil {
					m.err = fmt.Errorf("read source: %w", err)
				} else {
					m.previewing = true
					m.preview = string(content)
					m.err = nil
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the Prompts tab as a horizontal split: list + detail.
func (m PromptsModel) View() string {
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

func (m PromptsModel) renderDetail(width, height int) string {
	prompt, ok := m.selectedPrompt()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No prompt selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(prompt.Name))
	b.WriteString("\n")
	b.WriteString("\n")
	writePromptFields(&b, prompt, detailLabelStyle)

	if m.previewing && m.preview != "" {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("Content:"))
		b.WriteString("\n")
		b.WriteString(m.preview)
	}

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", prompt.Name)))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

func (m *PromptsModel) refreshList() {
	items := buildPromptItems(m.service)
	m.list.SetItems(items)
}
