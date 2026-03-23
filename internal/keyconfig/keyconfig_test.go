package keyconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	hysterr "github.com/hystak/hystak/internal/errors"
)

func TestProfile_Valid(t *testing.T) {
	tests := []struct {
		name  string
		value Profile
		want  bool
	}{
		{"arrows", ProfileArrows, true},
		{"vim", ProfileVim, true},
		{"classic", ProfileClassic, true},
		{"empty", Profile(""), false},
		{"unknown", Profile("emacs"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Valid(); got != tt.want {
				t.Errorf("Profile(%q).Valid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Profile != ProfileArrows {
		t.Errorf("Profile = %q, want arrows", cfg.Profile)
	}
}

func TestConfig_ResolvedBindings_DefaultProfile(t *testing.T) {
	cfg := Config{Profile: ProfileArrows}
	bindings := cfg.ResolvedBindings()

	if bindings["list_up"][0] != "Up" {
		t.Errorf("arrows list_up = %v, want [Up]", bindings["list_up"])
	}
	if bindings["list_down"][0] != "Down" {
		t.Errorf("arrows list_down = %v, want [Down]", bindings["list_down"])
	}
}

func TestConfig_ResolvedBindings_VimProfile(t *testing.T) {
	cfg := Config{Profile: ProfileVim}
	bindings := cfg.ResolvedBindings()

	if bindings["list_up"][0] != "k" {
		t.Errorf("vim list_up = %v, want [k]", bindings["list_up"])
	}
	if bindings["list_down"][0] != "j" {
		t.Errorf("vim list_down = %v, want [j]", bindings["list_down"])
	}
	// Vim next_tab should have both Tab and l
	if len(bindings["next_tab"]) != 2 {
		t.Errorf("vim next_tab = %v, want 2 keys", bindings["next_tab"])
	}
}

func TestConfig_ResolvedBindings_UserOverride(t *testing.T) {
	cfg := Config{
		Profile: ProfileArrows,
		Bindings: Bindings{
			"list_up": {"w"},
		},
	}
	bindings := cfg.ResolvedBindings()

	if !reflect.DeepEqual(bindings["list_up"], []string{"w"}) {
		t.Errorf("list_up = %v, want [w] (user override)", bindings["list_up"])
	}
	// Non-overridden keys should still have defaults
	if bindings["list_down"][0] != "Down" {
		t.Errorf("list_down = %v, want [Down] (default preserved)", bindings["list_down"])
	}
}

func TestConfig_ResolvedBindings_UnknownProfile(t *testing.T) {
	cfg := Config{Profile: Profile("unknown")}
	bindings := cfg.ResolvedBindings()

	// Should fall back to arrows
	if bindings["list_up"][0] != "Up" {
		t.Errorf("unknown profile list_up = %v, want [Up] (fallback to arrows)", bindings["list_up"])
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "keys.yaml")

	original := Config{
		Profile: ProfileVim,
		Bindings: Bindings{
			"list_up": {"w"},
		},
	}

	if err := Save(original, path); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(loaded, original) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", loaded, original)
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Profile != ProfileArrows {
		t.Errorf("Profile = %q, want arrows (default)", cfg.Profile)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "keys.yaml")

	if err := os.WriteFile(path, []byte("profile: [invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
	if _, ok := err.(*hysterr.ConfigParseError); !ok {
		t.Errorf("expected *ConfigParseError, got %T", err)
	}
}

func TestDefaultBindings_AllProfilesHaveSameActions(t *testing.T) {
	arrowsActions := make(map[string]bool)
	for action := range defaultBindings[ProfileArrows] {
		arrowsActions[action] = true
	}

	for _, profile := range []Profile{ProfileVim, ProfileClassic} {
		t.Run(string(profile), func(t *testing.T) {
			bindings := defaultBindings[profile]
			for action := range arrowsActions {
				if _, ok := bindings[action]; !ok {
					t.Errorf("profile %q missing action %q", profile, action)
				}
			}
			for action := range bindings {
				if !arrowsActions[action] {
					t.Errorf("profile %q has extra action %q not in arrows", profile, action)
				}
			}
		})
	}
}
