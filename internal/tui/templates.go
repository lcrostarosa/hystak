package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// templateItem implements list.DefaultItem for the templates list.
type templateItem struct {
	tmpl         model.TemplateDef
	profileCount int
}

func (i templateItem) Title() string {
	if i.profileCount > 0 {
		return fmt.Sprintf("%s ⌂%d", i.tmpl.Name, i.profileCount)
	}
	return i.tmpl.Name
}

func (i templateItem) Description() string {
	return i.tmpl.Source
}

func (i templateItem) FilterValue() string { return i.tmpl.Name }

// TemplateDeletedMsg is sent when a template has been deleted.
type TemplateDeletedMsg struct{ Name string }

// TemplatesModel is the sub-model for the Templates tab.
type TemplatesModel struct {
	list       list.Model
	service    *service.Service
	keys       KeyMap
	width      int
	height     int
	confirming bool
	err        error
}

// NewTemplatesModel creates a new TemplatesModel.
func NewTemplatesModel(svc *service.Service, keys KeyMap) TemplatesModel {
	items := buildTemplateItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return TemplatesModel{
		list:    l,
		service: svc,
		keys:    keys,
	}
}

func buildTemplateItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}

	profileCounts := svc.CountTemplateProfileRefs()
	tmpls := svc.ListTemplates()
	items := make([]list.Item, len(tmpls))
	for i, t := range tmpls {
		items[i] = templateItem{
			tmpl:         t,
			profileCount: profileCounts[t.Name],
		}
	}
	return items
}

func (m TemplatesModel) selectedTemplate() (model.TemplateDef, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.TemplateDef{}, false
	}
	ti, ok := item.(templateItem)
	if !ok {
		return model.TemplateDef{}, false
	}
	return ti.tmpl, true
}

// IsConsuming returns true when the model handles its own input.
func (m TemplatesModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming
}

// SetSize updates the dimensions available to the Templates tab.
func (m *TemplatesModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m TemplatesModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	return fmt.Sprintf("%s: add | %s: edit | %s: delete | /: filter | %s | q: quit",
		m.keys.ResourceAdd.Help().Key, m.keys.ResourceEdit.Help().Key,
		m.keys.ResourceDelete.Help().Key, m.keys.tabNavHelp())
}

// Update handles messages for the Templates tab.
func (m TemplatesModel) Update(msg tea.Msg) (TemplatesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if tmpl, ok := m.selectedTemplate(); ok {
					if err := m.service.DeleteTemplate(tmpl.Name); err != nil {
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
		case key.Matches(msg, m.keys.ResourceAdd):
			return m, func() tea.Msg { return RequestTemplateFormMsg{} }
		case key.Matches(msg, m.keys.ResourceEdit):
			if tmpl, ok := m.selectedTemplate(); ok {
				return m, func() tea.Msg { return RequestTemplateFormMsg{EditTemplate: &tmpl} }
			}
			return m, nil
		case key.Matches(msg, m.keys.ResourceDelete):
			if _, ok := m.selectedTemplate(); ok {
				m.confirming = true
				m.err = nil
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the Templates tab as a horizontal split: list + detail.
func (m TemplatesModel) View() string {
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

func (m TemplatesModel) renderDetail(width, height int) string {
	tmpl, ok := m.selectedTemplate()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No template selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(tmpl.Name))
	b.WriteString("\n")
	b.WriteString("\n")
	writeTemplateFields(&b, tmpl, detailLabelStyle)

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", tmpl.Name)))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

func (m *TemplatesModel) refreshList() {
	items := buildTemplateItems(m.service)
	m.list.SetItems(items)
}
