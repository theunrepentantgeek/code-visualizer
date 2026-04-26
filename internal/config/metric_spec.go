package config

import (
	"encoding/json"
	"strings"

	"github.com/rotisserie/eris"
	"go.yaml.in/yaml/v3"
)

// MetricSpec combines a metric name and an optional palette name into a single
// value. It implements encoding.TextUnmarshaler so Kong can parse "metric,palette"
// from the command line, and custom YAML/JSON marshaling for config files.
//
// Format: "metric" or "metric,palette".
type MetricSpec struct { //nolint:recvcheck // marshal methods need value receivers, unmarshal need pointer
	Metric  string
	Palette string
}

// IsZero reports whether the MetricSpec is empty (no metric specified).
func (m MetricSpec) IsZero() bool {
	return m.Metric == ""
}

// String returns the canonical text form: "metric,palette" or just "metric".
func (m MetricSpec) String() string {
	if m.Palette != "" {
		return m.Metric + "," + m.Palette
	}

	return m.Metric
}

// UnmarshalText parses "metric,palette" or "metric" from text.
// Implements encoding.TextUnmarshaler for Kong CLI integration.
func (m *MetricSpec) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	if s == "" {
		*m = MetricSpec{}

		return nil
	}

	parts := strings.SplitN(s, ",", 2)
	m.Metric = strings.TrimSpace(parts[0])

	if m.Metric == "" {
		return eris.New("metric name must not be empty in metric spec")
	}

	if len(parts) == 2 {
		m.Palette = strings.TrimSpace(parts[1])

		if m.Palette == "" {
			return eris.Errorf("palette name must not be empty after comma in %q", s)
		}
	}

	return nil
}

// MarshalText produces the canonical "metric,palette" or "metric" form.
// Implements encoding.TextMarshaler.
func (m MetricSpec) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// MarshalYAML produces a scalar string for YAML output.
func (m MetricSpec) MarshalYAML() (any, error) {
	return m.String(), nil
}

// UnmarshalYAML reads a MetricSpec from a YAML scalar string.
func (m *MetricSpec) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return eris.New("metric spec must be a string")
	}

	return m.UnmarshalText([]byte(value.Value))
}

// MarshalJSON produces a JSON string.
func (m MetricSpec) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(m.String())
	if err != nil {
		return nil, eris.Wrap(err, "failed to marshal metric spec to JSON")
	}

	return data, nil
}

// UnmarshalJSON reads a MetricSpec from a JSON string.
func (m *MetricSpec) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return eris.Wrap(err, "metric spec must be a JSON string")
	}

	return m.UnmarshalText([]byte(s))
}
