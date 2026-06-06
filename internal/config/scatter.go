package config

// Scatter holds persistent configuration for scatter plot visualizations.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Scatter struct {
	XAxis  *string     `yaml:"xAxis,omitempty"             json:"xAxis,omitempty"`
	YAxis  *string     `yaml:"yAxis,omitempty"             json:"yAxis,omitempty"`
	Size   *string     `yaml:"size,omitempty"              json:"size,omitempty"`
	Fill   *MetricSpec `yaml:"fill,omitempty"              json:"fill,omitempty"`
	Border *MetricSpec `yaml:"border,omitempty"            json:"border,omitempty"`
}

// OverrideXAxis sets XAxis to v if v is non-empty.
func (s *Scatter) OverrideXAxis(v string) { overrideString(&s.XAxis, v) }

// OverrideYAxis sets YAxis to v if v is non-empty.
func (s *Scatter) OverrideYAxis(v string) { overrideString(&s.YAxis, v) }

// OverrideSize sets Size to v if v is non-empty.
func (s *Scatter) OverrideSize(v string) { overrideString(&s.Size, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (s *Scatter) OverrideFill(v MetricSpec) { overrideMetricSpec(&s.Fill, v) }

// OverrideBorder sets Border to v if v is non-zero.
func (s *Scatter) OverrideBorder(v MetricSpec) { overrideMetricSpec(&s.Border, v) }
