package keyconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPresetArrows(t *testing.T) {
	r, err := Preset(ProfileArrows)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Global.TabNext; len(got) != 1 || got[0] != "right" {
		t.Errorf("arrows TabNext = %v, want [right]", got)
	}
	if got := r.Global.TabPrev; len(got) != 1 || got[0] != "left" {
		t.Errorf("arrows TabPrev = %v, want [left]", got)
	}
}

func TestPresetVim(t *testing.T) {
	r, err := Preset(ProfileVim)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Global.TabNext; len(got) != 1 || got[0] != "l" {
		t.Errorf("vim TabNext = %v, want [l]", got)
	}
	if got := r.Global.TabPrev; len(got) != 1 || got[0] != "h" {
		t.Errorf("vim TabPrev = %v, want [h]", got)
	}
}

func TestPresetClassic(t *testing.T) {
	r, err := Preset(ProfileClassic)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Global.TabNext; len(got) != 1 || got[0] != "tab" {
		t.Errorf("classic TabNext = %v, want [tab]", got)
	}
}

func TestPresetUnknown(t *testing.T) {
	_, err := Preset("nonexistent")
	if err == nil {
		t.Error("expected error for unknown preset")
	}
}

func TestPresetEmpty(t *testing.T) {
	r, err := Preset("")
	if err != nil {
		t.Fatal(err)
	}
	// Empty string falls back to arrows.
	if got := r.Global.TabNext; len(got) != 1 || got[0] != "right" {
		t.Errorf("empty preset TabNext = %v, want [right]", got)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != ProfileArrows {
		t.Errorf("missing file profile = %q, want %q", cfg.Profile, ProfileArrows)
	}
}

func TestLoadEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "keys.yaml")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != ProfileArrows {
		t.Errorf("empty file profile = %q, want %q", cfg.Profile, ProfileArrows)
	}
}

func TestLoadValid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "keys.yaml")
	content := `profile: vim
overrides:
  global:
    quit: ["q"]
`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != ProfileVim {
		t.Errorf("profile = %q, want %q", cfg.Profile, ProfileVim)
	}
	if len(cfg.Overrides.Global.Quit) != 1 || cfg.Overrides.Global.Quit[0] != "q" {
		t.Errorf("overrides quit = %v, want [q]", cfg.Overrides.Global.Quit)
	}
}

func TestLoadMalformed(t *testing.T) {
	p := filepath.Join(t.TempDir(), "keys.yaml")
	if err := os.WriteFile(p, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestResolveWithOverrides(t *testing.T) {
	cfg := Config{
		Profile: ProfileArrows,
		Overrides: Overrides{
			Global: GlobalKeys{
				Quit: []string{"ctrl+q"},
			},
			MCPs: ResourceTabKeys{
				Add: []string{"n"},
			},
		},
	}
	r, err := Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Quit should be overridden.
	if len(r.Global.Quit) != 1 || r.Global.Quit[0] != "ctrl+q" {
		t.Errorf("quit = %v, want [ctrl+q]", r.Global.Quit)
	}
	// TabNext should remain from arrows preset.
	if len(r.Global.TabNext) != 1 || r.Global.TabNext[0] != "right" {
		t.Errorf("tab_next = %v, want [right]", r.Global.TabNext)
	}
	// MCPs add overridden.
	if len(r.MCPs.Add) != 1 || r.MCPs.Add[0] != "n" {
		t.Errorf("mcps add = %v, want [n]", r.MCPs.Add)
	}
	// MCPs edit should remain default.
	if len(r.MCPs.Edit) != 1 || r.MCPs.Edit[0] != "e" {
		t.Errorf("mcps edit = %v, want [e]", r.MCPs.Edit)
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	cfg := Config{Profile: "bad"}
	_, err := Resolve(cfg)
	if err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestSaveAndLoad(t *testing.T) {
	p := filepath.Join(t.TempDir(), "keys.yaml")
	cfg := Config{Profile: ProfileVim}
	if err := Save(p, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profile != ProfileVim {
		t.Errorf("roundtrip profile = %q, want %q", loaded.Profile, ProfileVim)
	}
}

func TestKeysPath(t *testing.T) {
	got := KeysPath("/home/user/.hystak")
	want := "/home/user/.hystak/keys.yaml"
	if got != want {
		t.Errorf("KeysPath = %q, want %q", got, want)
	}
}

func TestPresetNames(t *testing.T) {
	names := PresetNames()
	if len(names) != 3 {
		t.Errorf("PresetNames() returned %d names, want 3", len(names))
	}
}
