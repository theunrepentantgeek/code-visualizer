package config

// Radial holds persistent configuration for radial tree visualizations.
// All fields are pointers: nil means not configured, non-nil means explicitly set.
type Radial struct {
	DiscSize        *string     `yaml:"discSize,omitempty"         json:"discSize,omitempty"`
	Fill            *MetricSpec `yaml:"fill,omitempty"             json:"fill,omitempty"`
	Border          *MetricSpec `yaml:"border,omitempty"           json:"border,omitempty"`
	DirectoryFill   *MetricSpec `yaml:"directoryFill,omitempty"    json:"directoryFill,omitempty"`
	DirectoryBorder *MetricSpec `yaml:"directoryBorder,omitempty"  json:"directoryBorder,omitempty"`
	Labels          *string     `yaml:"labels,omitempty"           json:"labels,omitempty"`
	Grain           *string     `yaml:"grain,omitempty"            json:"grain,omitempty"`
}

// OverrideDiscSize sets DiscSize to v if v is non-empty.
func (r *Radial) OverrideDiscSize(v string) { overrideString(&r.DiscSize, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (r *Radial) OverrideFill(v MetricSpec) { overrideMetricSpec(&r.Fill, v) }

// OverrideBorder sets Border to v if v is non-zero.
func (r *Radial) OverrideBorder(v MetricSpec) { overrideMetricSpec(&r.Border, v) }

// OverrideDirectoryFill sets DirectoryFill to v if v is non-zero.
func (r *Radial) OverrideDirectoryFill(v MetricSpec) { overrideMetricSpec(&r.DirectoryFill, v) }

// OverrideDirectoryBorder sets DirectoryBorder to v if v is non-zero.
func (r *Radial) OverrideDirectoryBorder(v MetricSpec) {
	overrideMetricSpec(&r.DirectoryBorder, v)
}

// OverrideLabels sets Labels to v if v is non-empty.
func (r *Radial) OverrideLabels(v string) { overrideString(&r.Labels, v) }

// OverrideGrain sets Grain to v if v is non-empty.
func (r *Radial) OverrideGrain(v string) { overrideString(&r.Grain, v) }
