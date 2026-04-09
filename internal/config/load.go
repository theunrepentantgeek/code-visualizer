package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rotisserie/eris"
)

// Load reads the file at path and unmarshals it on top of cfg.
// The format is determined by the file extension (.yaml/.yml → YAML, .json → JSON).
// Any field absent from the file retains the value already set in cfg.
// Returns an error if the file has an unknown extension, cannot be read, or fails to parse.
func Load(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return eris.Wrapf(err, "failed to read config file %q", path)
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return eris.Wrapf(err, "failed to parse YAML config file %q", path)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return eris.Wrapf(err, "failed to parse JSON config file %q", path)
		}
	default:
		return eris.Errorf("unsupported config file extension %q (use .yaml, .yml, or .json)", ext)
	}

	return nil
}
