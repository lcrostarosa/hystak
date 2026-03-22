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

// skillItem implements list.DefaultItem for the skills list.
type skillItem struct {
	skill        model.SkillDef
	profileCount int
}

func (i skillItem) Title() string {
	if i.profileCount > 0 {
		return fmt.Sprintf("%s ⌂%d", i.skill.Name, i.profileCount)
	}
	return i.skill.Name
}

func (i skillItem) Description() string {
	if i.skill.Description != "" {
		return i.skill.Description
	}
	return i.skill.Source
}

func (i skillItem) FilterValue() string { return i.skill.Name }

// SkillDeletedMsg is sent when a skill has been deleted.
type SkillDeletedMsg struct{ Name string }

// SkillsModel is the sub-model for the Skills tab.
type SkillsModel struct {
	list       list.Model
	service    *service.Service
	keys       KeyMap
	width      int
	height     int
	confirming bool
	err        error
}

// NewSkillsModel creates a new SkillsModel.
func NewSkillsModel(svc *service.Service, keys KeyMap) SkillsModel {
	items := buildSkillItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return SkillsModel{
		list:    l,
		service: svc,
		keys:    keys,
	}
}

func buildSkillItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}

	profileCounts := svc.CountSkillProfileRefs()
	skills := svc.ListSkills()
	items := make([]list.Item, len(skills))
	for i, s := range skills {
		items[i] = skillItem{
			skill:        s,
			profileCount: profileCounts[s.Name],
		}
	}
	return items
}

func (m SkillsModel) selectedSkill() (model.SkillDef, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.SkillDef{}, false
	}
	si, ok := item.(skillItem)
	if !ok {
		return model.SkillDef{}, false
	}
	return si.skill, true
}

// IsConsuming returns true when the model handles its own input.
func (m SkillsModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming
}

// SetSize updates the dimensions available to the Skills tab.
func (m *SkillsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m SkillsModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	return fmt.Sprintf("%s: add | %s: edit | %s: delete | /: filter | %s | q: quit",
		m.keys.ResourceAdd.Help().Key, m.keys.ResourceEdit.Help().Key,
		m.keys.ResourceDelete.Help().Key, m.keys.tabNavHelp())
}

// Update handles messages for the Skills tab.
func (m SkillsModel) Update(msg tea.Msg) (SkillsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if skill, ok := m.selectedSkill(); ok {
					if err := m.service.DeleteSkill(skill.Name); err != nil {
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
			return m, func() tea.Msg { return RequestSkillFormMsg{} }
		case key.Matches(msg, m.keys.ResourceEdit):
			if skill, ok := m.selectedSkill(); ok {
				return m, func() tea.Msg { return RequestSkillFormMsg{EditSkill: &skill} }
			}
			return m, nil
		case key.Matches(msg, m.keys.ResourceDelete):
			if _, ok := m.selectedSkill(); ok {
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

// View renders the Skills tab as a horizontal split: list + detail.
func (m SkillsModel) View() string {
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

func (m SkillsModel) renderDetail(width, height int) string {
	skill, ok := m.selectedSkill()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No skill selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(skill.Name))
	b.WriteString("\n")
	b.WriteString("\n")
	writeSkillFields(&b, skill, detailLabelStyle)

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", skill.Name)))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

func (m *SkillsModel) refreshList() {
	items := buildSkillItems(m.service)
	m.list.SetItems(items)
}
