package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/service"
)

// SubNav identifies the active resource type within the Registry tab.
type SubNav int

const (
	SubNavMCPs SubNav = iota
	SubNavSkills
	SubNavHooks
	SubNavPermissions
	SubNavTemplates
	SubNavPrompts
	subNavCount
)

var subNavNames = [subNavCount]string{
	"MCPs", "Skills", "Hooks", "Permissions", "Templates", "Prompts",
}

// registryMode tracks the current interaction mode.
type registryMode int

const (
	modeList registryMode = iota
	modeFilter
	modeForm
	modeConfirm
)

// registryTab is the Registry tab — shows all managed resources with sub-navigation.
// Uses the subView abstraction to handle all 6 resource types uniformly.
type registryTab struct {
	keys   KeyMap
	svc    *service.Service
	sub    SubNav
	mode   registryMode
	width  int
	height int
	err    string // transient error message

	// List state (generic across all sub-navs)
	items    []listItem
	filtered []listItem
	cursor   int
	selected map[string]bool

	// Filter state
	filterInput textinput.Model
	filterText  string

	// Form state
	form     formModel
	editName string

	// Confirm state
	confirm     confirmModel
	deleteName  string
	deleteNames []string
}

func newRegistryTab(keys KeyMap, svc *service.Service) *registryTab {
	fi := textinput.New()
	fi.Placeholder = "type to filter..."
	fi.CharLimit = 64
	return &registryTab{
		keys:        keys,
		svc:         svc,
		selected:    make(map[string]bool),
		filterInput: fi,
	}
}

func (t *registryTab) Title() string { return "Registry" }

func (t *registryTab) HelpKeys() []HelpEntry {
	switch t.mode {
	case modeFilter:
		return []HelpEntry{{"Esc", "Clear filter"}, {"Enter", "Apply"}}
	case modeForm:
		return []HelpEntry{{"Tab", "Next field"}, {"Enter", "Save"}, {"Esc", "Cancel"}}
	case modeConfirm:
		return []HelpEntry{{"Y", "Confirm"}, {"N/Esc", "Cancel"}}
	default:
		return []HelpEntry{
			{"A", "Add"}, {"E", "Edit"}, {"D", "Delete"},
			{"/", "Filter"}, {"Space", "Select"},
		}
	}
}

// --- Messages ---

type registryLoadedMsg struct{ items []listItem }
type serverSavedMsg struct{}
type serverDeletedMsg struct{}
type serverErrorMsg struct{ err error }

// --- Init / Update ---

func (t *registryTab) Init() tea.Cmd {
	return t.loadData
}

func (t *registryTab) loadData() tea.Msg {
	sv := subViews[t.sub]
	return registryLoadedMsg{items: sv.loadItems(t.svc)}
}

func (t *registryTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case registryLoadedMsg:
		t.items = msg.items
		t.applyFilter()
		t.clampCursor()
		t.err = ""
		return t, nil

	case serverSavedMsg:
		t.mode = modeList
		t.editName = ""
		return t, t.loadData

	case serverDeletedMsg:
		t.mode = modeList
		t.deleteName = ""
		t.deleteNames = nil
		t.selected = make(map[string]bool)
		return t, t.loadData

	case serverErrorMsg:
		t.err = msg.err.Error()
		t.mode = modeList
		return t, nil

	case formSubmitMsg:
		return t, t.handleFormSubmit(msg.values)

	case formCancelMsg:
		t.mode = modeList
		t.editName = ""
		return t, nil

	case confirmYesMsg:
		return t, t.handleConfirmYes()

	case confirmNoMsg:
		t.mode = modeList
		t.deleteName = ""
		t.deleteNames = nil
		return t, nil

	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		switch t.mode {
		case modeFilter:
			return t.handleFilterKey(msg)
		case modeForm:
			return t.handleFormKey(msg)
		case modeConfirm:
			return t.handleConfirmKey(msg)
		default:
			return t.handleListKey(msg)
		}
	}
	return t, nil
}

// --- List mode ---

func (t *registryTab) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, t.keys.ListUp):
		if t.cursor > 0 {
			t.cursor--
		}
	case key.Matches(msg, t.keys.ListDown):
		if t.cursor < len(t.filtered)-1 {
			t.cursor++
		}
	case key.Matches(msg, t.keys.Select):
		if len(t.filtered) > 0 {
			name := t.filtered[t.cursor].name
			t.selected[name] = !t.selected[name]
			if !t.selected[name] {
				delete(t.selected, name)
			}
		}
	case key.Matches(msg, t.keys.Add):
		t.openAddForm()
		return t, textinput.Blink
	case key.Matches(msg, t.keys.Edit):
		if len(t.filtered) > 0 {
			t.openEditForm(t.filtered[t.cursor].name)
			return t, textinput.Blink
		}
	case key.Matches(msg, t.keys.Delete):
		return t, t.openDelete()
	case key.Matches(msg, t.keys.Filter):
		t.mode = modeFilter
		t.filterInput.SetValue(t.filterText)
		return t, t.filterInput.Focus()
	case msg.String() == "left" || msg.String() == "shift+tab":
		if t.sub > 0 {
			t.sub--
			t.clearFilter()
			return t, t.loadData
		}
	case msg.String() == "right" || msg.String() == "tab":
		if t.sub < subNavCount-1 {
			t.sub++
			t.clearFilter()
			return t, t.loadData
		}
	}
	return t, nil
}

// --- Filter mode ---

func (t *registryTab) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.clearFilter()
		t.mode = modeList
		return t, nil
	case "enter":
		t.filterText = t.filterInput.Value()
		t.applyFilter()
		t.clampCursor()
		t.mode = modeList
		return t, nil
	}
	var cmd tea.Cmd
	t.filterInput, cmd = t.filterInput.Update(msg)
	t.filterText = t.filterInput.Value()
	t.applyFilter()
	t.clampCursor()
	return t, cmd
}

// --- Form mode ---

func (t *registryTab) openAddForm() {
	sv := subViews[t.sub]
	t.editName = ""
	title := "Add " + subNavNames[t.sub][:len(subNavNames[t.sub])-1] // strip trailing 's'
	t.form = newFormModel(title, sv.addFields(), t.keys)
	t.mode = modeForm
}

func (t *registryTab) openEditForm(name string) {
	sv := subViews[t.sub]
	fields := sv.editFields(name, t.svc)
	if fields == nil {
		return
	}
	t.editName = name
	title := "Edit " + subNavNames[t.sub][:len(subNavNames[t.sub])-1]
	t.form = newFormModel(title, fields, t.keys)
	t.mode = modeForm
}

func (t *registryTab) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.form, cmd = t.form.Update(msg)
	return t, cmd
}

func (t *registryTab) handleFormSubmit(values map[string]string) tea.Cmd {
	sv := subViews[t.sub]
	editName := t.editName
	svc := t.svc
	saveFn := sv.save
	return func() tea.Msg {
		if err := saveFn(svc, values, editName); err != nil {
			return serverErrorMsg{err: err}
		}
		return serverSavedMsg{}
	}
}

// --- Confirm/Delete mode ---

func (t *registryTab) openDelete() tea.Cmd {
	kind := subNavNames[t.sub]
	if len(t.selected) > 0 {
		names := make([]string, 0, len(t.selected))
		for name := range t.selected {
			names = append(names, name)
		}
		t.deleteNames = names
		t.confirm = newConfirmModel(
			"Delete "+kind,
			fmt.Sprintf("Delete %d selected item(s)?", len(names)),
		)
		t.mode = modeConfirm
		return nil
	}
	if len(t.filtered) == 0 {
		return nil
	}
	name := t.filtered[t.cursor].name
	t.deleteName = name
	t.confirm = newConfirmModel(
		"Delete "+kind[:len(kind)-1],
		fmt.Sprintf("Delete %q?", name),
	)
	t.mode = modeConfirm
	return nil
}

func (t *registryTab) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.confirm, cmd = t.confirm.Update(msg)
	return t, cmd
}

func (t *registryTab) handleConfirmYes() tea.Cmd {
	sv := subViews[t.sub]
	svc := t.svc
	deleteFn := sv.delete
	names := t.deleteNames
	if len(names) == 0 && t.deleteName != "" {
		names = []string{t.deleteName}
	}
	return func() tea.Msg {
		for _, name := range names {
			if err := deleteFn(svc, name); err != nil {
				return serverErrorMsg{err: err}
			}
		}
		return serverDeletedMsg{}
	}
}

// --- Filter helpers ---

func (t *registryTab) applyFilter() {
	if t.filterText == "" {
		t.filtered = t.items
		return
	}
	lower := strings.ToLower(t.filterText)
	filtered := make([]listItem, 0, len(t.items))
	for _, item := range t.items {
		if strings.Contains(strings.ToLower(item.name), lower) {
			filtered = append(filtered, item)
		}
	}
	t.filtered = filtered
}

func (t *registryTab) clearFilter() {
	t.filterText = ""
	t.filterInput.SetValue("")
	t.filtered = t.items
}

func (t *registryTab) clampCursor() {
	if t.cursor >= len(t.filtered) {
		t.cursor = max(0, len(t.filtered)-1)
	}
}

// --- View ---

func (t *registryTab) View() string {
	switch t.mode {
	case modeForm:
		return t.form.View(t.width, t.height)
	case modeConfirm:
		return t.confirm.View(t.width, t.height)
	}

	var b strings.Builder

	// Sub-nav bar
	for i := SubNav(0); i < subNavCount; i++ {
		name := subNavNames[i]
		if i == t.sub {
			b.WriteString(styleTabActive.Render("[" + name + "]"))
		} else {
			b.WriteString(styleTabInactive.Render(" " + name + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	if t.err != "" {
		b.WriteString("  " + styleError.Render("Error: "+t.err) + "\n\n")
	}

	// Filter bar
	if t.mode == modeFilter {
		b.WriteString("  Filter: " + t.filterInput.View() + "\n\n")
	} else if t.filterText != "" {
		b.WriteString("  " + styleHelpDesc.Render(fmt.Sprintf("Filter: %q (%d/%d)", t.filterText, len(t.filtered), len(t.items))) + "\n\n")
	}

	// Column header
	sv := subViews[t.sub]
	b.WriteString(styleListHeader.Render(sv.header))
	b.WriteString("\n")

	if len(t.filtered) == 0 {
		if t.filterText != "" {
			b.WriteString("  (no matches)\n")
		} else {
			b.WriteString("  (none registered)\n")
		}
		return b.String()
	}

	for i, item := range t.filtered {
		sel := "  "
		if t.selected[item.name] {
			sel = styleSynced.Render("* ")
		}
		line := "  " + sel + formatColumns(item.columns)
		if i == t.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(styleListNormal.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// formatColumns joins columns with appropriate padding.
func formatColumns(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	// Use fixed-width formatting matching the header
	var b strings.Builder
	for i, c := range cols {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(c)
	}
	return b.String()
}

// --- Parsing helpers ---

func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseKV(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	result := make(map[string]string)
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func formatEnv(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + m[k]
	}
	return strings.Join(parts, ", ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
