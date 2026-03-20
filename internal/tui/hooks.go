package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// hookItem implements list.DefaultItem for the hooks list.
type hookItem struct {
	hook         model.HookDef
	profileCount int
}

func (i hookItem) Title() string {
	if i.profileCount > 0 {
		return fmt.Sprintf("%s ⌂%d", i.hook.Name, i.profileCount)
	}
	return i.hook.Name
}

func (i hookItem) Description() string {
	return fmt.Sprintf("%s: %s", i.hook.Event, i.hook.Command)
}

func (i hookItem) FilterValue() string { return i.hook.Name }

// HookDeletedMsg is sent when a hook has been deleted.
type HookDeletedMsg struct{ Name string }

// HooksModel is the sub-model for the Hooks tab.
type HooksModel struct {
	list       list.Model
	service    *service.Service
	width      int
	height     int
	confirming bool
	err        error
}

// NewHooksModel creates a new HooksModel.
func NewHooksModel(svc *service.Service) HooksModel {
	items := buildHookItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return HooksModel{
		list:    l,
		service: svc,
	}
}

func buildHookItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}

	profileCounts := svc.CountHookProfileRefs()
	hooks := svc.ListHooks()
	items := make([]list.Item, len(hooks))
	for i, h := range hooks {
		items[i] = hookItem{
			hook:         h,
			profileCount: profileCounts[h.Name],
		}
	}
	return items
}

func (m HooksModel) selectedHook() (model.HookDef, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.HookDef{}, false
	}
	hi, ok := item.(hookItem)
	if !ok {
		return model.HookDef{}, false
	}
	return hi.hook, true
}

// IsConsuming returns true when the model handles its own input.
func (m HooksModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming
}

// SetSize updates the dimensions available to the Hooks tab.
func (m *HooksModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m HooksModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	return "a: add | e: edit | d: delete | /: filter | tab: switch tabs | q: quit"
}

// Update handles messages for the Hooks tab.
func (m HooksModel) Update(msg tea.Msg) (HooksModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if hook, ok := m.selectedHook(); ok {
					if err := m.service.DeleteHook(hook.Name); err != nil {
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
			return m, func() tea.Msg { return RequestHookFormMsg{} }
		case "e":
			if hook, ok := m.selectedHook(); ok {
				return m, func() tea.Msg { return RequestHookFormMsg{EditHook: &hook} }
			}
			return m, nil
		case "d":
			if _, ok := m.selectedHook(); ok {
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

// View renders the Hooks tab as a horizontal split: list + detail.
func (m HooksModel) View() string {
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

func (m HooksModel) renderDetail(width, height int) string {
	hook, ok := m.selectedHook()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No hook selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(hook.Name))
	b.WriteString("\n")
	b.WriteString("\n")
	writeHookFields(&b, hook, detailLabelStyle)

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", hook.Name)))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

func (m *HooksModel) refreshList() {
	items := buildHookItems(m.service)
	m.list.SetItems(items)
}
