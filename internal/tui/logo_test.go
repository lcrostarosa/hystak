package tui

import (
	"strings"
	"testing"
)

func TestRenderLogo_ContainsArt(t *testing.T) {
	logo := RenderLogo(80)
	// The ASCII art renders "hystak" as box-drawing characters, not literal text.
	// Check for a distinctive part of the art.
	if !strings.Contains(logo, "|___/") {
		t.Errorf("expected logo to contain ASCII art, got:\n%s", logo)
	}
}

func TestRenderLogo_ContainsSubtitle(t *testing.T) {
	logo := RenderLogo(80)
	if !strings.Contains(logo, "session launcher") {
		t.Errorf("expected logo to contain subtitle, got:\n%s", logo)
	}
}

func TestRenderLogo_NoPanic_ZeroWidth(t *testing.T) {
	logo := RenderLogo(0)
	if logo == "" {
		t.Error("expected non-empty logo at zero width")
	}
}

func TestRenderLogo_NoPanic_NarrowWidth(t *testing.T) {
	logo := RenderLogo(20)
	if logo == "" {
		t.Error("expected non-empty logo at narrow width")
	}
}

func TestRenderLogo_NoPanic_WideWidth(t *testing.T) {
	logo := RenderLogo(200)
	if logo == "" {
		t.Error("expected non-empty logo at wide width")
	}
}

func TestLogoHeight(t *testing.T) {
	h := logoHeight()
	if h < 3 {
		t.Errorf("expected logo height >= 3, got %d", h)
	}
}
