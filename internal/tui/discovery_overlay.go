package tui

import (
	"fmt"
	"strings"

	"github.com/hystak/hystak/internal/discovery"
)

// discoveryModel displays scan results.
type discoveryModel struct {
	candidates []discovery.Candidate
	imported   int
}

func newDiscoveryModel(candidates []discovery.Candidate, imported int) discoveryModel {
	return discoveryModel{candidates: candidates, imported: imported}
}

func (m discoveryModel) helpKeys() []HelpEntry {
	return []HelpEntry{{"Esc", "Close"}}
}

func (m discoveryModel) view(width, height int) string {
	var b strings.Builder
	b.WriteString("Discovery Results\n\n")

	if len(m.candidates) == 0 {
		b.WriteString("  No new MCP servers found.\n")
	} else {
		b.WriteString(fmt.Sprintf("  Found %d server(s), imported %d:\n\n", len(m.candidates), m.imported))
		for _, c := range m.candidates {
			b.WriteString(fmt.Sprintf("  %s (%s) from %s\n", c.Name, c.Server.Transport, truncate(c.Source, 40)))
		}
	}

	b.WriteString("\n  " + styleHelpKey.Render("Esc") + styleHelpDesc.Render(":Close"))

	content := b.String()
	box := styleOverlayBorder.Width(min(width-4, 60)).Render(content)
	return centerOverlay(box, width, height)
}
