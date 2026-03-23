package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	hysterr "github.com/hystak/hystak/internal/errors"
)

func TestLoadUserConfig_Defaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	cfg, err := LoadUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultUserConfig()
	if cfg.AutoSync != want.AutoSync {
		t.Errorf("AutoSync = %v, want %v", cfg.AutoSync, want.AutoSync)
	}
	if cfg.BackupPolicy != want.BackupPolicy {
		t.Errorf("BackupPolicy = %q, want %q", cfg.BackupPolicy, want.BackupPolicy)
	}
	if cfg.MaxBackups != want.MaxBackups {
		t.Errorf("MaxBackups = %d, want %d", cfg.MaxBackups, want.MaxBackups)
	}
}

func TestLoadUserConfig_CustomValues(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	content := []byte("auto_sync: false\nbackup_policy: never\nmax_backups: 5\n")
	if err := os.WriteFile(filepath.Join(tmp, "user.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AutoSync {
		t.Error("AutoSync should be false")
	}
	if cfg.BackupPolicy != "never" {
		t.Errorf("BackupPolicy = %q, want %q", cfg.BackupPolicy, "never")
	}
	if cfg.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d, want %d", cfg.MaxBackups, 5)
	}
}

func TestLoadUserConfig_MalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	content := []byte("auto_sync: [invalid yaml\n")
	if err := os.WriteFile(filepath.Join(tmp, "user.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadUserConfig()
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
	if _, ok := err.(*hysterr.ConfigParseError); !ok {
		t.Errorf("expected *ConfigParseError, got %T: %v", err, err)
	}
}

func TestSaveUserConfig_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HYSTAK_CONFIG_DIR", tmp)

	original := UserConfig{
		AutoSync:     false,
		BackupPolicy: "never",
		MaxBackups:   3,
	}
	if err := SaveUserConfig(original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadUserConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", loaded, original)
	}
}
