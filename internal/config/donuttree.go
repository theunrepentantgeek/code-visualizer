package config

// DonutTree holds persistent configuration for donut tree visualizations.
// Nil fields were not configured; non-nil fields were set by a file or CLI override.
type DonutTree struct {
	Size   *string     `yaml:"size,omitempty"   json:"size,omitempty"`
	Fill   *MetricSpec `yaml:"fill,omitempty"   json:"fill,omitempty"`
	Border *MetricSpec `yaml:"border,omitempty" json:"border,omitempty"`
}

func (d *DonutTree) OverrideSize(v string)       { overrideString(&d.Size, v) }
func (d *DonutTree) OverrideFill(v MetricSpec)   { overrideMetricSpec(&d.Fill, v) }
func (d *DonutTree) OverrideBorder(v MetricSpec) { overrideMetricSpec(&d.Border, v) }
