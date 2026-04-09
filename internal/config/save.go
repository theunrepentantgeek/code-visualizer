package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rotisserie/eris"
)

// Save writes cfg to path in YAML or JSON format, determined by the file extension.
// The format is: .yaml/.yml → YAML, .json → JSON.
// The file is created or overwritten with mode 0600.
func Save(path string, cfg *Config) error {
	ext := strings.ToLower(filepath.Ext(path))

	var (
		data []byte
		err  error
	)

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cfg)
		if err != nil {
			return eris.Wrap(err, "failed to marshal config to YAML")
		}
	case ".json":
		data, err = json.MarshalIndent(cfg, "", "  ")
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
