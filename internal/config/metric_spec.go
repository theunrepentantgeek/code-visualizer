package config

import (
	"encoding/json"
	"strings"

	"github.com/rotisserie/eris"
	"go.yaml.in/yaml/v3"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
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

// MetricName returns the metric name, or "" if the receiver is nil or empty.
// Safe to call on a nil *MetricSpec.
func (m *MetricSpec) MetricName() metric.Name {
	if m == nil {
		return ""
	}

	return m.Metric
}

// PaletteName returns the palette name, or "" if the receiver is nil or empty.
// Safe to call on a nil *MetricSpec.
func (m *MetricSpec) PaletteName() palette.PaletteName {
	if m == nil {
		return ""
	}

	return m.Palette
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
		if err := m.validateMetric(label); err != nil {
			return err
		}
	}

	if m.Palette != "" {
		if !m.Palette.IsValid() {
			return eris.Errorf("invalid %s palette %q", label, m.Palette)
		}
	}

	return nil
}

func (m *MetricSpec) validateMetric(label string) error {
	name := string(m.Metric)

	// Try expression parse + resolve
	expr, parseErr := metric.ParseExpression(name)
	if parseErr == nil {
		_, resolveErr := provider.ResolveExpression(expr, metric.LevelFile)
		if resolveErr == nil {
			return nil
		}

		return eris.Wrapf(resolveErr, "invalid %s metric", label)
	}

	// If it doesn't parse as an expression, check if it's a known base metric
	if _, ok := provider.GetBase(m.Metric); ok {
		return nil
	}

	// Not found — provide helpful error
	return eris.Errorf(
		"invalid %s metric %q; use expression syntax: [filter.]metric[.aggregation]",
		label, m.Metric,
	)
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
