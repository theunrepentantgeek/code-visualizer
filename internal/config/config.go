// Package config provides the application configuration model, including
// default construction, file loading, and file export.
package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	"go.yaml.in/yaml/v3"

	"github.com/bevan/code-visualizer/internal/filter"
)

const (
	extYAML = ".yaml"
	extYML  = ".yml"
	extJSON = ".json"
)

// Config is the root configuration struct for the application.
// It is the single source of truth for all configuration, regardless of
// whether values came from defaults, a config file, or CLI flags.
// All fields are optional: nil or empty means the field was not configured,
// non-nil or non-empty means it was explicitly set (by a config file or
// by a CLI flag override).
type Config struct {
	Width      *int          `yaml:"width,omitempty"      json:"width,omitempty"`
	Height     *int          `yaml:"height,omitempty"     json:"height,omitempty"`
	Treemap    *Treemap      `yaml:"treemap,omitempty"    json:"treemap,omitempty"`
	Radial     *Radial       `yaml:"radial,omitempty"     json:"radial,omitempty"`
	Bubbletree *Bubbletree   `yaml:"bubbletree,omitempty" json:"bubbletree,omitempty"`
	Spiral     *Spiral       `yaml:"spiral,omitempty"     json:"spiral,omitempty"`
	FileFilter []filter.Rule `yaml:"fileFilter,omitempty" json:"fileFilter,omitempty"`

	// Source is the path of the config file from which this Config was loaded, or nil if it was not loaded from a file.
	Source *string `yaml:"-" json:"-"`
}

// New returns a Config populated with sensible defaults.
// Call this unconditionally at startup; subsequent layers (config file, CLI
// flags) overlay their values on top of the struct returned here.
func New() *Config {
	width := 1920
	height := 1080

	return &Config{
		Width:      &width,
		Height:     &height,
		Treemap:    &Treemap{},
		Radial:     &Radial{Labels: new("all")},
		Bubbletree: &Bubbletree{Labels: new("folders")},
		Spiral:     &Spiral{Resolution: new("daily"), Labels: new("laps")},
		FileFilter: []filter.Rule{
			{Pattern: ".*", Mode: filter.Exclude},
		},
	}
}

// Load reads the file at path and unmarshals it on top of c.
// The format is determined by the file extension (.yaml/.yml → YAML, .json → JSON).
// Any field absent from the file retains the value already set in c.
// Returns an error if the file has an unknown extension, cannot be read, or fails to parse.
func (c *Config) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return eris.Wrapf(err, "failed to read config file %q", path)
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case extYAML, extYML:
		if err := yaml.Unmarshal(data, c); err != nil {
			return eris.Wrapf(err, "failed to parse YAML config file %q", path)
		}
	case extJSON:
		if err := json.Unmarshal(data, c); err != nil {
			return eris.Wrapf(err, "failed to parse JSON config file %q", path)
		}
	default:
		return eris.Errorf("unsupported config file extension %q (use .yaml, .yml, or .json)", ext)
	}

	// Record the source path for informational purposes.
	c.Source = &path

	return nil
}

func (c *Config) TryAutoLoad(outputPath string) error {
	if c.Source != nil {
		// Already loaded from a file, no need to autoload
		return nil
	}

	if autoPath, ok := FindAutoConfig(outputPath); ok {
		slog.Info("Auto-loading config", "path", autoPath)

		if err := c.Load(autoPath); err != nil {
			return eris.Wrap(err, "auto-config load failed")
		}
	}

	return nil
}

// Save writes c to path in YAML or JSON format, determined by the file extension.
// The format is: .yaml/.yml → YAML, .json → JSON.
// The file is created or overwritten with mode 0600.
func (c *Config) Save(path string) error {
	ext := strings.ToLower(filepath.Ext(path))

	var (
		data []byte
		err  error
	)

	switch ext {
	case extYAML, extYML:
		data, err = yaml.Marshal(c)
		if err != nil {
			return eris.Wrap(err, "failed to marshal config to YAML")
		}
	case extJSON:
		data, err = json.MarshalIndent(c, "", "  ")
		if err != nil {
			return eris.Wrap(err, "failed to marshal config to JSON")
		}

		data = append(data, '\n')
	default:
		return eris.Errorf("unsupported config file extension %q (use .yaml, .yml, or .json)", ext)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return eris.Wrapf(err, "failed to write config file %q", path)
	}

	slog.Info("Config saved", "path", path)

	return nil
}

// OverrideWidth sets Width to v if v is non-zero.
func (c *Config) OverrideWidth(v int) { overrideInt(&c.Width, v) }

// OverrideHeight sets Height to v if v is non-zero.
func (c *Config) OverrideHeight(v int) { overrideInt(&c.Height, v) }

// FindAutoConfig looks for a config file alongside the output file.
// It strips the output file extension, appends "-config", and probes for
// .yml, .yaml, and .json variants in that order.
// Returns the path and true if found, or ("", false) if none exists.
func FindAutoConfig(outputPath string) (string, bool) {
	base := strings.TrimSuffix(outputPath, filepath.Ext(outputPath))

	for _, ext := range []string{extYML, extYAML, extJSON} {
		candidate := base + "-config" + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}

	return "", false
}
