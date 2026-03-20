package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/service"
)

// Tab identifies which tab is active.
type Tab int

const (
	ProfilesTab Tab = iota
	MCPsTab
	SkillsTab
	HooksTab
	PermissionsTab
	TemplatesTab
	tabCount
)

// Mode represents the current UI mode.
type Mode int

const (
	ModeBrowse Mode = iota
	ModeForm
	ModeConfirm
	ModeDiff
	ModeImport
	ModeSkillForm
	ModeHookForm
	ModePermissionForm
	ModeTemplateForm
	ModeConflictResolve
	ModeDiscovery
	ModeLaunchWizard
)

var tabLabels = []string{"Profiles", "MCPs", "Skills", "Hooks", "Permissions", "Templates"}

// overlay is the interface for full-screen overlay models.
type overlay interface {
	SetSize(w, h int)
	View() string
}

// AppModel is the root Bubble Tea model for the hystak TUI.
type AppModel struct {
	service   *service.Service
	activeTab Tab
	mode      Mode
	keys      KeyMap
	width     int
	height    int
	err       error

	mcps        MCPsModel
	profiles    ProfilesModel
	skills      SkillsModel
	hooks       HooksModel
	permissions PermissionsModel
	templates   TemplatesModel
	form        FormModel
	importer    ImportModel
	diff        DiffModel

	skillForm      SkillFormModel
	hookForm       HookFormModel
	permissionForm PermissionFormModel
	templateForm   TemplateFormModel
	conflict       ConflictModel
	discovery      DiscoveryModel
	launchWizard   LaunchWizardModel

	launchProfile *model.Project
}

// LaunchRequest returns the project to launch after the TUI exits, or nil.
func (m AppModel) LaunchRequest() *model.Project { return m.launchProfile }

// NewApp creates a new TUI application model.
func NewApp(svc *service.Service) AppModel {
	return AppModel{
		service:     svc,
		activeTab:   ProfilesTab,
		mode:        ModeBrowse,
		keys:        newKeyMap(),
		mcps:        NewMCPsModel(svc),
		profiles:    NewProfilesModel(svc),
		skills:      NewSkillsModel(svc),
		hooks:       NewHooksModel(svc),
		permissions: NewPermissionsModel(svc),
		templates:   NewTemplatesModel(svc),
	}
}

// activeOverlay returns the current overlay model, or nil in browse/confirm modes.
func (m *AppModel) activeOverlay() overlay {
	switch m.mode {
	case ModeForm:
		return &m.form
	case ModeImport:
		return &m.importer
	case ModeDiff:
		return &m.diff
	case ModeSkillForm:
		return &m.skillForm
	case ModeHookForm:
		return &m.hookForm
	case ModePermissionForm:
		return &m.permissionForm
	case ModeTemplateForm:
		return &m.templateForm
	case ModeConflictResolve:
		return &m.conflict
	case ModeDiscovery:
		return &m.discovery
	case ModeLaunchWizard:
		return &m.launchWizard
	}
	return nil
}

// Init implements tea.Model.
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateContentSize()
		if ov := m.activeOverlay(); ov != nil {
			ov.SetSize(m.width, m.height)
		}
		return m, nil

	case RequestFormMsg:
		if msg.EditServer != nil {
			m.form = NewEditFormModel(*msg.EditServer)
		} else {
			m.form = NewFormModel()
		}
		m.form.SetSize(m.width, m.height)
		m.mode = ModeForm
		return m, nil

	case FormSubmittedMsg:
		m.mode = ModeBrowse
		m.err = nil
		var err error
		if msg.IsEdit {
			err = m.service.UpdateServer(m.form.editName, msg.Server)
		} else {
			err = m.service.AddServer(msg.Server)
		}
		if err != nil {
			m.err = err
		}
		m.mcps.refreshList()
		return m, nil

	case FormCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestImportMsg:
		m.importer = NewImportModel(m.service)
		m.importer.SetSize(m.width, m.height)
		m.mode = ModeImport
		return m, nil

	case ImportCompletedMsg:
		m.mode = ModeBrowse
		m.mcps.refreshList()
		return m, nil

	case ImportCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestDiffMsg:
		m.diff = NewDiffModel(m.service, msg.ProjectName)
		m.diff.SetSize(m.width, m.height)
		m.mode = ModeDiff
		return m, nil

	case DiffClosedMsg:
		m.mode = ModeBrowse
		m.profiles.refreshList()
		return m, nil

	case AutoSyncResultMsg:
		var cmd tea.Cmd
		m.profiles, cmd = m.profiles.Update(msg)
		return m, cmd

	case RequestLaunchMsg:
		if proj, ok := m.service.GetProject(msg.ProfileName); ok {
			m.launchProfile = &proj
		}
		return m, tea.Quit

	case RequestSkillFormMsg:
		if msg.EditSkill != nil {
			m.skillForm = NewEditSkillFormModel(*msg.EditSkill)
		} else {
			m.skillForm = NewSkillFormModel()
		}
		m.skillForm.SetSize(m.width, m.height)
		m.mode = ModeSkillForm
		return m, nil

	case SkillFormSubmittedMsg:
		m.mode = ModeBrowse
		m.err = nil
		var err error
		if msg.IsEdit {
			err = m.service.UpdateSkill(m.skillForm.editName, msg.Skill)
		} else {
			err = m.service.AddSkill(msg.Skill)
		}
		if err != nil {
			m.err = err
		}
		m.skills.refreshList()
		return m, nil

	case SkillFormCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestHookFormMsg:
		if msg.EditHook != nil {
			m.hookForm = NewEditHookFormModel(*msg.EditHook)
		} else {
			m.hookForm = NewHookFormModel()
		}
		m.hookForm.SetSize(m.width, m.height)
		m.mode = ModeHookForm
		return m, nil

	case HookFormSubmittedMsg:
		m.mode = ModeBrowse
		m.err = nil
		var err error
		if msg.IsEdit {
			err = m.service.UpdateHook(m.hookForm.editName, msg.Hook)
		} else {
			err = m.service.AddHook(msg.Hook)
		}
		if err != nil {
			m.err = err
		}
		m.hooks.refreshList()
		return m, nil

	case HookFormCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestPermissionFormMsg:
		if msg.EditPermission != nil {
			m.permissionForm = NewEditPermissionFormModel(*msg.EditPermission)
		} else {
			m.permissionForm = NewPermissionFormModel()
		}
		m.permissionForm.SetSize(m.width, m.height)
		m.mode = ModePermissionForm
		return m, nil

	case PermissionFormSubmittedMsg:
		m.mode = ModeBrowse
		m.err = nil
		var err error
		if msg.IsEdit {
			err = m.service.UpdatePermission(m.permissionForm.editName, msg.Permission)
		} else {
			err = m.service.AddPermission(msg.Permission)
		}
		if err != nil {
			m.err = err
		}
		m.permissions.refreshList()
		return m, nil

	case PermissionFormCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestTemplateFormMsg:
		if msg.EditTemplate != nil {
			m.templateForm = NewEditTemplateFormModel(*msg.EditTemplate)
		} else {
			m.templateForm = NewTemplateFormModel()
		}
		m.templateForm.SetSize(m.width, m.height)
		m.mode = ModeTemplateForm
		return m, nil

	case TemplateFormSubmittedMsg:
		m.mode = ModeBrowse
		m.err = nil
		var err error
		if msg.IsEdit {
			err = m.service.UpdateTemplate(m.templateForm.editName, msg.Template)
		} else {
			err = m.service.AddTemplate(msg.Template)
		}
		if err != nil {
			m.err = err
		}
		m.templates.refreshList()
		return m, nil

	case TemplateFormCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestConflictResolveMsg:
		m.conflict = NewConflictModel(msg.ProjectName, msg.Conflicts)
		m.conflict.SetSize(m.width, m.height)
		m.mode = ModeConflictResolve
		return m, nil

	case ConflictResolvedMsg:
		m.mode = ModeBrowse
		// After resolution, kick off the actual sync.
		svc := m.service
		projName := msg.ProjectName
		return m, func() tea.Msg {
			results, err := svc.SyncProject(projName)
			return AutoSyncResultMsg{ProjectName: projName, Err: err, Results: results}
		}

	case ConflictCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestDiscoveryMsg:
		m.discovery = NewDiscoveryModel(m.service, msg.ProjectName, msg.ProjectPath)
		m.discovery.SetSize(m.width, m.height)
		m.mode = ModeDiscovery
		return m, nil

	case DiscoveryCompletedMsg:
		m.mode = ModeBrowse
		m.skills.refreshList()
		m.profiles.refreshList()
		return m, nil

	case DiscoveryCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case RequestLaunchWizardMsg:
		m.launchWizard = NewLaunchWizardModel(msg.Project, msg.Mode, msg.Discovered, nil)
		m.launchWizard.SetSize(m.width, m.height)
		m.mode = ModeLaunchWizard
		return m, nil

	case LaunchWizardCompleteMsg:
		m.mode = ModeBrowse
		m.profiles.refreshList()
		return m, nil

	case LaunchWizardCancelledMsg:
		m.mode = ModeBrowse
		return m, nil

	case tea.KeyMsg:
		// In form mode, route all input to the form.
		if m.mode == ModeForm {
			var cmd tea.Cmd
			m.form, cmd = m.form.Update(msg)
			return m, cmd
		}

		// In import mode, route all input to the importer.
		if m.mode == ModeImport {
			var cmd tea.Cmd
			m.importer, cmd = m.importer.Update(msg)
			return m, cmd
		}

		// In diff mode, route all input to the diff model.
		if m.mode == ModeDiff {
			var cmd tea.Cmd
			m.diff, cmd = m.diff.Update(msg)
			return m, cmd
		}

		// In skill form mode, route all input to the skill form.
		if m.mode == ModeSkillForm {
			var cmd tea.Cmd
			m.skillForm, cmd = m.skillForm.Update(msg)
			return m, cmd
		}

		// In hook form mode, route all input to the hook form.
		if m.mode == ModeHookForm {
			var cmd tea.Cmd
			m.hookForm, cmd = m.hookForm.Update(msg)
			return m, cmd
		}

		// In permission form mode, route all input to the permission form.
		if m.mode == ModePermissionForm {
			var cmd tea.Cmd
			m.permissionForm, cmd = m.permissionForm.Update(msg)
			return m, cmd
		}

		// In template form mode, route all input to the template form.
		if m.mode == ModeTemplateForm {
			var cmd tea.Cmd
			m.templateForm, cmd = m.templateForm.Update(msg)
			return m, cmd
		}

		// In conflict resolve mode, route all input to the conflict model.
		if m.mode == ModeConflictResolve {
			var cmd tea.Cmd
			m.conflict, cmd = m.conflict.Update(msg)
			return m, cmd
		}

		// In discovery mode, route all input to the discovery model.
		if m.mode == ModeDiscovery {
			var cmd tea.Cmd
			m.discovery, cmd = m.discovery.Update(msg)
			return m, cmd
		}

		// In launch wizard mode, route all input to the launch wizard.
		if m.mode == ModeLaunchWizard {
			var cmd tea.Cmd
			m.launchWizard, cmd = m.launchWizard.Update(msg)
			return m, cmd
		}

		// In other overlay modes, handle escape to return to browse.
		if m.mode != ModeBrowse {
			if msg.String() == "esc" {
				m.mode = ModeBrowse
			}
			return m, nil
		}

		// If the active tab is consuming input (filtering, confirming),
		// skip global key handling and route directly to the tab.
		if !m.activeTabConsuming() {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit

			case key.Matches(msg, m.keys.TabNext):
				m.activeTab = (m.activeTab + 1) % tabCount
				return m, nil

			case key.Matches(msg, m.keys.TabPrev):
				m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
				return m, nil
			}
		}
	}

	// Route remaining messages to the active tab.
	return m.updateActiveTab(msg)
}

// activeTabConsuming returns true if the active tab is handling its own input.
func (m AppModel) activeTabConsuming() bool {
	switch m.activeTab {
	case ProfilesTab:
		return m.profiles.IsConsuming()
	case MCPsTab:
		return m.mcps.IsConsuming()
	case SkillsTab:
		return m.skills.IsConsuming()
	case HooksTab:
		return m.hooks.IsConsuming()
	case PermissionsTab:
		return m.permissions.IsConsuming()
	case TemplatesTab:
		return m.templates.IsConsuming()
	}
	return false
}

// updateActiveTab forwards a message to the currently active tab.
func (m AppModel) updateActiveTab(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case ProfilesTab:
		var cmd tea.Cmd
		m.profiles, cmd = m.profiles.Update(msg)
		return m, cmd
	case MCPsTab:
		var cmd tea.Cmd
		m.mcps, cmd = m.mcps.Update(msg)
		return m, cmd
	case SkillsTab:
		var cmd tea.Cmd
		m.skills, cmd = m.skills.Update(msg)
		return m, cmd
	case HooksTab:
		var cmd tea.Cmd
		m.hooks, cmd = m.hooks.Update(msg)
		return m, cmd
	case PermissionsTab:
		var cmd tea.Cmd
		m.permissions, cmd = m.permissions.Update(msg)
		return m, cmd
	case TemplatesTab:
		var cmd tea.Cmd
		m.templates, cmd = m.templates.Update(msg)
		return m, cmd
	}
	return m, nil
}

// updateContentSize recalculates and propagates dimensions to sub-models.
func (m *AppModel) updateContentSize() {
	contentHeight := m.contentHeight()
	m.mcps.SetSize(m.width, contentHeight)
	m.profiles.SetSize(m.width, contentHeight)
	m.skills.SetSize(m.width, contentHeight)
	m.hooks.SetSize(m.width, contentHeight)
	m.permissions.SetSize(m.width, contentHeight)
	m.templates.SetSize(m.width, contentHeight)
}

// contentHeight returns the height available for tab content.
func (m AppModel) contentHeight() int {
	// Tab bar is 1 line + border = 2 lines, status bar is 1 line.
	// contentStyle has padding(1,2) = 2 vertical lines.
	overhead := 2 + 1 + 2
	return max(0, m.height-overhead)
}

// View implements tea.Model.
func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Overlay modes take over the full screen.
	if ov := m.activeOverlay(); ov != nil {
		return ov.View()
	}

	tabBar := m.renderTabBar()

	var content string
	switch m.activeTab {
	case ProfilesTab:
		content = m.profiles.View()
	case MCPsTab:
		content = m.mcps.View()
	case SkillsTab:
		content = m.skills.View()
	case HooksTab:
		content = m.hooks.View()
	case PermissionsTab:
		content = m.permissions.View()
	case TemplatesTab:
		content = m.templates.View()
	}

	statusBar := m.renderStatusBar()

	// Calculate content height: total - tab bar - status bar - padding
	tabBarHeight := lipgloss.Height(tabBar)
	statusBarHeight := lipgloss.Height(statusBar)
	contentHeight := max(0, m.height-tabBarHeight-statusBarHeight)

	styledContent := contentStyle.
		Width(m.width).
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, styledContent, statusBar)
}

func (m AppModel) renderTabBar() string {
	var tabs []string
	for i, label := range tabLabels {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Fill remaining width with gap
	gap := tabGapStyle.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(row))))
	return lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
}

func (m AppModel) renderStatusBar() string {
	var help string
	switch m.activeTab {
	case ProfilesTab:
		help = m.profiles.StatusHelp()
	case MCPsTab:
		help = m.mcps.StatusHelp()
	case SkillsTab:
		help = m.skills.StatusHelp()
	case HooksTab:
		help = m.hooks.StatusHelp()
	case PermissionsTab:
		help = m.permissions.StatusHelp()
	case TemplatesTab:
		help = m.templates.StatusHelp()
	default:
		help = "tab: switch tabs | q: quit"
	}
	return statusBarStyle.Render(help)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
