package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/catalog"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// wizardStep tracks the current step in the setup wizard.
type wizardStep int

const (
	wizardWelcome wizardStep = iota
	wizardScanResult
	wizardCatalog
	wizardProjectForm
	wizardSummary
	wizardDone
)

// catalogSection identifies which entity type is shown in the catalog browser.
type catalogSection int

const (
	catMCPs catalogSection = iota
	catSkills
	catHooks
	catPermissions
	catSectionCount
)

var catSectionLabels = []string{"MCPs", "Skills", "Hooks", "Permissions"}

// scanCompleteMsg is sent when the config scan finishes.
type scanCompleteMsg struct {
	results []service.ConfigScanResult
}

// WizardModel is the setup wizard for first-time and on-demand configuration.
type WizardModel struct {
	service *service.Service
	step    wizardStep
	width   int
	height  int
	err     string

	// Scan results (import step)
	scanResults   []service.ConfigScanResult
	allCandidates []service.ImportCandidate
	scanSelected  []bool
	scanCursor    int

	// Catalog (browse step)
	cat            catalog.Catalog
	catSection     catalogSection
	catCursors     [catSectionCount]int
	catSelectedMCP []bool
	catSelectedSk  []bool
	catSelectedHk  []bool
	catSelectedPm  []bool

	// Project form
	nameInput    textinput.Model
	pathInput    textinput.Model
	focused      int  // 0 = name, 1 = path
	nameModified bool // true once user manually edits the name

	// Outcome
	completed     bool
	importedCount int
	catalogCount  int
	projectName   string
}

// Completed returns true if the wizard finished setup (vs. being skipped).
func (m WizardModel) Completed() bool { return m.completed }

// NewWizardModel creates a new setup wizard.
func NewWizardModel(svc *service.Service) WizardModel {
	ni := textinput.New()
	ni.Placeholder = "my-project"
	ni.Prompt = "  "
	ni.CharLimit = 128

	pi := textinput.New()
	pi.Placeholder = "/path/to/project"
	pi.Prompt = "  "
	pi.CharLimit = 512
	if cwd, err := os.Getwd(); err == nil {
		pi.SetValue(cwd)
		ni.SetValue(filepath.Base(cwd))
	}

	cat := catalog.Load()

	return WizardModel{
		service:        svc,
		step:           wizardWelcome,
		nameInput:      ni,
		pathInput:      pi,
		cat:            cat,
		catSelectedMCP: make([]bool, len(cat.MCPs)),
		catSelectedSk:  make([]bool, len(cat.Skills)),
		catSelectedHk:  make([]bool, len(cat.Hooks)),
		catSelectedPm:  make([]bool, len(cat.Permissions)),
	}
}

func (m WizardModel) Init() tea.Cmd {
	return nil
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputWidth := clamp(m.width-10, 30, 70)
		m.nameInput.Width = inputWidth
		m.pathInput.Width = inputWidth
		return m, nil

	case scanCompleteMsg:
		m.scanResults = msg.results
		for _, r := range msg.results {
			m.allCandidates = append(m.allCandidates, r.Candidates...)
		}
		m.scanSelected = make([]bool, len(m.allCandidates))
		for i := range m.scanSelected {
			m.scanSelected[i] = true
		}
		if len(m.allCandidates) > 0 {
			m.step = wizardScanResult
		} else {
			m.step = wizardCatalog
		}
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case wizardWelcome:
			return m.updateWelcome(msg)
		case wizardScanResult:
			return m.updateScanResult(msg)
		case wizardCatalog:
			return m.updateCatalog(msg)
		case wizardProjectForm:
			return m.updateProjectForm(msg)
		case wizardSummary:
			return m.updateSummary(msg)
		case wizardDone:
			return m, tea.Quit
		}
	}

	if m.step == wizardProjectForm {
		return m.updateTextInputs(msg)
	}

	return m, nil
}

// --- Step handlers ---

func (m WizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		svc := m.service
		return m, func() tea.Msg {
			results := svc.ScanForConfigs()
			return scanCompleteMsg{results: results}
		}
	case "esc", "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m WizardModel) updateScanResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.allCandidates = nil
		m.scanSelected = nil
		m.step = wizardCatalog
		return m, nil
	case "up", "k":
		m.scanCursor = moveCursor(m.scanCursor, -1, len(m.allCandidates))
		return m, nil
	case "down", "j":
		m.scanCursor = moveCursor(m.scanCursor, 1, len(m.allCandidates))
		return m, nil
	case " ":
		if m.scanCursor < len(m.scanSelected) {
			m.scanSelected[m.scanCursor] = !m.scanSelected[m.scanCursor]
		}
		return m, nil
	case "enter":
		m.step = wizardCatalog
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateCatalog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = wizardProjectForm
		m.nameInput.Focus()
		return m, nil
	case "tab":
		m.catSection = (m.catSection + 1) % catSectionCount
		return m, nil
	case "shift+tab":
		m.catSection = (m.catSection - 1 + catSectionCount) % catSectionCount
		return m, nil
	case "up", "k":
		n := m.catSectionLen()
		m.catCursors[m.catSection] = moveCursor(m.catCursors[m.catSection], -1, n)
		return m, nil
	case "down", "j":
		n := m.catSectionLen()
		m.catCursors[m.catSection] = moveCursor(m.catCursors[m.catSection], 1, n)
		return m, nil
	case " ":
		m.toggleCatalogItem()
		return m, nil
	case "enter":
		m.step = wizardProjectForm
		m.nameInput.Focus()
		return m, nil
	}
	return m, nil
}

func (m WizardModel) updateProjectForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = wizardCatalog
		return m, nil
	case "tab", "shift+tab":
		if m.focused == 0 {
			m.focused = 1
			m.nameInput.Blur()
			m.pathInput.Focus()
		} else {
			m.focused = 0
			m.pathInput.Blur()
			m.nameInput.Focus()
		}
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			m.err = "Profile name is required"
			return m, nil
		}
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			m.err = "Project path is required"
			return m, nil
		}
		m.err = ""
		m.projectName = name
		m.step = wizardSummary
		return m, nil
	}
	return m.updateTextInputs(msg)
}

func (m WizardModel) updateTextInputs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.focused == 0 {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		m.nameModified = true
		return m, cmd
	}
	var cmd tea.Cmd
	oldPath := m.pathInput.Value()
	m.pathInput, cmd = m.pathInput.Update(msg)
	// Auto-update name from path basename when user hasn't manually edited it.
	if !m.nameModified && m.pathInput.Value() != oldPath {
		base := filepath.Base(strings.TrimSpace(m.pathInput.Value()))
		if base != "" && base != "." && base != "/" {
			m.nameInput.SetValue(base)
		}
	}
	return m, cmd
}

func (m WizardModel) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.step = wizardProjectForm
		m.nameInput.Focus()
		return m, nil
	case "enter":
		return m.applySetup()
	}
	return m, nil
}

// --- Catalog helpers ---

func (m WizardModel) catSectionLen() int {
	switch m.catSection {
	case catMCPs:
		return len(m.cat.MCPs)
	case catSkills:
		return len(m.cat.Skills)
	case catHooks:
		return len(m.cat.Hooks)
	case catPermissions:
		return len(m.cat.Permissions)
	}
	return 0
}

func (m *WizardModel) toggleCatalogItem() {
	cur := m.catCursors[m.catSection]
	switch m.catSection {
	case catMCPs:
		if cur < len(m.catSelectedMCP) {
			m.catSelectedMCP[cur] = !m.catSelectedMCP[cur]
		}
	case catSkills:
		if cur < len(m.catSelectedSk) {
			m.catSelectedSk[cur] = !m.catSelectedSk[cur]
		}
	case catHooks:
		if cur < len(m.catSelectedHk) {
			m.catSelectedHk[cur] = !m.catSelectedHk[cur]
		}
	case catPermissions:
		if cur < len(m.catSelectedPm) {
			m.catSelectedPm[cur] = !m.catSelectedPm[cur]
		}
	}
}

func (m WizardModel) selectedCandidates() []service.ImportCandidate {
	var kept []service.ImportCandidate
	for i, c := range m.allCandidates {
		if i < len(m.scanSelected) && m.scanSelected[i] {
			kept = append(kept, c)
		}
	}
	return kept
}

func (m WizardModel) catalogSelectionSummary() (mcps []string, skills []string, hooks []string, perms []string) {
	for i, sel := range m.catSelectedMCP {
		if sel {
			mcps = append(mcps, m.cat.MCPs[i].Name)
		}
	}
	for i, sel := range m.catSelectedSk {
		if sel {
			skills = append(skills, m.cat.Skills[i].Name)
		}
	}
	for i, sel := range m.catSelectedHk {
		if sel {
			hooks = append(hooks, m.cat.Hooks[i].Name)
		}
	}
	for i, sel := range m.catSelectedPm {
		if sel {
			perms = append(perms, m.cat.Permissions[i].Name)
		}
	}
	return
}

// --- Apply ---

func (m WizardModel) applySetup() (tea.Model, tea.Cmd) {
	// 1. Import scanned servers.
	candidates := m.selectedCandidates()
	if len(candidates) > 0 {
		if err := m.service.ApplyImport(candidates); err != nil {
			m.err = fmt.Sprintf("import failed: %v", err)
			return m, nil
		}
		m.importedCount = len(candidates)
	}

	// 2. Install catalog selections.
	catMCPs, catSkills, catHooks, catPerms := m.catalogSelectionSummary()

	for i, sel := range m.catSelectedMCP {
		if sel {
			entry := m.cat.MCPs[i]
			_ = m.service.AddServer(entry.ServerDef)
		}
	}
	for i, sel := range m.catSelectedSk {
		if sel {
			entry := m.cat.Skills[i]
			_ = m.service.InstallCatalogSkill(entry.Name, entry.Description, entry.Content)
		}
	}
	for i, sel := range m.catSelectedHk {
		if sel {
			entry := m.cat.Hooks[i]
			_ = m.service.AddHook(entry.HookDef)
		}
	}
	for i, sel := range m.catSelectedPm {
		if sel {
			entry := m.cat.Permissions[i]
			_ = m.service.AddPermission(entry.PermissionRule)
		}
	}

	m.catalogCount = len(catMCPs) + len(catSkills) + len(catHooks) + len(catPerms)

	// 3. Create the project.
	projPath := strings.TrimSpace(m.pathInput.Value())
	proj := model.Project{
		Name:    m.projectName,
		Path:    projPath,
		Clients: []model.ClientType{model.ClientClaudeCode},
	}
	if err := m.service.AddProject(proj); err != nil {
		m.err = fmt.Sprintf("creating profile: %v", err)
		return m, nil
	}

	// 4. Assign all servers (imported + catalog) to the project.
	for _, c := range candidates {
		name := c.Name
		if c.Resolution == service.ImportRename {
			name = c.RenameTo
		}
		_ = m.service.AssignServer(m.projectName, name)
	}
	for _, name := range catMCPs {
		_ = m.service.AssignServer(m.projectName, name)
	}
	for _, name := range catSkills {
		_ = m.service.AssignSkill(m.projectName, name)
	}
	for _, name := range catHooks {
		_ = m.service.AssignHook(m.projectName, name)
	}
	for _, name := range catPerms {
		_ = m.service.AssignPermission(m.projectName, name)
	}

	m.completed = true
	m.step = wizardDone
	return m, nil
}

// --- Views ---

func (m WizardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	switch m.step {
	case wizardWelcome:
		m.renderWelcome(&b)
	case wizardScanResult:
		m.renderScanResult(&b)
	case wizardCatalog:
		m.renderCatalog(&b)
	case wizardProjectForm:
		m.renderProjectForm(&b)
	case wizardSummary:
		m.renderSummary(&b)
	case wizardDone:
		m.renderDone(&b)
	}

	formWidth := clamp(m.width-4, 40, 76)
	content := formBoxStyle.Width(formWidth).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m WizardModel) renderWelcome(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Welcome to hystak"))
	b.WriteString("\n\n")
	b.WriteString("hystak is your local shelf for Claude Code\n")
	b.WriteString("artifacts: MCPs, skills, hooks, permissions.\n")
	b.WriteString("\n")
	b.WriteString(formHintStyle.Render("Configure once, grab from the shelf when you need it."))
	b.WriteString("\n\n")
	b.WriteString("This wizard will help you:\n")
	b.WriteString("  1. Import existing MCP configs\n")
	b.WriteString("  2. Browse the built-in catalog\n")
	b.WriteString("  3. Create your first profile\n")
	b.WriteString("\n")
	b.WriteString(formHintStyle.Render("enter: begin | esc: skip setup"))
}

func (m WizardModel) renderScanResult(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Import Existing Configs"))
	b.WriteString("\n\n")

	idx := 0
	for _, r := range m.scanResults {
		count := len(r.Candidates)
		b.WriteString(formLabelStyle.Render(fmt.Sprintf("%s (%d servers)", r.Path, count)))
		b.WriteString("\n")

		for _, c := range r.Candidates {
			cur := "  "
			if idx == m.scanCursor {
				cur = "\u25b8 "
			}
			check := "[x]"
			if !m.scanSelected[idx] {
				check = "[ ]"
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cur, check, c.Name))
			detail := formatServerCompact(c.Server)
			b.WriteString(fmt.Sprintf("       %s\n", formHintStyle.Render(detail)))
			idx++
		}
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(formHintStyle.Render("space: toggle | enter: continue | esc: skip"))
}

func (m WizardModel) renderCatalog(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Browse Catalog"))
	b.WriteString("\n")

	// Section tabs.
	var tabs []string
	for i, label := range catSectionLabels {
		if catalogSection(i) == m.catSection {
			tabs = append(tabs, detailTitleStyle.Render("["+label+"]"))
		} else {
			tabs = append(tabs, formHintStyle.Render(" "+label+" "))
		}
	}
	b.WriteString(strings.Join(tabs, " "))
	b.WriteString("\n\n")

	cursor := m.catCursors[m.catSection]

	switch m.catSection {
	case catMCPs:
		for i, entry := range m.cat.MCPs {
			cur := "  "
			if i == cursor {
				cur = "\u25b8 "
			}
			check := "[ ]"
			if m.catSelectedMCP[i] {
				check = "[x]"
			}
			label := entry.Name
			if entry.Popular {
				label += " *"
			}
			b.WriteString(fmt.Sprintf("%s%s %-28s %s\n", cur, check, label,
				formHintStyle.Render(entry.Description)))
		}

	case catSkills:
		for i, entry := range m.cat.Skills {
			cur := "  "
			if i == cursor {
				cur = "\u25b8 "
			}
			check := "[ ]"
			if m.catSelectedSk[i] {
				check = "[x]"
			}
			b.WriteString(fmt.Sprintf("%s%s %-28s %s\n", cur, check, entry.Name,
				formHintStyle.Render(entry.Description)))
		}

	case catHooks:
		for i, entry := range m.cat.Hooks {
			cur := "  "
			if i == cursor {
				cur = "\u25b8 "
			}
			check := "[ ]"
			if m.catSelectedHk[i] {
				check = "[x]"
			}
			desc := fmt.Sprintf("%s: %s", entry.Event, entry.Command)
			b.WriteString(fmt.Sprintf("%s%s %-28s %s\n", cur, check, entry.Name,
				formHintStyle.Render(desc)))
		}

	case catPermissions:
		for i, entry := range m.cat.Permissions {
			cur := "  "
			if i == cursor {
				cur = "\u25b8 "
			}
			check := "[ ]"
			if m.catSelectedPm[i] {
				check = "[x]"
			}
			desc := fmt.Sprintf("%s: %s", entry.EffectiveType(), entry.Rule)
			b.WriteString(fmt.Sprintf("%s%s %-28s %s\n", cur, check, entry.Name,
				formHintStyle.Render(desc)))
		}
	}

	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(formHintStyle.Render("space: toggle | tab: section | enter: continue | esc: skip"))
}

func (m WizardModel) renderProjectForm(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Create Profile"))
	b.WriteString("\n\n")

	nameLabel := formLabelStyle.Render("Profile name")
	pathLabel := formLabelStyle.Render("Project path")

	if m.focused == 0 {
		nameLabel = detailTitleStyle.Render("Profile name")
	} else {
		pathLabel = detailTitleStyle.Render("Project path")
	}

	b.WriteString(nameLabel)
	b.WriteString("\n")
	b.WriteString(m.nameInput.View())
	b.WriteString("\n\n")

	b.WriteString(pathLabel)
	b.WriteString("\n")
	b.WriteString(m.pathInput.View())
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(formHintStyle.Render("tab: switch field | enter: continue | esc: back"))
}

func (m WizardModel) renderSummary(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Setup Summary"))
	b.WriteString("\n\n")

	// Imported servers.
	candidates := m.selectedCandidates()
	if len(candidates) > 0 {
		names := make([]string, len(candidates))
		for i, c := range candidates {
			names[i] = c.Name
		}
		b.WriteString(fmt.Sprintf("%s %d imported MCPs\n", formLabelStyle.Render("Import:"), len(candidates)))
		b.WriteString(formHintStyle.Render("  " + strings.Join(names, ", ")))
		b.WriteString("\n\n")
	}

	// Catalog selections.
	catMCPs, catSkills, catHooks, catPerms := m.catalogSelectionSummary()
	if len(catMCPs) > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", formLabelStyle.Render("Catalog MCPs:"), strings.Join(catMCPs, ", ")))
	}
	if len(catSkills) > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", formLabelStyle.Render("Catalog Skills:"), strings.Join(catSkills, ", ")))
	}
	if len(catHooks) > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", formLabelStyle.Render("Catalog Hooks:"), strings.Join(catHooks, ", ")))
	}
	if len(catPerms) > 0 {
		b.WriteString(fmt.Sprintf("%s %s\n", formLabelStyle.Render("Catalog Perms:"), strings.Join(catPerms, ", ")))
	}
	totalCatalog := len(catMCPs) + len(catSkills) + len(catHooks) + len(catPerms)
	if totalCatalog > 0 || len(candidates) > 0 {
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("%s %s\n", formLabelStyle.Render("Profile:"), detailTitleStyle.Render(m.projectName)))
	b.WriteString(fmt.Sprintf("  %s %s\n", formLabelStyle.Render("Path:"), m.pathInput.Value()))
	b.WriteString(fmt.Sprintf("  %s claude-code\n", formLabelStyle.Render("Client:")))

	totalMCPs := len(candidates) + len(catMCPs)
	if totalMCPs > 0 {
		b.WriteString(fmt.Sprintf("  %s %d servers\n", formLabelStyle.Render("MCPs:"), totalMCPs))
	}
	if len(catSkills) > 0 {
		b.WriteString(fmt.Sprintf("  %s %d\n", formLabelStyle.Render("Skills:"), len(catSkills)))
	}
	if len(catHooks) > 0 {
		b.WriteString(fmt.Sprintf("  %s %d\n", formLabelStyle.Render("Hooks:"), len(catHooks)))
	}
	if len(catPerms) > 0 {
		b.WriteString(fmt.Sprintf("  %s %d\n", formLabelStyle.Render("Permissions:"), len(catPerms)))
	}

	b.WriteString("\n")
	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(formHintStyle.Render("enter: confirm | esc: back"))
}

func (m WizardModel) renderDone(b *strings.Builder) {
	b.WriteString(formTitleStyle.Render("Setup Complete"))
	b.WriteString("\n\n")
	if m.importedCount > 0 {
		b.WriteString(fmt.Sprintf("Imported %d MCPs from existing configs.\n", m.importedCount))
	}
	if m.catalogCount > 0 {
		b.WriteString(fmt.Sprintf("Installed %d items from catalog.\n", m.catalogCount))
	}
	b.WriteString(fmt.Sprintf("Created profile %s.\n", detailTitleStyle.Render(m.projectName)))
	b.WriteString("\n")
	b.WriteString(syncMsgStyle.Render("Your shelf is stocked. Run hystak to pick a profile and launch."))
	b.WriteString("\n\n")
	b.WriteString(formHintStyle.Render("Press any key to continue..."))
}
