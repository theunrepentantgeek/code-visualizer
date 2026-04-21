package config

// Treemap holds persistent configuration for treemap visualizations.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Treemap struct {
	Fill              *string `yaml:"fill,omitempty"              json:"fill,omitempty"`
	FillPalette       *string `yaml:"fillPalette,omitempty"       json:"fillPalette,omitempty"`
	Border            *string `yaml:"border,omitempty"            json:"border,omitempty"`
	BorderPalette     *string `yaml:"borderPalette,omitempty"     json:"borderPalette,omitempty"`
	Legend            *string `yaml:"legend,omitempty"            json:"legend,omitempty"`
	LegendOrientation *string `yaml:"legendOrientation,omitempty" json:"legendOrientation,omitempty"`
}
