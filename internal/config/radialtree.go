//nolint:dupl // config structs and their overrides are structurally similar by design
package config

// Radial holds persistent configuration for radial tree visualizations.
// All fields are pointers: nil means not configured, non-nil means explicitly set.
type Radial struct {
	FileDiscSize   *string     `yaml:"fileDiscSize,omitempty"   json:"fileDiscSize,omitempty"`
	FileFill       *MetricSpec `yaml:"fileFill,omitempty"       json:"fileFill,omitempty"`
	FileBorder     *MetricSpec `yaml:"fileBorder,omitempty"     json:"fileBorder,omitempty"`
	FolderDiscSize *string     `yaml:"folderDiscSize,omitempty" json:"folderDiscSize,omitempty"`
	FolderFill     *MetricSpec `yaml:"folderFill,omitempty"     json:"folderFill,omitempty"`
	FolderBorder   *MetricSpec `yaml:"folderBorder,omitempty"   json:"folderBorder,omitempty"`
	Labels         *string     `yaml:"labels,omitempty"         json:"labels,omitempty"`
	Grain          *string     `yaml:"grain,omitempty"          json:"grain,omitempty"`
}

// OverrideFileDiscSize sets FileDiscSize to v if v is non-empty.
func (r *Radial) OverrideFileDiscSize(v string) { overrideString(&r.FileDiscSize, v) }

// OverrideFileFill sets FileFill to v if v is non-zero.
func (r *Radial) OverrideFileFill(v MetricSpec) { overrideMetricSpec(&r.FileFill, v) }

// OverrideFileBorder sets FileBorder to v if v is non-zero.
func (r *Radial) OverrideFileBorder(v MetricSpec) { overrideMetricSpec(&r.FileBorder, v) }

// OverrideFolderDiscSize sets FolderDiscSize to v if v is non-empty.
func (r *Radial) OverrideFolderDiscSize(v string) { overrideString(&r.FolderDiscSize, v) }

// OverrideFolderFill sets FolderFill to v if v is non-zero.
func (r *Radial) OverrideFolderFill(v MetricSpec) { overrideMetricSpec(&r.FolderFill, v) }

// OverrideFolderBorder sets FolderBorder to v if v is non-zero.
func (r *Radial) OverrideFolderBorder(v MetricSpec) { overrideMetricSpec(&r.FolderBorder, v) }

// OverrideLabels sets Labels to v if v is non-empty.
func (r *Radial) OverrideLabels(v string) { overrideString(&r.Labels, v) }

// OverrideGrain sets Grain to v if v is non-empty.
func (r *Radial) OverrideGrain(v string) { overrideString(&r.Grain, v) }
