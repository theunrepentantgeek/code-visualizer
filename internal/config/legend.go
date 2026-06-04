package config

// Legend holds top-level configuration for the legend rendered on each
// generated image.
type Legend struct {
	Visible     *bool   `yaml:"visible,omitempty"     json:"visible,omitempty"`
	Orientation *string `yaml:"orientation,omitempty" json:"orientation,omitempty"`
	Position    *string `yaml:"position,omitempty"    json:"position,omitempty"`
}

// PositionStr returns the legend position string for rendering.
// Returns "none" when Visible is explicitly false, otherwise the
// configured position (or empty string to use the rendering default).
func (l *Legend) PositionStr() string {
	if l == nil {
		return ""
	}

	if l.Visible != nil && !*l.Visible {
		return "none"
	}

	if l.Position == nil {
		return ""
	}

	return *l.Position
}

// OrientationStr returns the legend orientation string, or empty string when
// not set (callers apply their own default).
func (l *Legend) OrientationStr() string {
	if l == nil || l.Orientation == nil {
		return ""
	}

	return *l.Orientation
}

// OverridePosition sets Position to v if v is non-empty.
func (l *Legend) OverridePosition(v string) { overrideString(&l.Position, v) }

// OverrideOrientation sets Orientation to v if v is non-empty.
func (l *Legend) OverrideOrientation(v string) { overrideString(&l.Orientation, v) }
