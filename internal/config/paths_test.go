package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearOverride(t *testing.T) {
	t.Helper()
	OverrideDir("")
	t.Cleanup(func() { OverrideDir("") })
}

func TestDir_OverrideDir(t *testing.T) {
	clearOverride(t)
	OverrideDir("/override/path")
	t.Setenv("HYSTAK_CONFIG_DIR", "/should/not/use")
	if got := Dir(); got != "/override/path" {
		t.Errorf("Dir() = %q, want %q", got, "/override/path")
	}
}

func TestDir_HYSTAK_CONFIG_DIR(t *testing.T) {
	clearOverride(t)
	t.Setenv("HYSTAK_CONFIG_DIR", "/custom/hystak")
	t.Setenv("XDG_CONFIG_HOME", "/should/not/use")
	if got := Dir(); got != "/custom/hystak" {
		t.Errorf("Dir() = %q, want %q", got, "/custom/hystak")
	}
}

func TestDir_XDG_CONFIG_HOME(t *testing.T) {
	clearOverride(t)
	t.Setenv("HYSTAK_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	if got := Dir(); got != "/xdg/config/hystak" {
		t.Errorf("Dir() = %q, want %q", got, "/xdg/config/hystak")
	}
}

func TestDir_DefaultHome(t *testing.T) {
	clearOverride(t)
	t.Setenv("HYSTAK_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".hystak")
	if got := Dir(); got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestRegistryPath(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/test")
	if got := RegistryPath(); got != "/test/registry.yaml" {
		t.Errorf("RegistryPath() = %q, want %q", got, "/test/registry.yaml")
	}
}

func TestProjectsPath(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/test")
	if got := ProjectsPath(); got != "/test/projects.yaml" {
		t.Errorf("ProjectsPath() = %q, want %q", got, "/test/projects.yaml")
	}
}

func TestUserConfigPath(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/test")
	if got := UserConfigPath(); got != "/test/user.yaml" {
		t.Errorf("UserConfigPath() = %q, want %q", got, "/test/user.yaml")
	}
}

func TestKeysConfigPath(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/test")
	if got := KeysConfigPath(); got != "/test/keys.yaml" {
		t.Errorf("KeysConfigPath() = %q, want %q", got, "/test/keys.yaml")
	}
}

func TestProfilesDir(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/test")
	if got := ProfilesDir(); got != "/test/profiles" {
		t.Errorf("ProfilesDir() = %q, want %q", got, "/test/profiles")
	}
}

func TestBackupsDir(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/test")
	if got := BackupsDir(); got != "/test/backups" {
		t.Errorf("BackupsDir() = %q, want %q", got, "/test/backups")
	}
}

func TestSubdirs_AllPresent(t *testing.T) {
	want := map[string]bool{
		"profiles":  false,
		"backups":   false,
		"skills":    false,
		"templates": false,
		"prompts":   false,
	}
	for _, s := range Subdirs() {
		if _, ok := want[s]; !ok {
			t.Errorf("unexpected subdir %q", s)
		}
		want[s] = true
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing subdir %q", name)
		}
	}
}

func TestPathFunctions_UseDir(t *testing.T) {
	t.Setenv("HYSTAK_CONFIG_DIR", "/base")
	funcs := map[string]func() string{
		"RegistryPath":   RegistryPath,
		"ProjectsPath":   ProjectsPath,
		"UserConfigPath": UserConfigPath,
		"KeysConfigPath": KeysConfigPath,
		"ProfilesDir":    ProfilesDir,
		"BackupsDir":     BackupsDir,
	}
	for name, fn := range funcs {
		t.Run(name, func(t *testing.T) {
			if !strings.HasPrefix(fn(), "/base") {
				t.Errorf("%s() = %q, does not start with Dir()", name, fn())
			}
		})
	}
}
