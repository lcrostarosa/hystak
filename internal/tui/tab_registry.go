package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hystak/hystak/internal/model"
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
type registryTab struct {
	keys   KeyMap
	svc    *service.Service
	sub    SubNav
	mode   registryMode
	width  int
	height int
	err    string // transient error message

	// List state
	servers  []model.ServerDef
	filtered []model.ServerDef
	cursor   int
	selected map[string]bool // multi-select by name

	// Filter state
	filterInput textinput.Model
	filterText  string

	// Form state
	form     formModel
	editName string // non-empty means editing existing server

	// Confirm state
	confirm     confirmModel
	deleteName  string
	deleteNames []string // batch delete
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
		return []HelpEntry{
			{"Esc", "Clear filter"},
			{"Enter", "Apply"},
		}
	case modeForm:
		return []HelpEntry{
			{"Tab", "Next field"},
			{"Enter", "Save"},
			{"Esc", "Cancel"},
		}
	case modeConfirm:
		return []HelpEntry{
			{"Y", "Confirm"},
			{"N/Esc", "Cancel"},
		}
	default:
		return []HelpEntry{
			{"A", "Add"},
			{"E", "Edit"},
			{"D", "Delete"},
			{"/", "Filter"},
			{"Space", "Select"},
		}
	}
}

// registryLoadedMsg is sent when registry data has been loaded asynchronously.
type registryLoadedMsg struct {
	servers []model.ServerDef
}

// serverSavedMsg is sent after a server has been successfully added/updated.
type serverSavedMsg struct{}

// serverDeletedMsg is sent after server(s) have been successfully deleted.
type serverDeletedMsg struct{}

// serverErrorMsg is sent when a CRUD operation fails.
type serverErrorMsg struct {
	err error
}

func (t *registryTab) Init() tea.Cmd {
	return t.loadData
}

func (t *registryTab) loadData() tea.Msg {
	return registryLoadedMsg{
		servers: t.svc.ListServers(),
	}
}

func (t *registryTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case registryLoadedMsg:
		t.servers = msg.servers
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
			name := t.filtered[t.cursor].Name
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
			t.openEditForm(t.filtered[t.cursor])
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
	// Live filtering as user types
	t.filterText = t.filterInput.Value()
	t.applyFilter()
	t.clampCursor()
	return t, cmd
}

// --- Form mode ---

func (t *registryTab) openAddForm() {
	t.editName = ""
	t.form = newFormModel("Add MCP Server", []FormField{
		{Label: "Name", Placeholder: "server-name"},
		{Label: "Transport", Placeholder: "stdio | sse | http", Value: "stdio"},
		{Label: "Command", Placeholder: "npx"},
		{Label: "Args", Placeholder: "-y, @anthropic/mcp-github"},
		{Label: "URL", Placeholder: "https://..."},
		{Label: "Env", Placeholder: "KEY=val, KEY2=val2"},
		{Label: "Description", Placeholder: "optional description"},
	}, t.keys)
	t.mode = modeForm
}

func (t *registryTab) openEditForm(srv model.ServerDef) {
	t.editName = srv.Name
	t.form = newFormModel("Edit MCP Server", []FormField{
		{Label: "Name", Placeholder: "server-name", Value: srv.Name},
		{Label: "Transport", Placeholder: "stdio | sse | http", Value: string(srv.Transport)},
		{Label: "Command", Placeholder: "npx", Value: srv.Command},
		{Label: "Args", Placeholder: "-y, @anthropic/mcp-github", Value: strings.Join(srv.Args, ", ")},
		{Label: "URL", Placeholder: "https://...", Value: srv.URL},
		{Label: "Env", Placeholder: "KEY=val, KEY2=val2", Value: formatEnv(srv.Env)},
		{Label: "Description", Placeholder: "optional description", Value: srv.Description},
	}, t.keys)
	t.mode = modeForm
}

func (t *registryTab) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	t.form, cmd = t.form.Update(msg)
	return t, cmd
}

func (t *registryTab) handleFormSubmit(values map[string]string) tea.Cmd {
	editName := t.editName
	svc := t.svc
	return func() tea.Msg {
		transport := model.Transport(strings.TrimSpace(values["Transport"]))
		if !transport.Valid() {
			return serverErrorMsg{err: fmt.Errorf("invalid transport %q", transport)}
		}

		srv := model.ServerDef{
			Name:        strings.TrimSpace(values["Name"]),
			Transport:   transport,
			Command:     strings.TrimSpace(values["Command"]),
			Args:        parseCSV(values["Args"]),
			URL:         strings.TrimSpace(values["URL"]),
			Env:         parseKV(values["Env"]),
			Description: strings.TrimSpace(values["Description"]),
		}

		if editName != "" {
			if err := svc.UpdateServer(srv); err != nil {
				return serverErrorMsg{err: err}
			}
		} else {
			if err := svc.AddServer(srv); err != nil {
				return serverErrorMsg{err: err}
			}
		}
		return serverSavedMsg{}
	}
}

// --- Confirm/Delete mode ---

func (t *registryTab) openDelete() tea.Cmd {
	// Batch delete if anything selected
	if len(t.selected) > 0 {
		names := make([]string, 0, len(t.selected))
		for name := range t.selected {
			names = append(names, name)
		}
		t.deleteNames = names
		t.confirm = newConfirmModel(
			"Delete Servers",
			fmt.Sprintf("Delete %d selected server(s)?", len(names)),
		)
		t.mode = modeConfirm
		return nil
	}

	// Single delete
	if len(t.filtered) == 0 {
		return nil
	}
	name := t.filtered[t.cursor].Name
	t.deleteName = name
	t.confirm = newConfirmModel(
		"Delete Server",
		fmt.Sprintf("Delete server %q?", name),
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
	svc := t.svc
	names := t.deleteNames
	if len(names) == 0 && t.deleteName != "" {
		names = []string{t.deleteName}
	}
	return func() tea.Msg {
		for _, name := range names {
			if err := svc.DeleteServer(name); err != nil {
				return serverErrorMsg{err: err}
			}
		}
		return serverDeletedMsg{}
	}
}

// --- Filter helpers ---

func (t *registryTab) applyFilter() {
	if t.filterText == "" {
		t.filtered = t.servers
		return
	}
	lower := strings.ToLower(t.filterText)
	filtered := make([]model.ServerDef, 0, len(t.servers))
	for _, s := range t.servers {
		if strings.Contains(strings.ToLower(s.Name), lower) {
			filtered = append(filtered, s)
		}
	}
	t.filtered = filtered
}

func (t *registryTab) clearFilter() {
	t.filterText = ""
	t.filterInput.SetValue("")
	t.filtered = t.servers
}

func (t *registryTab) clampCursor() {
	if t.cursor >= len(t.filtered) {
		t.cursor = max(0, len(t.filtered)-1)
	}
}

// --- View ---

func (t *registryTab) View() string {
	// If a modal is active, render it over the list
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

	// Error message
	if t.err != "" {
		b.WriteString("  " + styleError.Render("Error: "+t.err) + "\n\n")
	}

	// Non-MCPs sub-nav: placeholder
	if t.sub != SubNavMCPs {
		b.WriteString(styleListHeader.Render(
			fmt.Sprintf("  %s — coming soon", subNavNames[t.sub]),
		))
		b.WriteString("\n")
		return b.String()
	}

	// Filter bar
	if t.mode == modeFilter {
		b.WriteString("  Filter: " + t.filterInput.View() + "\n\n")
	} else if t.filterText != "" {
		b.WriteString("  " + styleHelpDesc.Render(fmt.Sprintf("Filter: %q (%d/%d)", t.filterText, len(t.filtered), len(t.servers))) + "\n\n")
	}

	// Column header
	b.WriteString(styleListHeader.Render(
		fmt.Sprintf("  %-2s %-20s  %-10s  %s", "", "NAME", "TRANSPORT", "COMMAND/URL"),
	))
	b.WriteString("\n")

	if len(t.filtered) == 0 {
		if t.filterText != "" {
			b.WriteString("  (no matches)\n")
		} else {
			b.WriteString("  (no servers registered)\n")
		}
		return b.String()
	}

	for i, s := range t.filtered {
		endpoint := s.Command
		if s.Transport == model.TransportSSE || s.Transport == model.TransportHTTP {
			endpoint = s.URL
		}
		sel := "  "
		if t.selected[s.Name] {
			sel = styleSynced.Render("* ")
		}
		line := fmt.Sprintf("  %s%-20s  %-10s  %s", sel, truncate(s.Name, 20), s.Transport, truncate(endpoint, 40))
		if i == t.cursor {
			b.WriteString(styleListSelected.Render(line))
		} else {
			b.WriteString(styleListNormal.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// --- Parsing helpers ---

// parseCSV splits a comma-separated string into trimmed parts.
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

// parseKV parses "KEY=val, KEY2=val2" into a map.
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

// formatEnv formats a map as "KEY=val, KEY2=val2" in sorted key order.
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

// truncate shortens a string to max length, appending "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
