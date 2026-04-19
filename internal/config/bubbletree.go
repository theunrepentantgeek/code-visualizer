package config

// Bubbletree holds persistent configuration for bubble tree visualizations.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Bubbletree struct {
	Fill          *string `yaml:"fill,omitempty"          json:"fill,omitempty"`
	FillPalette   *string `yaml:"fillPalette,omitempty"   json:"fillPalette,omitempty"`
	Border        *string `yaml:"border,omitempty"        json:"border,omitempty"`
	BorderPalette *string `yaml:"borderPalette,omitempty" json:"borderPalette,omitempty"`
	Labels        *string `yaml:"labels,omitempty"        json:"labels,omitempty"`
	NoLegend      *bool   `yaml:"noLegend,omitempty"      json:"noLegend,omitempty"`
}
