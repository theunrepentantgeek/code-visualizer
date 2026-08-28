package config

// Radial holds persistent configuration for radial tree visualizations.
// All fields are pointers: nil means not configured, non-nil means explicitly set.
type Radial struct {
	FileDiscSize      *string     `yaml:"fileDiscSize,omitempty"      json:"fileDiscSize,omitempty"`
	FileFill          *MetricSpec `yaml:"fileFill,omitempty"          json:"fileFill,omitempty"`
	FileBorder        *MetricSpec `yaml:"fileBorder,omitempty"        json:"fileBorder,omitempty"`
	DirectoryDiscSize *string     `yaml:"directoryDiscSize,omitempty" json:"directoryDiscSize,omitempty"`
	DirectoryFill     *MetricSpec `yaml:"directoryFill,omitempty"     json:"directoryFill,omitempty"`
	DirectoryBorder   *MetricSpec `yaml:"directoryBorder,omitempty"   json:"directoryBorder,omitempty"`
	Labels            *string     `yaml:"labels,omitempty"            json:"labels,omitempty"`
	Grain             *string     `yaml:"grain,omitempty"             json:"grain,omitempty"`
	MaxLayers         *int        `yaml:"maxLayers,omitempty"         json:"maxLayers,omitempty"`
}

// OverrideFileDiscSize sets FileDiscSize to v if v is non-empty.
func (r *Radial) OverrideFileDiscSize(v string) { overrideString(&r.FileDiscSize, v) }

// OverrideFileFill sets FileFill to v if v is non-zero.
func (r *Radial) OverrideFileFill(v MetricSpec) { overrideMetricSpec(&r.FileFill, v) }

// OverrideFileBorder sets FileBorder to v if v is non-zero.
func (r *Radial) OverrideFileBorder(v MetricSpec) { overrideMetricSpec(&r.FileBorder, v) }

// OverrideDirectoryDiscSize sets DirectoryDiscSize to v if v is non-empty.
func (r *Radial) OverrideDirectoryDiscSize(v string) { overrideString(&r.DirectoryDiscSize, v) }

// OverrideDirectoryFill sets DirectoryFill to v if v is non-zero.
func (r *Radial) OverrideDirectoryFill(v MetricSpec) { overrideMetricSpec(&r.DirectoryFill, v) }

// OverrideDirectoryBorder sets DirectoryBorder to v if v is non-zero.
func (r *Radial) OverrideDirectoryBorder(v MetricSpec) { overrideMetricSpec(&r.DirectoryBorder, v) }

// OverrideLabels sets Labels to v if v is non-empty.
func (r *Radial) OverrideLabels(v string) { overrideString(&r.Labels, v) }

// OverrideGrain sets Grain to v if v is non-empty.
func (r *Radial) OverrideGrain(v string) { overrideString(&r.Grain, v) }

// OverrideMaxLayers sets MaxLayers to v if v is non-zero.
func (r *Radial) OverrideMaxLayers(v int) { overrideInt(&r.MaxLayers, v) }
