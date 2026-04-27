package config

import (
	"encoding/json"
	"strings"

	"github.com/rotisserie/eris"
	"go.yaml.in/yaml/v3"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
)

// MetricSpec combines a metric name and an optional palette name into a single
// value. It implements encoding.TextUnmarshaler so Kong can parse "metric,palette"
// from the command line, and custom YAML/JSON marshaling for config files.
//
// Format: "metric" or "metric,palette".
type MetricSpec struct { //nolint:recvcheck // marshal methods need value receivers, unmarshal need pointer
	Metric  metric.Name         `json:"metric" yaml:"metric"`
	Palette palette.PaletteName `json:"palette,omitempty" yaml:"palette,omitempty"`
}

// IsZero reports whether the MetricSpec is empty (no metric specified).
func (m MetricSpec) IsZero() bool {
	return m.Metric == ""
}

// String returns the canonical text form: "metric,palette" or just "metric".
func (m MetricSpec) String() string {
	if m.Palette != "" {
		return string(m.Metric) + "," + string(m.Palette)
	}

	return string(m.Metric)
}

// UnmarshalText parses "metric,palette" or "metric" from text.
// Implements encoding.TextUnmarshaler for Kong CLI integration.
func (m *MetricSpec) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	if s == "" {
		*m = MetricSpec{}

		return nil
	}

	metricPart, rest, hasSep := strings.Cut(s, ",")
	m.Metric = metric.Name(strings.TrimSpace(metricPart))

	if m.Metric == "" {
		return eris.New("metric name must not be empty in metric spec")
	}

	if hasSep {
		palettePart, extra, hasExtra := strings.Cut(rest, ",")
		m.Palette = palette.PaletteName(strings.TrimSpace(palettePart))

		if m.Palette == "" {
			return eris.Errorf("palette name must not be empty after comma in %q", s)
		}

		if hasExtra {
			return eris.Errorf("unexpected extra content %q after palette in %q", strings.TrimSpace(extra), s)
		}
	}

	return nil
}

// Validate checks that the metric name (if set) is a known metric and that
// the palette name (if set) is a valid palette. The label describes the field
// being validated (e.g. "fill" or "border") for error messages.
// A nil receiver is valid (no metric specified).
func (m *MetricSpec) Validate(label string) error {
	if m == nil {
		return nil
	}

	if m.Metric != "" {
		if _, ok := provider.Get(m.Metric); !ok {
			names := provider.Names()
			strs := make([]string, len(names))

			for i, n := range names {
				strs[i] = string(n)
			}

			return eris.Errorf("invalid %s metric %q; available metrics: %s", label, m.Metric, strings.Join(strs, ", "))
		}
	}

	if m.Palette != "" {
		if !m.Palette.IsValid() {
			return eris.Errorf("invalid %s palette %q", label, m.Palette)
		}
	}

	return nil
}

// MarshalText produces the canonical "metric,palette" or "metric" form.
// Implements encoding.TextMarshaler.
func (m MetricSpec) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// MarshalYAML produces a YAML mapping with metric and palette fields.
func (m MetricSpec) MarshalYAML() (any, error) {
	type plain MetricSpec // strips methods to avoid recursion via TextMarshaler

	return plain(m), nil
}

// UnmarshalYAML reads a MetricSpec from a YAML mapping with metric and palette fields.
// Also accepts a scalar string for backward compatibility (delegating to UnmarshalText).
func (m *MetricSpec) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		return m.UnmarshalText([]byte(value.Value))
	}

	type plain MetricSpec // strips methods to avoid recursion via TextUnmarshaler

	var p plain
	if err := value.Decode(&p); err != nil {
		return eris.Wrap(err, "failed to decode metric spec from YAML")
	}

	*m = MetricSpec(p)

	return nil
}

// MarshalJSON produces a JSON object with metric and palette fields.
func (m MetricSpec) MarshalJSON() ([]byte, error) {
	type plain MetricSpec // strips methods to avoid recursion via TextMarshaler

	data, err := json.Marshal(plain(m))
	if err != nil {
		return nil, eris.Wrap(err, "failed to marshal metric spec to JSON")
	}

	return data, nil
}

// UnmarshalJSON reads a MetricSpec from a JSON object with metric and palette fields.
// Also accepts a JSON string for backward compatibility (delegating to UnmarshalText).
func (m *MetricSpec) UnmarshalJSON(data []byte) error {
	// Try string first for backward compatibility.
	var s string
	if json.Unmarshal(data, &s) == nil {
		return m.UnmarshalText([]byte(s))
	}

	type plain MetricSpec // strips methods to avoid recursion via TextUnmarshaler

	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return eris.Wrap(err, "failed to decode metric spec from JSON")
	}

	*m = MetricSpec(p)

	return nil
}
