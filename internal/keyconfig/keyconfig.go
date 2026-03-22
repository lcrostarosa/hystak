package keyconfig

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Profile names for built-in keybinding presets.
const (
	ProfileArrows  = "arrows"
	ProfileVim     = "vim"
	ProfileClassic = "classic"
)

// Config represents the user's keybinding configuration file.
type Config struct {
	Profile   string    `yaml:"profile,omitempty"`
	Overrides Overrides `yaml:"overrides,omitempty"`
}

// Overrides allows per-section key customisation on top of a profile.
type Overrides struct {
	Global      GlobalKeys      `yaml:"global,omitempty"`
	Profiles    ProfilesKeys    `yaml:"profiles,omitempty"`
	Tools       ToolsKeys       `yaml:"tools,omitempty"`
	MCPs        ResourceTabKeys `yaml:"mcps,omitempty"`
	Skills      ResourceTabKeys `yaml:"skills,omitempty"`
	Hooks       ResourceTabKeys `yaml:"hooks,omitempty"`
	Permissions ResourceTabKeys `yaml:"permissions,omitempty"`
	Templates   ResourceTabKeys `yaml:"templates,omitempty"`
	Prompts     ResourceTabKeys `yaml:"prompts,omitempty"`
}

// GlobalKeys holds keybindings for app-wide actions.
type GlobalKeys struct {
	Quit    []string `yaml:"quit,omitempty"`
	TabNext []string `yaml:"tab_next,omitempty"`
	TabPrev []string `yaml:"tab_prev,omitempty"`
}

// ProfilesKeys holds keybindings for the Profiles tab.
type ProfilesKeys struct {
	Launch    []string `yaml:"launch,omitempty"`
	Configure []string `yaml:"configure,omitempty"`
	Delete    []string `yaml:"delete,omitempty"`
}

// ToolsKeys holds keybindings for the Tools tab.
type ToolsKeys struct {
	Execute []string `yaml:"execute,omitempty"`
}

// ResourceTabKeys holds keybindings common to resource tabs (MCPs, Skills, etc.).
type ResourceTabKeys struct {
	Add    []string `yaml:"add,omitempty"`
	Edit   []string `yaml:"edit,omitempty"`
	Delete []string `yaml:"delete,omitempty"`
	Import []string `yaml:"import,omitempty"`
}

// ResolvedKeys is the fully-resolved set of keybindings ready for the TUI.
// Each field is a slice of key strings (charmbracelet format).
type ResolvedKeys struct {
	Global      GlobalKeys
	Profiles    ProfilesKeys
	Tools       ToolsKeys
	MCPs        ResourceTabKeys
	Skills      ResourceTabKeys
	Hooks       ResourceTabKeys
	Permissions ResourceTabKeys
	Templates   ResourceTabKeys
	Prompts     ResourceTabKeys
}

// Preset returns the built-in keybinding preset for the given profile name.
// Returns an error if the profile name is unknown.
func Preset(name string) (ResolvedKeys, error) {
	switch name {
	case ProfileArrows, "":
		return arrowsPreset(), nil
	case ProfileVim:
		return vimPreset(), nil
	case ProfileClassic:
		return classicPreset(), nil
	default:
		return ResolvedKeys{}, fmt.Errorf("unknown keybinding profile: %q (valid: arrows, vim, classic)", name)
	}
}

// PresetNames returns the list of valid preset profile names.
func PresetNames() []string {
	return []string{ProfileArrows, ProfileVim, ProfileClassic}
}

func arrowsPreset() ResolvedKeys {
	return ResolvedKeys{
		Global: GlobalKeys{
			Quit:    []string{"q", "ctrl+c"},
			TabNext: []string{"right"},
			TabPrev: []string{"left"},
		},
		Profiles: ProfilesKeys{
			Launch:    []string{"enter"},
			Configure: []string{"c"},
			Delete:    []string{"d"},
		},
		Tools: ToolsKeys{
			Execute: []string{"enter"},
		},
		MCPs:        defaultResourceKeys(true),
		Skills:      defaultResourceKeys(false),
		Hooks:       defaultResourceKeys(false),
		Permissions: defaultResourceKeys(false),
		Templates:   defaultResourceKeys(false),
		Prompts:     defaultResourceKeys(false),
	}
}

func vimPreset() ResolvedKeys {
	return ResolvedKeys{
		Global: GlobalKeys{
			Quit:    []string{"q", "ctrl+c"},
			TabNext: []string{"l"},
			TabPrev: []string{"h"},
		},
		Profiles: ProfilesKeys{
			Launch:    []string{"enter"},
			Configure: []string{"c"},
			Delete:    []string{"d"},
		},
		Tools: ToolsKeys{
			Execute: []string{"enter"},
		},
		MCPs:        defaultResourceKeys(true),
		Skills:      defaultResourceKeys(false),
		Hooks:       defaultResourceKeys(false),
		Permissions: defaultResourceKeys(false),
		Templates:   defaultResourceKeys(false),
		Prompts:     defaultResourceKeys(false),
	}
}

func classicPreset() ResolvedKeys {
	return ResolvedKeys{
		Global: GlobalKeys{
			Quit:    []string{"q", "ctrl+c"},
			TabNext: []string{"tab"},
			TabPrev: []string{"shift+tab"},
		},
		Profiles: ProfilesKeys{
			Launch:    []string{"enter"},
			Configure: []string{"c"},
			Delete:    []string{"d"},
		},
		Tools: ToolsKeys{
			Execute: []string{"enter"},
		},
		MCPs:        defaultResourceKeys(true),
		Skills:      defaultResourceKeys(false),
		Hooks:       defaultResourceKeys(false),
		Permissions: defaultResourceKeys(false),
		Templates:   defaultResourceKeys(false),
		Prompts:     defaultResourceKeys(false),
	}
}

func defaultResourceKeys(withImport bool) ResourceTabKeys {
	k := ResourceTabKeys{
		Add:    []string{"a"},
		Edit:   []string{"e"},
		Delete: []string{"d"},
	}
	if withImport {
		k.Import = []string{"i"}
	}
	return k
}

// Load reads a keybinding config from the given file path.
// Returns default config if the file does not exist.
// Returns an error if the file exists but is malformed.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{Profile: ProfileArrows}, nil
		}
		return Config{}, fmt.Errorf("reading key config: %w", err)
	}

	if len(data) == 0 {
		return Config{Profile: ProfileArrows}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing key config %s: %w", path, err)
	}

	if cfg.Profile == "" {
		cfg.Profile = ProfileArrows
	}

	return cfg, nil
}

// Save writes a keybinding config to the given file path.
func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshalling key config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// Resolve takes a Config (profile + overrides) and produces the final
// set of keybindings by applying overrides on top of the base preset.
func Resolve(cfg Config) (ResolvedKeys, error) {
	base, err := Preset(cfg.Profile)
	if err != nil {
		return ResolvedKeys{}, err
	}

	applyOverrides(&base, cfg.Overrides)
	return base, nil
}

// KeysPath returns the conventional keys.yaml path within a config directory.
func KeysPath(configDir string) string {
	return configDir + "/keys.yaml"
}

func applyOverrides(r *ResolvedKeys, o Overrides) {
	// Global
	overrideSlice(&r.Global.Quit, o.Global.Quit)
	overrideSlice(&r.Global.TabNext, o.Global.TabNext)
	overrideSlice(&r.Global.TabPrev, o.Global.TabPrev)

	// Profiles
	overrideSlice(&r.Profiles.Launch, o.Profiles.Launch)
	overrideSlice(&r.Profiles.Configure, o.Profiles.Configure)
	overrideSlice(&r.Profiles.Delete, o.Profiles.Delete)

	// Tools
	overrideSlice(&r.Tools.Execute, o.Tools.Execute)

	// Resource tabs
	applyResourceOverrides(&r.MCPs, o.MCPs)
	applyResourceOverrides(&r.Skills, o.Skills)
	applyResourceOverrides(&r.Hooks, o.Hooks)
	applyResourceOverrides(&r.Permissions, o.Permissions)
	applyResourceOverrides(&r.Templates, o.Templates)
	applyResourceOverrides(&r.Prompts, o.Prompts)
}

func applyResourceOverrides(dst *ResourceTabKeys, src ResourceTabKeys) {
	overrideSlice(&dst.Add, src.Add)
	overrideSlice(&dst.Edit, src.Edit)
	overrideSlice(&dst.Delete, src.Delete)
	overrideSlice(&dst.Import, src.Import)
}

func overrideSlice(dst *[]string, src []string) {
	if len(src) > 0 {
		*dst = src
	}
}
