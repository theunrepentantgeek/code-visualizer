package canvas

// InkOption configures ink behaviour.
type InkOption func(*inkConfig)

type inkConfig struct {
	opacity float64
}

func defaultInkConfig() inkConfig {
	return inkConfig{
		opacity: 1.0,
	}
}

// WithOpacity sets the opacity applied when Dip() resolves a colour.
// Default is 1.0 (fully opaque). The opacity is applied to the alpha channel
// of the resolved colour.
func WithOpacity(opacity float64) InkOption {
	return func(c *inkConfig) {
		c.opacity = opacity
	}
}
