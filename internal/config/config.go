// Package config provides the application configuration model, including
// default construction, file loading, and file export.
package config

// TreemapConfig holds persistent configuration for the treemap render command.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type TreemapConfig struct {
	Width         *int    `yaml:"width,omitempty"         json:"width,omitempty"`
	Height        *int    `yaml:"height,omitempty"        json:"height,omitempty"`
	Fill          *string `yaml:"fill,omitempty"          json:"fill,omitempty"`
	FillPalette   *string `yaml:"fillPalette,omitempty"   json:"fillPalette,omitempty"`
	Border        *string `yaml:"border,omitempty"        json:"border,omitempty"`
	BorderPalette *string `yaml:"borderPalette,omitempty" json:"borderPalette,omitempty"`
}

// Config is the root configuration struct for the application.
// It is the single source of truth for all configuration, regardless of
// whether values came from defaults, a config file, or CLI flags.
type Config struct {
	Treemap *TreemapConfig `yaml:"treemap,omitempty" json:"treemap,omitempty"`
}

// New returns a Config populated with sensible defaults.
// Call this unconditionally at startup; subsequent layers (config file, CLI
// flags) overlay their values on top of the struct returned here.
func New() *Config {
	width := 1920
	height := 1080

	return &Config{
		Treemap: &TreemapConfig{
			Width:  &width,
			Height: &height,
		},
	}
}
