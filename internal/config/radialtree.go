package config

// Radial holds persistent configuration for radial tree visualizations.
// All fields are pointers: nil means not configured, non-nil means explicitly set.
type Radial struct {
	Fill              *string `yaml:"fill,omitempty"              json:"fill,omitempty"`
	FillPalette       *string `yaml:"fillPalette,omitempty"       json:"fillPalette,omitempty"`
	Border            *string `yaml:"border,omitempty"            json:"border,omitempty"`
	BorderPalette     *string `yaml:"borderPalette,omitempty"     json:"borderPalette,omitempty"`
	Labels            *string `yaml:"labels,omitempty"            json:"labels,omitempty"`
	Legend            *string `yaml:"legend,omitempty"            json:"legend,omitempty"`
	LegendOrientation *string `yaml:"legendOrientation,omitempty" json:"legendOrientation,omitempty"`
}
