package tui

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestTeatest_LaunchWizardMCPs(t *testing.T) {
	tm := newTeatestWizard(t, testProject(), LWModeSequential, testDiscoveredItems(), nil)

	// Wait for wizard to render with MCP step content.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "MCPs") && strings.Contains(s, "mcp-a")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_LaunchWizardToggleAndAdvance(t *testing.T) {
	tm := newTeatestWizard(t, testProject(), LWModeSequential, testDiscoveredItems(), nil)

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "mcp-a")
	}, teatest.WithDuration(2*time.Second))

	// Toggle mcp-a with space.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	// Advance to Skills step with Enter.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for skills step.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Skills")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_LaunchWizardChecklist(t *testing.T) {
	tm := newTeatestWizard(t, testProject(), LWModeSequential, testDiscoveredItems(), nil)

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "mcp-a")
	}, teatest.WithDuration(2*time.Second))

	// Toggle mcp-a.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	// Advance through all steps to checklist (7 steps: MCPs, Skills, Permissions, Hooks, CLAUDE.md, Prompts, EnvVars, Isolation).
	for i := 0; i < 8; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		time.Sleep(50 * time.Millisecond) // small delay for message processing
	}

	// Wait for checklist phase.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "launch") || strings.Contains(s, "mcp-a")
	}, teatest.WithDuration(3*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_LaunchWizardHubMode(t *testing.T) {
	tm := newTeatestWizard(t, testProject(), LWModeHub, testDiscoveredItems(), nil)

	// Wait for hub mode to render with category menu.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "MCPs") && strings.Contains(s, "Skills") && strings.Contains(s, "mcp-a")
	}, teatest.WithDuration(2*time.Second))

	// Toggle some items.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	// Switch category with Tab.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "skill-1")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}
