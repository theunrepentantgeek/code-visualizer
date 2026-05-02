//nolint:dupl // Different visualization types will evolve in different ways
package config

// Radial holds persistent configuration for radial tree visualizations.
// All fields are pointers: nil means not configured, non-nil means explicitly set.
type Radial struct {
	DiscSize          *string     `yaml:"discSize,omitempty"          json:"discSize,omitempty"`
	Fill              *MetricSpec `yaml:"fill,omitempty"              json:"fill,omitempty"`
	Border            *MetricSpec `yaml:"border,omitempty"            json:"border,omitempty"`
	Labels            *string     `yaml:"labels,omitempty"            json:"labels,omitempty"`
	Legend            *string     `yaml:"legend,omitempty"            json:"legend,omitempty"`
	LegendOrientation *string     `yaml:"legendOrientation,omitempty" json:"legendOrientation,omitempty"`
}

// OverrideDiscSize sets DiscSize to v if v is non-empty.
func (r *Radial) OverrideDiscSize(v string) { overrideString(&r.DiscSize, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (r *Radial) OverrideFill(v MetricSpec) { overrideMetricSpec(&r.Fill, v) }

// OverrideBorder sets Border to v if v is non-zero.
func (r *Radial) OverrideBorder(v MetricSpec) { overrideMetricSpec(&r.Border, v) }

// OverrideLabels sets Labels to v if v is non-empty.
func (r *Radial) OverrideLabels(v string) { overrideString(&r.Labels, v) }

// OverrideLegend sets Legend to v if v is non-empty.
func (r *Radial) OverrideLegend(v string) { overrideString(&r.Legend, v) }

// OverrideLegendOrientation sets LegendOrientation to v if v is non-empty.
func (r *Radial) OverrideLegendOrientation(v string) { overrideString(&r.LegendOrientation, v) }
