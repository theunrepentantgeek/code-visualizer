// Package config provides the application configuration model, including
// default construction, file loading, and file export.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rotisserie/eris"
)

// Config is the root configuration struct for the application.
// It is the single source of truth for all configuration, regardless of
// whether values came from defaults, a config file, or CLI flags.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Config struct {
	Width   *int     `yaml:"width,omitempty"   json:"width,omitempty"`
	Height  *int     `yaml:"height,omitempty"  json:"height,omitempty"`
	Treemap *Treemap `yaml:"treemap,omitempty" json:"treemap,omitempty"`
}

// New returns a Config populated with sensible defaults.
// Call this unconditionally at startup; subsequent layers (config file, CLI
// flags) overlay their values on top of the struct returned here.
func New() *Config {
	width := 1920
	height := 1080

	return &Config{
		Width:   &width,
		Height:  &height,
		Treemap: &Treemap{},
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
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, c); err != nil {
			return eris.Wrapf(err, "failed to parse YAML config file %q", path)
		}
	case ".json":
		if err := json.Unmarshal(data, c); err != nil {
			return eris.Wrapf(err, "failed to parse JSON config file %q", path)
		}
	default:
		return eris.Errorf("unsupported config file extension %q (use .yaml, .yml, or .json)", ext)
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
	case ".yaml", ".yml":
		data, err = yaml.Marshal(c)
		if err != nil {
			return eris.Wrap(err, "failed to marshal config to YAML")
		}
	case ".json":
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

	return nil
}
