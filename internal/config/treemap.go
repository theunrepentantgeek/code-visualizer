package config

// Treemap holds persistent configuration for treemap visualizations.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Treemap struct {
	Size              *string     `yaml:"size,omitempty"              json:"size,omitempty"`
	Fill              *MetricSpec `yaml:"fill,omitempty"              json:"fill,omitempty"`
	Border            *MetricSpec `yaml:"border,omitempty"            json:"border,omitempty"`
}

// OverrideSize sets Size to v if v is non-empty.
func (t *Treemap) OverrideSize(v string) { overrideString(&t.Size, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (t *Treemap) OverrideFill(v MetricSpec) { overrideMetricSpec(&t.Fill, v) }

// OverrideBorder sets Border to v if v is non-zero.
func (t *Treemap) OverrideBorder(v MetricSpec) { overrideMetricSpec(&t.Border, v) }
