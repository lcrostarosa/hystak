package keyconfig

import (
	"errors"
	"io/fs"
	"os"

	"github.com/hystak/hystak/internal/config"
	hysterr "github.com/hystak/hystak/internal/errors"
	"gopkg.in/yaml.v3"
)

// Profile is a named keybinding preset.
type Profile string

const (
	ProfileArrows  Profile = "arrows"
	ProfileVim     Profile = "vim"
	ProfileClassic Profile = "classic"
)

// Valid reports whether p is a known keybinding profile.
func (p Profile) Valid() bool {
	switch p {
	case ProfileArrows, ProfileVim, ProfileClassic:
		return true
	}
	return false
}

// Bindings maps action names to key sequences.
type Bindings map[string][]string

// Config is the keybinding configuration from keys.yaml.
type Config struct {
	Profile  Profile  `yaml:"profile"`
	Bindings Bindings `yaml:"bindings,omitempty"`
}

// DefaultConfig returns the default keybinding config (arrows profile).
func DefaultConfig() Config {
	return Config{
		Profile: ProfileArrows,
	}
}

// defaultBindings returns the built-in bindings for each profile.
var defaultBindings = map[Profile]Bindings{
	ProfileArrows: {
		"next_tab":       {"Tab"},
		"prev_tab":       {"Shift+Tab"},
		"list_up":        {"Up"},
		"list_down":      {"Down"},
		"page_up":        {"PgUp"},
		"page_down":      {"PgDn"},
		"top":            {"Home"},
		"bottom":         {"End"},
		"select":         {"Space"},
		"confirm":        {"Enter"},
		"cancel":         {"Esc", "q"},
		"add":            {"a"},
		"edit":           {"e"},
		"delete":         {"d"},
		"filter":         {"/"},
		"launch":         {"l"},
		"import":         {"i"},
		"preview":        {"p"},
		"sync_from_diff": {"s"},
	},
	ProfileVim: {
		"next_tab":       {"Tab", "l"},
		"prev_tab":       {"Shift+Tab", "h"},
		"list_up":        {"k"},
		"list_down":      {"j"},
		"page_up":        {"Ctrl+u"},
		"page_down":      {"Ctrl+d"},
		"top":            {"g"},
		"bottom":         {"G"},
		"select":         {"Space"},
		"confirm":        {"Enter"},
		"cancel":         {"Esc", "q"},
		"add":            {"a"},
		"edit":           {"e"},
		"delete":         {"d"},
		"filter":         {"/"},
		"launch":         {"Ctrl+l"},
		"import":         {"i"},
		"preview":        {"p"},
		"sync_from_diff": {"s"},
	},
	ProfileClassic: {
		"next_tab":       {"Tab"},
		"prev_tab":       {"Shift+Tab"},
		"list_up":        {"Up"},
		"list_down":      {"Down"},
		"page_up":        {"PgUp"},
		"page_down":      {"PgDn"},
		"top":            {"Home"},
		"bottom":         {"End"},
		"select":         {"Space"},
		"confirm":        {"Enter"},
		"cancel":         {"Esc", "q"},
		"add":            {"a"},
		"edit":           {"e"},
		"delete":         {"d"},
		"filter":         {"/"},
		"launch":         {"l"},
		"import":         {"i"},
		"preview":        {"p"},
		"sync_from_diff": {"s"},
	},
}

// ResolvedBindings returns the effective bindings for the config.
// User overrides in Bindings take precedence over profile defaults.
func (c Config) ResolvedBindings() Bindings {
	base, ok := defaultBindings[c.Profile]
	if !ok {
		base = defaultBindings[ProfileArrows]
	}

	resolved := make(Bindings, len(base))
	for k, v := range base {
		resolved[k] = v
	}
	for k, v := range c.Bindings {
		resolved[k] = v
	}
	return resolved
}

// Load reads keys.yaml from the given path.
// Returns the default config if the file does not exist.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		// file exists
	case errors.Is(err, fs.ErrNotExist):
		return DefaultConfig(), nil
	default:
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, &hysterr.ConfigParseError{Path: path, Err: err}
	}
	return cfg, nil
}

// Save writes keys.yaml atomically to the given path.
func Save(cfg Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return config.AtomicWrite(path, data, 0o644)
}

// LoadDefault loads keys.yaml from the default config directory.
func LoadDefault() (Config, error) {
	return Load(config.KeysConfigPath())
}

// SaveDefault saves keys.yaml to the default config directory.
func SaveDefault(cfg Config) error {
	return Save(cfg, config.KeysConfigPath())
}
