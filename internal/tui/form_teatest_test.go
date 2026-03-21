package tui

import (
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/lcrostarosa/hystak/internal/model"
)

func TestTeatest_FormAddServer(t *testing.T) {
	tm := newTeatestForm(t)

	// Wait for form to render with transport selector and fields.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "Transport") && strings.Contains(s, "server-name")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_FormEditServer(t *testing.T) {
	srv := model.ServerDef{
		Name:        "github",
		Description: "GitHub MCP server",
		Transport:   model.TransportStdio,
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-github"},
		Env:         map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	}
	tm := newTeatestEditForm(t, srv)

	// Wait for edit form with pre-populated values.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "github") && strings.Contains(s, "npx")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}

func TestTeatest_FormTransportCycle(t *testing.T) {
	tm := newTeatestForm(t)

	// Wait for initial form render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Transport")
	}, teatest.WithDuration(2*time.Second))

	// Press Ctrl+T to cycle transport from stdio → sse.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlT})

	// Wait for transport change — URL field should appear for SSE.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "URL")
	}, teatest.WithDuration(2*time.Second))

	quitAndWait(t, tm)

	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(testTimeout)))
	if err != nil {
		t.Fatal(err)
	}
	teatest.RequireEqualOutput(t, out)
}
