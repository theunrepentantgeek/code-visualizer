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
