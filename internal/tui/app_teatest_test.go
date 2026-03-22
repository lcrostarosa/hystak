package tui

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestTeatest_AppStartup(t *testing.T) {
	tm := newTeatestApp(t, testService())

	// Wait for initial render — Profiles tab shows project details.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "myproject")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_TabNavigation(t *testing.T) {
	tm := newTeatestApp(t, testService())

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "myproject")
	}, teatest.WithDuration(2*time.Second))

	// Press right to switch to Tools, then right again to MCPs.
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "github") && strings.Contains(s, "Transport:")
	}, teatest.WithDuration(2*time.Second))

	// Press right again to switch to Skills.
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})

	// Skills tab should show "No skill selected" or similar.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "No skill") || strings.Contains(s, "skill")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_MCPsTabContent(t *testing.T) {
	tm := newTeatestApp(t, testService())

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "myproject")
	}, teatest.WithDuration(2*time.Second))

	// Switch to MCPs tab (right twice: Profiles → Tools → MCPs).
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})

	// Wait for MCP detail pane with server info.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "github") && strings.Contains(s, "qdrant")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_FormOverlay(t *testing.T) {
	tm := newTeatestApp(t, testService())

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "myproject")
	}, teatest.WithDuration(2*time.Second))

	// Switch to MCPs tab (right twice).
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "github")
	}, teatest.WithDuration(2*time.Second))

	// Press 'a' to open add form.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Wait for form overlay — form shows transport selector and field labels.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "Transport") && strings.Contains(s, "server-name")
	}, teatest.WithDuration(2*time.Second))

	// Press Esc to cancel form.
	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})

	// Wait for return to MCPs browse mode.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Transport:")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}
