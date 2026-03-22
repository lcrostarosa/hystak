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

// permissionItem implements list.DefaultItem for the permissions list.
type permissionItem struct {
	perm         model.PermissionRule
	profileCount int
}

func (i permissionItem) Title() string {
	if i.profileCount > 0 {
		return fmt.Sprintf("%s ⌂%d", i.perm.Name, i.profileCount)
	}
	return i.perm.Name
}

func (i permissionItem) Description() string {
	return fmt.Sprintf("%s: %s", i.perm.EffectiveType(), i.perm.Rule)
}

func (i permissionItem) FilterValue() string { return i.perm.Name }

// PermissionDeletedMsg is sent when a permission has been deleted.
type PermissionDeletedMsg struct{ Name string }

// PermissionsModel is the sub-model for the Permissions tab.
type PermissionsModel struct {
	list       list.Model
	service    *service.Service
	keys       KeyMap
	width      int
	height     int
	confirming bool
	err        error
}

// NewPermissionsModel creates a new PermissionsModel.
func NewPermissionsModel(svc *service.Service, keys KeyMap) PermissionsModel {
	items := buildPermissionItems(svc)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return PermissionsModel{
		list:    l,
		service: svc,
		keys:    keys,
	}
}

func buildPermissionItems(svc *service.Service) []list.Item {
	if svc == nil {
		return nil
	}

	profileCounts := svc.CountPermissionProfileRefs()
	perms := svc.ListPermissions()
	items := make([]list.Item, len(perms))
	for i, p := range perms {
		items[i] = permissionItem{
			perm:         p,
			profileCount: profileCounts[p.Name],
		}
	}
	return items
}

func (m PermissionsModel) selectedPermission() (model.PermissionRule, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return model.PermissionRule{}, false
	}
	pi, ok := item.(permissionItem)
	if !ok {
		return model.PermissionRule{}, false
	}
	return pi.perm, true
}

// IsConsuming returns true when the model handles its own input.
func (m PermissionsModel) IsConsuming() bool {
	return m.list.FilterState() == list.Filtering || m.confirming
}

// SetSize updates the dimensions available to the Permissions tab.
func (m *PermissionsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	listWidth := w * 2 / 5
	if listWidth < 20 {
		listWidth = 20
	}
	m.list.SetSize(listWidth, h)
}

// StatusHelp returns context-sensitive help text for the status bar.
func (m PermissionsModel) StatusHelp() string {
	if m.confirming {
		return "y: confirm delete | n: cancel"
	}
	return fmt.Sprintf("%s: add | %s: edit | %s: delete | /: filter | %s | q: quit",
		m.keys.ResourceAdd.Help().Key, m.keys.ResourceEdit.Help().Key,
		m.keys.ResourceDelete.Help().Key, m.keys.tabNavHelp())
}

// Update handles messages for the Permissions tab.
func (m PermissionsModel) Update(msg tea.Msg) (PermissionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.err = nil
				if perm, ok := m.selectedPermission(); ok {
					if err := m.service.DeletePermission(perm.Name); err != nil {
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
			return m, func() tea.Msg { return RequestPermissionFormMsg{} }
		case key.Matches(msg, m.keys.ResourceEdit):
			if perm, ok := m.selectedPermission(); ok {
				return m, func() tea.Msg { return RequestPermissionFormMsg{EditPermission: &perm} }
			}
			return m, nil
		case key.Matches(msg, m.keys.ResourceDelete):
			if _, ok := m.selectedPermission(); ok {
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

// View renders the Permissions tab as a horizontal split: list + detail.
func (m PermissionsModel) View() string {
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

func (m PermissionsModel) renderDetail(width, height int) string {
	perm, ok := m.selectedPermission()
	if !ok {
		return detailPaneStyle.Width(width).Height(height).Render("No permission selected")
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(perm.Name))
	b.WriteString("\n")
	b.WriteString("\n")
	writePermissionFields(&b, perm, detailLabelStyle)

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %q? (y/n)", perm.Name)))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
	}

	return detailPaneStyle.Width(width).Height(height).Render(b.String())
}

func (m *PermissionsModel) refreshList() {
	items := buildPermissionItems(m.service)
	m.list.SetItems(items)
}
