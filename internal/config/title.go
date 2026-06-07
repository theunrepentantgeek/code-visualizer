package config

// Title holds configuration for the title rendered at the top of each
// generated image.
type Title struct {
	Text   *string `yaml:"text,omitempty"   json:"text,omitempty"`
	Hidden *bool   `yaml:"hidden,omitempty" json:"hidden,omitempty"`
}

// ShowTitle reports whether the title should be rendered.
func (t *Title) ShowTitle() bool {
	if t == nil {
		return false
	}

	if t.Hidden != nil && *t.Hidden {
		return false
	}

	if t.Text == nil || *t.Text == "" {
		return false
	}

	return true
}

// OverrideText sets Text to v if v is non-empty.
func (t *Title) OverrideText(v string) { overrideString(&t.Text, v) }
