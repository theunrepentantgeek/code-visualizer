package config

// Footer holds configuration for the attribution footer rendered at the bottom
// of each generated image.
type Footer struct {
	Text   *string `yaml:"text,omitempty"   json:"text,omitempty"`
	Hidden *bool   `yaml:"hidden,omitempty" json:"hidden,omitempty"`
}

// ShowFooter reports whether the footer should be rendered.
func (f *Footer) ShowFooter() bool {
	if f == nil {
		return false
	}

	if f.Hidden != nil && *f.Hidden {
		return false
	}

	if f.Text == nil || *f.Text == "" {
		return false
	}

	return true
}

// OverrideText sets Text to v if v is non-empty.
func (f *Footer) OverrideText(v string) { overrideString(&f.Text, v) }

// OverrideHidden sets Hidden to true when v is true.
func (f *Footer) OverrideHidden(v bool) {
	f.Hidden = &v
}
