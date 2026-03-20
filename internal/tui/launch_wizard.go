package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/discovery"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
)

// launchStep tracks the current step in the launch wizard.
type launchStep int

const (
	launchStepMCPs launchStep = iota
	launchStepSkills
	launchStepPermissions
	launchStepHooks
	launchStepClaudeMD
	launchStepEnvVars
	launchStepIsolation
	launchStepCount // sentinel for step count
)

var launchStepLabels = []string{
	"MCPs",
	"Skills",
	"Permissions",
	"Hooks",
	"CLAUDE.md",
	"Env Vars",
	"Isolation",
}

// LaunchWizardMode is the entry mode for the wizard.
type LaunchWizardMode int

const (
	LWModeSequential LaunchWizardMode = iota // first launch: walk through all
	LWModeHub                                // on-demand: jump to any category
)

// wizardPhase tracks the high-level phase of the wizard.
type wizardPhase int

const (
	phaseSteps     wizardPhase = iota // editing categories (sequential or hub)
	phaseChecklist                    // review & confirm
)

// RequestLaunchWizardMsg triggers the launch wizard overlay.
type RequestLaunchWizardMsg struct {
	Project     *model.Project
	ProjectPath string
	Mode        LaunchWizardMode
	Discovered  *discovery.Items
}

// LaunchWizardCompleteMsg is emitted when the wizard finishes.
type LaunchWizardCompleteMsg struct {
	Profile profile.Profile
	Launch  bool // true = sync & launch, false = save only
}

// LaunchWizardCancelledMsg is emitted when the wizard is cancelled.
type LaunchWizardCancelledMsg struct{}

// LaunchWizardModel is the Bubble Tea model for the launch wizard.
type LaunchWizardModel struct {
	project    *model.Project
	mode       LaunchWizardMode
	phase      wizardPhase
	step       launchStep
	discovered *discovery.Items
	width      int
	height     int

	// Per-step cursors
	cursors [launchStepCount]int

	// Selections (keyed by item name)
	mcpSelections   map[string]bool
	skillSelections map[string]bool
	permSelections  map[string]bool
	hookSelections  map[string]bool

	// CLAUDE.md: index into discovered templates, -1 = none
	claudeMDOptions []string // available template names/paths
	claudeMDCursor  int
	claudeMDChoice  int // selected index, -1 = none

	// Env vars: key-value pairs
	envKeys    []string
	envValues  []string
	envCursor  int
	envEditing bool // true when editing a value
	envField   int  // 0 = key, 1 = value

	// Isolation
	isolationCursor int
	isolation       profile.IsolationStrategy
}

var isolationOptions = []struct {
	strategy profile.IsolationStrategy
	label    string
	desc     string
}{
	{profile.IsolationNone, "none", "Deploy to project root. One session at a time."},
	{profile.IsolationWorktree, "worktree", "Each launch gets a git worktree with isolated configs."},
	{profile.IsolationLock, "lock", "Deploy to project root, prevent concurrent launches."},
}

// NewLaunchWizardModel creates a new launch wizard.
func NewLaunchWizardModel(
	proj *model.Project,
	mode LaunchWizardMode,
	discovered *discovery.Items,
	existingProfile *profile.Profile,
) LaunchWizardModel {
	m := LaunchWizardModel{
		project:         proj,
		mode:            mode,
		phase:           phaseSteps,
		step:            launchStepMCPs,
		discovered:      discovered,
		mcpSelections:   make(map[string]bool),
		skillSelections: make(map[string]bool),
		permSelections:  make(map[string]bool),
		hookSelections:  make(map[string]bool),
		claudeMDChoice:  -1,
		isolation:       profile.IsolationNone,
	}

	// Build CLAUDE.md options from discovered skills that look like templates
	// For now, "none" is always available, plus any template from the registry
	// We'll just use the template names if available
	m.claudeMDOptions = []string{"(none)"}

	// Pre-populate selections from existing profile if provided
	if existingProfile != nil {
		for _, name := range existingProfile.MCPs {
			m.mcpSelections[name] = true
		}
		for _, name := range existingProfile.Skills {
			m.skillSelections[name] = true
		}
		for _, name := range existingProfile.Permissions {
			m.permSelections[name] = true
		}
		for _, name := range existingProfile.Hooks {
			m.hookSelections[name] = true
		}
		for k, v := range existingProfile.EnvVars {
			m.envKeys = append(m.envKeys, k)
			m.envValues = append(m.envValues, v)
		}
		if existingProfile.ClaudeMD != "" {
			m.claudeMDOptions = append(m.claudeMDOptions, existingProfile.ClaudeMD)
			m.claudeMDChoice = 1
		}
		m.isolation = existingProfile.Isolation
		for i, opt := range isolationOptions {
			if opt.strategy == m.isolation {
				m.isolationCursor = i
				break
			}
		}
	}

	return m
}

// SetSize updates the wizard dimensions.
func (m *LaunchWizardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Init implements tea.Model.
func (m LaunchWizardModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m LaunchWizardModel) Update(msg tea.Msg) (LaunchWizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m LaunchWizardModel) handleKey(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	// Global cancel
	if msg.String() == "ctrl+c" {
		return m, func() tea.Msg { return LaunchWizardCancelledMsg{} }
	}

	switch m.phase {
	case phaseChecklist:
		return m.handleChecklistKey(msg)
	default:
		if m.mode == LWModeHub {
			return m.handleHubKey(msg)
		}
		return m.handleSequentialKey(msg)
	}
}

func (m LaunchWizardModel) handleSequentialKey(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.step == launchStepMCPs {
			return m, func() tea.Msg { return LaunchWizardCancelledMsg{} }
		}
		m.step--
		return m, nil

	case "enter":
		if m.step < launchStepCount-1 {
			m.step++
			return m, nil
		}
		// Last step → go to checklist
		m.phase = phaseChecklist
		return m, nil

	case "tab":
		if m.step < launchStepCount-1 {
			m.step++
		}
		return m, nil
	}

	return m.handleStepInput(msg)
}

func (m LaunchWizardModel) handleHubKey(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return LaunchWizardCancelledMsg{} }
	case "tab":
		m.step = launchStep((int(m.step) + 1) % int(launchStepCount))
		return m, nil
	case "shift+tab":
		m.step = launchStep((int(m.step) - 1 + int(launchStepCount)) % int(launchStepCount))
		return m, nil
	case "enter":
		m.phase = phaseChecklist
		return m, nil
	}

	return m.handleStepInput(msg)
}

func (m LaunchWizardModel) handleChecklistKey(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "ctrl+l":
		return m, func() tea.Msg {
			return LaunchWizardCompleteMsg{
				Profile: m.buildProfile(),
				Launch:  true,
			}
		}
	case "e":
		// Back to hub mode for editing
		m.phase = phaseSteps
		m.mode = LWModeHub
		return m, nil
	case "esc":
		return m, func() tea.Msg { return LaunchWizardCancelledMsg{} }
	}
	return m, nil
}

// handleStepInput delegates to the step-specific handler for the current step.
func (m LaunchWizardModel) handleStepInput(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	switch m.step {
	case launchStepMCPs:
		return m.updateMultiSelect(msg, m.mcpItems(), m.mcpSelections)
	case launchStepSkills:
		return m.updateMultiSelect(msg, m.skillItems(), m.skillSelections)
	case launchStepPermissions:
		return m.updateMultiSelect(msg, m.permItems(), m.permSelections)
	case launchStepHooks:
		return m.updateMultiSelect(msg, m.hookItems(), m.hookSelections)
	case launchStepClaudeMD:
		return m.updateRadioSelect(msg, len(m.claudeMDOptions), &m.claudeMDCursor, &m.claudeMDChoice)
	case launchStepEnvVars:
		return m.updateEnvVars(msg)
	case launchStepIsolation:
		return m.updateIsolation(msg)
	}
	return m, nil
}

// updateMultiSelect handles up/down/space for multi-select lists.
func (m LaunchWizardModel) updateMultiSelect(msg tea.KeyMsg, items []wizardItem, selections map[string]bool) (LaunchWizardModel, tea.Cmd) {
	n := len(items)
	if n == 0 {
		return m, nil
	}
	cursor := &m.cursors[m.step]
	switch msg.String() {
	case "up", "k":
		*cursor = moveCursor(*cursor, -1, n)
	case "down", "j":
		*cursor = moveCursor(*cursor, 1, n)
	case " ":
		if *cursor < n {
			name := items[*cursor].name
			selections[name] = !selections[name]
		}
	case "a":
		// Toggle all
		allSelected := true
		for _, item := range items {
			if !selections[item.name] {
				allSelected = false
				break
			}
		}
		for _, item := range items {
			selections[item.name] = !allSelected
		}
	}
	return m, nil
}

// updateRadioSelect handles up/down/space for radio select.
func (m LaunchWizardModel) updateRadioSelect(msg tea.KeyMsg, n int, cursor *int, choice *int) (LaunchWizardModel, tea.Cmd) {
	if n == 0 {
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		*cursor = moveCursor(*cursor, -1, n)
	case "down", "j":
		*cursor = moveCursor(*cursor, 1, n)
	case " ":
		*choice = *cursor
	}
	return m, nil
}

// updateEnvVars handles the env vars step.
func (m LaunchWizardModel) updateEnvVars(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	n := len(m.envKeys)
	switch msg.String() {
	case "up", "k":
		if n > 0 {
			m.envCursor = moveCursor(m.envCursor, -1, n)
		}
	case "down", "j":
		if n > 0 {
			m.envCursor = moveCursor(m.envCursor, 1, n)
		}
	case "ctrl+a":
		// Add new env var
		m.envKeys = append(m.envKeys, "")
		m.envValues = append(m.envValues, "")
		m.envCursor = len(m.envKeys) - 1
	case "ctrl+d":
		// Delete current env var
		if n > 0 && m.envCursor < n {
			m.envKeys = append(m.envKeys[:m.envCursor], m.envKeys[m.envCursor+1:]...)
			m.envValues = append(m.envValues[:m.envCursor], m.envValues[m.envCursor+1:]...)
			if m.envCursor >= len(m.envKeys) && m.envCursor > 0 {
				m.envCursor--
			}
		}
	}
	return m, nil
}

// updateIsolation handles the isolation strategy step.
func (m LaunchWizardModel) updateIsolation(msg tea.KeyMsg) (LaunchWizardModel, tea.Cmd) {
	n := len(isolationOptions)
	switch msg.String() {
	case "up", "k":
		m.isolationCursor = moveCursor(m.isolationCursor, -1, n)
	case "down", "j":
		m.isolationCursor = moveCursor(m.isolationCursor, 1, n)
	case " ":
		m.isolation = isolationOptions[m.isolationCursor].strategy
	}
	return m, nil
}

// wizardItem is a displayable item in a multi-select list.
type wizardItem struct {
	name   string
	desc   string
	source string
}

func (m LaunchWizardModel) mcpItems() []wizardItem {
	if m.discovered == nil {
		return nil
	}
	items := make([]wizardItem, len(m.discovered.MCPs))
	for i, mcp := range m.discovered.MCPs {
		items[i] = wizardItem{
			name:   mcp.Name,
			desc:   formatMCPDesc(mcp),
			source: mcp.Source.String(),
		}
	}
	return items
}

func (m LaunchWizardModel) skillItems() []wizardItem {
	if m.discovered == nil {
		return nil
	}
	items := make([]wizardItem, len(m.discovered.Skills))
	for i, skill := range m.discovered.Skills {
		items[i] = wizardItem{
			name:   skill.Name,
			desc:   skill.Description,
			source: skill.Source.String(),
		}
	}
	return items
}

func (m LaunchWizardModel) permItems() []wizardItem {
	if m.discovered == nil {
		return nil
	}
	items := make([]wizardItem, len(m.discovered.Permissions))
	for i, perm := range m.discovered.Permissions {
		items[i] = wizardItem{
			name:   perm.Name,
			desc:   fmt.Sprintf("%s: %s", perm.Type, perm.Rule),
			source: perm.Source.String(),
		}
	}
	return items
}

func (m LaunchWizardModel) hookItems() []wizardItem {
	if m.discovered == nil {
		return nil
	}
	items := make([]wizardItem, len(m.discovered.Hooks))
	for i, hook := range m.discovered.Hooks {
		items[i] = wizardItem{
			name:   hook.Name,
			desc:   fmt.Sprintf("%s: %s", hook.Event, hook.Command),
			source: hook.Source.String(),
		}
	}
	return items
}

func formatMCPDesc(mcp discovery.DiscoveredMCP) string {
	srv := mcp.ServerDef
	switch srv.Transport {
	case model.TransportStdio:
		if len(srv.Args) > 0 {
			return fmt.Sprintf("stdio: %s %s", srv.Command, strings.Join(srv.Args, " "))
		}
		return fmt.Sprintf("stdio: %s", srv.Command)
	case model.TransportSSE, model.TransportHTTP:
		return fmt.Sprintf("%s: %s", srv.Transport, srv.URL)
	}
	return string(srv.Transport)
}

// buildProfile constructs a Profile from the wizard selections.
func (m LaunchWizardModel) buildProfile() profile.Profile {
	p := profile.Profile{
		Name:      "default",
		Isolation: m.isolation,
	}

	for name, sel := range m.mcpSelections {
		if sel {
			p.MCPs = append(p.MCPs, name)
		}
	}
	for name, sel := range m.skillSelections {
		if sel {
			p.Skills = append(p.Skills, name)
		}
	}
	for name, sel := range m.permSelections {
		if sel {
			p.Permissions = append(p.Permissions, name)
		}
	}
	for name, sel := range m.hookSelections {
		if sel {
			p.Hooks = append(p.Hooks, name)
		}
	}

	if m.claudeMDChoice > 0 && m.claudeMDChoice < len(m.claudeMDOptions) {
		p.ClaudeMD = m.claudeMDOptions[m.claudeMDChoice]
	}

	if len(m.envKeys) > 0 {
		p.EnvVars = make(map[string]string)
		for i, k := range m.envKeys {
			if k != "" && i < len(m.envValues) {
				p.EnvVars[k] = m.envValues[i]
			}
		}
	}

	return p
}

// --- Test accessors ---

// Step returns the current step.
func (m LaunchWizardModel) Step() launchStep { return m.step }

// Phase returns the current phase.
func (m LaunchWizardModel) Phase() wizardPhase { return m.phase }

// WizardMode returns the current wizard mode.
func (m LaunchWizardModel) WizardMode() LaunchWizardMode { return m.mode }

// Selection accessors.
func (m LaunchWizardModel) MCPSelections() map[string]bool        { return m.mcpSelections }
func (m LaunchWizardModel) SkillSelections() map[string]bool      { return m.skillSelections }
func (m LaunchWizardModel) PermSelections() map[string]bool       { return m.permSelections }
func (m LaunchWizardModel) HookSelections() map[string]bool       { return m.hookSelections }
func (m LaunchWizardModel) Isolation() profile.IsolationStrategy  { return m.isolation }

// --- View ---

func (m LaunchWizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.phase {
	case phaseChecklist:
		return m.renderChecklistView()
	default:
		if m.mode == LWModeHub {
			return m.renderHubView()
		}
		return m.renderSequentialView()
	}
}

// renderSequentialView renders the step-by-step wizard view.
func (m LaunchWizardModel) renderSequentialView() string {
	var b strings.Builder

	// Progress indicator
	b.WriteString(m.renderProgress())
	b.WriteString("\n\n")

	// Step title
	title := launchStepLabels[m.step]
	b.WriteString(formTitleStyle.Render(fmt.Sprintf("Step %d/%d: %s", m.step+1, int(launchStepCount), title)))
	b.WriteString("\n\n")

	// Step content
	m.renderStepContent(&b)

	// Navigation hints
	b.WriteString("\n")
	if m.step == launchStepCount-1 {
		b.WriteString(formHintStyle.Render("space: select | enter: review | esc: back | tab: skip"))
	} else {
		b.WriteString(formHintStyle.Render("space: toggle | enter: next | esc: back | tab: skip | a: toggle all"))
	}

	formWidth := clamp(m.width-4, 40, 80)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// renderHubView renders the hub mode with category menu and editing pane.
func (m LaunchWizardModel) renderHubView() string {
	var b strings.Builder

	b.WriteString(formTitleStyle.Render("Configure Profile"))
	b.WriteString("\n\n")

	// Left: category menu with selection counts
	var left strings.Builder
	for i := 0; i < int(launchStepCount); i++ {
		cur := "  "
		if launchStep(i) == m.step {
			cur = "\u25b8 "
		}
		label := launchStepLabels[i]
		summary := m.selectionSummary(launchStep(i))
		line := fmt.Sprintf("%s%-14s %s", cur, label, summary)
		if launchStep(i) == m.step {
			left.WriteString(sectionActiveStyle.Render(line))
		} else {
			left.WriteString(line)
		}
		left.WriteString("\n")
	}

	// Right: editing pane for current step
	var right strings.Builder
	right.WriteString(detailTitleStyle.Render(launchStepLabels[m.step]))
	right.WriteString("\n\n")
	m.renderStepContent(&right)

	leftWidth := 30
	rightWidth := clamp(m.width-leftWidth-10, 20, 60)

	leftRendered := lipgloss.NewStyle().Width(leftWidth).Render(left.String())
	rightRendered := detailPaneStyle.Width(rightWidth).Render(right.String())

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftRendered, rightRendered))
	b.WriteString("\n")
	b.WriteString(formHintStyle.Render("tab/shift+tab: category | space: toggle | enter: review | esc: cancel"))

	formWidth := clamp(m.width-4, 40, 100)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// renderChecklistView renders the review & confirm checklist.
func (m LaunchWizardModel) renderChecklistView() string {
	var b strings.Builder

	b.WriteString(formTitleStyle.Render("Review & Launch"))
	b.WriteString("\n\n")

	sections := []struct {
		label string
		items []string
	}{
		{"MCPs", m.selectedNames(m.mcpSelections)},
		{"Skills", m.selectedNames(m.skillSelections)},
		{"Permissions", m.selectedNames(m.permSelections)},
		{"Hooks", m.selectedNames(m.hookSelections)},
	}

	for _, sec := range sections {
		b.WriteString(sectionHeaderStyle.Render(sec.label))
		b.WriteString("\n")
		if len(sec.items) == 0 {
			b.WriteString(formHintStyle.Render("  (none)"))
			b.WriteString("\n")
		} else {
			for _, name := range sec.items {
				b.WriteString(fmt.Sprintf("  \u2022 %s\n", name))
			}
		}
	}

	// CLAUDE.md
	b.WriteString(sectionHeaderStyle.Render("CLAUDE.md"))
	b.WriteString("\n")
	claudeMD := "(none)"
	if m.claudeMDChoice > 0 && m.claudeMDChoice < len(m.claudeMDOptions) {
		claudeMD = m.claudeMDOptions[m.claudeMDChoice]
	}
	b.WriteString(fmt.Sprintf("  %s\n", claudeMD))

	// Env Vars
	b.WriteString(sectionHeaderStyle.Render("Environment Variables"))
	b.WriteString("\n")
	envCount := 0
	for i, k := range m.envKeys {
		if k != "" && i < len(m.envValues) {
			b.WriteString(fmt.Sprintf("  %s=%s\n", k, m.envValues[i]))
			envCount++
		}
	}
	if envCount == 0 {
		b.WriteString(formHintStyle.Render("  (none)"))
		b.WriteString("\n")
	}

	// Isolation
	b.WriteString(sectionHeaderStyle.Render("Isolation"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n", string(m.isolation)))

	b.WriteString("\n")
	b.WriteString(formHintStyle.Render("enter: launch | e: edit | esc: cancel"))

	formWidth := clamp(m.width-4, 40, 80)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// renderStepContent renders the content for the current step.
func (m LaunchWizardModel) renderStepContent(b *strings.Builder) {
	switch m.step {
	case launchStepMCPs:
		m.renderMultiSelect(b, m.mcpItems(), m.mcpSelections, "No MCP servers discovered. Add servers in the management TUI.")
	case launchStepSkills:
		m.renderMultiSelect(b, m.skillItems(), m.skillSelections, "No skills discovered. Add skills in the management TUI.")
	case launchStepPermissions:
		m.renderMultiSelect(b, m.permItems(), m.permSelections, "No permissions discovered.")
	case launchStepHooks:
		m.renderMultiSelect(b, m.hookItems(), m.hookSelections, "No hooks discovered.")
	case launchStepClaudeMD:
		m.renderClaudeMD(b)
	case launchStepEnvVars:
		m.renderEnvVars(b)
	case launchStepIsolation:
		m.renderIsolation(b)
	}
}

func (m LaunchWizardModel) renderProgress() string {
	var parts []string
	for i := 0; i < int(launchStepCount); i++ {
		label := launchStepLabels[i]
		if launchStep(i) == m.step {
			parts = append(parts, sectionActiveStyle.Render("["+label+"]"))
		} else if launchStep(i) < m.step {
			parts = append(parts, syncMsgStyle.Render(label))
		} else {
			parts = append(parts, sectionDimStyle.Render(label))
		}
	}
	return strings.Join(parts, " > ")
}

func (m LaunchWizardModel) renderMultiSelect(b *strings.Builder, items []wizardItem, selections map[string]bool, emptyMsg string) {
	if len(items) == 0 {
		b.WriteString(formHintStyle.Render(emptyMsg))
		return
	}

	cursor := m.cursors[m.step]
	maxVisible := clamp(m.height-14, 5, 30)
	start, end := visibleRange(cursor, len(items), maxVisible)

	if start > 0 {
		b.WriteString(formHintStyle.Render(fmt.Sprintf("  ... %d more above", start)))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		item := items[i]
		cur := "  "
		if i == cursor {
			cur = "\u25b8 "
		}
		check := "[ ]"
		if selections[item.name] {
			check = "[x]"
		}
		sourceTag := ""
		if item.source != "" {
			sourceTag = " " + formHintStyle.Render("("+item.source+")")
		}
		b.WriteString(fmt.Sprintf("%s%s %s%s\n", cur, check, item.name, sourceTag))
		if item.desc != "" {
			b.WriteString(fmt.Sprintf("       %s\n", formHintStyle.Render(item.desc)))
		}
	}

	if end < len(items) {
		b.WriteString(formHintStyle.Render(fmt.Sprintf("  ... %d more below", len(items)-end)))
		b.WriteString("\n")
	}

	selected := countSelected(selections)
	b.WriteString(fmt.Sprintf("\n%s", formLabelStyle.Render(fmt.Sprintf("%d/%d selected", selected, len(items)))))
}

func (m LaunchWizardModel) renderClaudeMD(b *strings.Builder) {
	b.WriteString("Select a CLAUDE.md template:\n\n")
	for i, opt := range m.claudeMDOptions {
		cur := "  "
		if i == m.claudeMDCursor {
			cur = "\u25b8 "
		}
		radio := "( )"
		if i == m.claudeMDChoice {
			radio = "(*)"
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cur, radio, opt))
	}
}

func (m LaunchWizardModel) renderEnvVars(b *strings.Builder) {
	b.WriteString("Environment variables:\n\n")

	if len(m.envKeys) == 0 {
		b.WriteString(formHintStyle.Render("No environment variables configured."))
		b.WriteString("\n")
	} else {
		for i, k := range m.envKeys {
			cur := "  "
			if i == m.envCursor {
				cur = "\u25b8 "
			}
			v := ""
			if i < len(m.envValues) {
				v = m.envValues[i]
			}
			b.WriteString(fmt.Sprintf("%s%s = %s\n", cur, formLabelStyle.Render(k), v))
		}
	}

	b.WriteString(fmt.Sprintf("\n%s", formHintStyle.Render("ctrl+a: add | ctrl+d: delete")))
}

func (m LaunchWizardModel) renderIsolation(b *strings.Builder) {
	b.WriteString("Select isolation strategy:\n\n")
	for i, opt := range isolationOptions {
		cur := "  "
		if i == m.isolationCursor {
			cur = "\u25b8 "
		}
		radio := "( )"
		if opt.strategy == m.isolation {
			radio = "(*)"
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cur, radio, detailTitleStyle.Render(opt.label)))
		b.WriteString(fmt.Sprintf("       %s\n", formHintStyle.Render(opt.desc)))
	}
}

// --- Helpers ---

// selectedNames returns sorted names of selected items from a selection map.
func (m LaunchWizardModel) selectedNames(selections map[string]bool) []string {
	var result []string
	for name, sel := range selections {
		if sel {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

// selectionSummary returns a short summary for a step (used in hub menu).
func (m LaunchWizardModel) selectionSummary(step launchStep) string {
	switch step {
	case launchStepMCPs:
		return fmt.Sprintf("(%d)", countSelected(m.mcpSelections))
	case launchStepSkills:
		return fmt.Sprintf("(%d)", countSelected(m.skillSelections))
	case launchStepPermissions:
		return fmt.Sprintf("(%d)", countSelected(m.permSelections))
	case launchStepHooks:
		return fmt.Sprintf("(%d)", countSelected(m.hookSelections))
	case launchStepClaudeMD:
		if m.claudeMDChoice > 0 && m.claudeMDChoice < len(m.claudeMDOptions) {
			return m.claudeMDOptions[m.claudeMDChoice]
		}
		return "(none)"
	case launchStepEnvVars:
		return fmt.Sprintf("(%d)", len(m.envKeys))
	case launchStepIsolation:
		return string(m.isolation)
	}
	return ""
}

// countSelected returns the number of true values in a selection map.
func countSelected(selections map[string]bool) int {
	n := 0
	for _, sel := range selections {
		if sel {
			n++
		}
	}
	return n
}

// visibleRange calculates the visible window for a scrollable list.
func visibleRange(cursor, total, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}
	half := maxVisible / 2
	start = cursor - half
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}
