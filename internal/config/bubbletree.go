//nolint:dupl // Different visualization types will evolve in different ways
package config

// Bubbletree holds persistent configuration for bubble tree visualizations.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Bubbletree struct {
	Size              *string     `yaml:"size,omitempty"              json:"size,omitempty"`
	Fill              *MetricSpec `yaml:"fill,omitempty"              json:"fill,omitempty"`
	Border            *MetricSpec `yaml:"border,omitempty"            json:"border,omitempty"`
	Labels            *string     `yaml:"labels,omitempty"            json:"labels,omitempty"`
	Legend            *string     `yaml:"legend,omitempty"            json:"legend,omitempty"`
	LegendOrientation *string     `yaml:"legendOrientation,omitempty" json:"legendOrientation,omitempty"`
}

// OverrideSize sets Size to v if v is non-empty.
func (b *Bubbletree) OverrideSize(v string) { overrideString(&b.Size, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (b *Bubbletree) OverrideFill(v MetricSpec) { overrideMetricSpec(&b.Fill, v) }

// OverrideBorder sets Border to v if v is non-zero.
func (b *Bubbletree) OverrideBorder(v MetricSpec) { overrideMetricSpec(&b.Border, v) }

// OverrideLabels sets Labels to v if v is non-empty.
func (b *Bubbletree) OverrideLabels(v string) { overrideString(&b.Labels, v) }

// OverrideLegend sets Legend to v if v is non-empty.
func (b *Bubbletree) OverrideLegend(v string) { overrideString(&b.Legend, v) }

// OverrideLegendOrientation sets LegendOrientation to v if v is non-empty.
func (b *Bubbletree) OverrideLegendOrientation(v string) { overrideString(&b.LegendOrientation, v) }
