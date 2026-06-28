package inks

// Option configures ink behaviour.
type Option func(*config)

type config struct {
	opacity float64
}

func defaultConfig() config {
	return config{
		opacity: 1.0,
	}
}

// WithOpacity sets the opacity applied when Dip() resolves a colour.
// Default is 1.0 (fully opaque). The opacity is applied to the alpha channel
// of the resolved colour.
func WithOpacity(opacity float64) Option {
	return func(c *config) {
		c.opacity = opacity
	}
}
