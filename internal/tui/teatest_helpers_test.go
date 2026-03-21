package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/lcrostarosa/hystak/internal/discovery"
	"github.com/lcrostarosa/hystak/internal/model"
	"github.com/lcrostarosa/hystak/internal/profile"
	"github.com/lcrostarosa/hystak/internal/service"
)

const (
	testTermWidth  = 80
	testTermHeight = 24
	testTimeout    = 3 * time.Second
)

// newTeatestApp creates a teatest TestModel wrapping a full AppModel.
func newTeatestApp(t *testing.T, svc *service.Service) *teatest.TestModel {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	app := NewApp(svc)
	return teatest.NewTestModel(t, app, teatest.WithInitialTermSize(testTermWidth, testTermHeight))
}

// wizardTestModel wraps LaunchWizardModel to satisfy tea.Model.
// LaunchWizardModel.Update returns (LaunchWizardModel, tea.Cmd) not (tea.Model, tea.Cmd).
type wizardTestModel struct {
	inner LaunchWizardModel
}

func (w wizardTestModel) Init() tea.Cmd { return nil }

func (w wizardTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	w.inner, cmd = w.inner.Update(msg)
	return w, cmd
}

func (w wizardTestModel) View() string { return w.inner.View() }

// newTeatestWizard creates a teatest TestModel wrapping a LaunchWizardModel.
func newTeatestWizard(t *testing.T, proj *model.Project, mode LaunchWizardMode, items *discovery.Items, existing *profile.Profile) *teatest.TestModel {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	wiz := NewLaunchWizardModel(proj, mode, items, existing)
	wrapped := wizardTestModel{inner: wiz}
	return teatest.NewTestModel(t, wrapped, teatest.WithInitialTermSize(testTermWidth, 30))
}

// formTestModel wraps FormModel to satisfy tea.Model.
type formTestModel struct {
	inner FormModel
}

func (f formTestModel) Init() tea.Cmd { return nil }

func (f formTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	f.inner, cmd = f.inner.Update(msg)
	return f, cmd
}

func (f formTestModel) View() string { return f.inner.View() }

// newTeatestForm creates a teatest TestModel wrapping a FormModel.
func newTeatestForm(t *testing.T) *teatest.TestModel {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	form := NewFormModel()
	wrapped := formTestModel{inner: form}
	return teatest.NewTestModel(t, wrapped, teatest.WithInitialTermSize(testTermWidth, testTermHeight))
}

// newTeatestEditForm creates a teatest TestModel wrapping a FormModel pre-populated for editing.
func newTeatestEditForm(t *testing.T, srv model.ServerDef) *teatest.TestModel {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	form := NewEditFormModel(srv)
	wrapped := formTestModel{inner: form}
	return teatest.NewTestModel(t, wrapped, teatest.WithInitialTermSize(testTermWidth, testTermHeight))
}

// quitAndWait sends a quit message and waits for the program to finish.
func quitAndWait(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	tm.Send(tea.QuitMsg{})
	tm.WaitFinished(t, teatest.WithFinalTimeout(testTimeout))
}
