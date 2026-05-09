package canvas

// MappingStrategy controls how numeric metric values are mapped to palette colours.
type MappingStrategy int

const (
	// Quantile uses equal-count buckets (current default behaviour).
	Quantile MappingStrategy = iota
	// Linear uses evenly spaced buckets across the min-max range.
	Linear
	// Logarithmic uses log-scale spacing.
	Logarithmic
)

// InkOption configures ink behaviour.
type InkOption func(*inkConfig)

type inkConfig struct {
	strategy MappingStrategy
	opacity  float64
}

func defaultInkConfig() inkConfig {
	return inkConfig{
		strategy: Quantile,
		opacity:  1.0,
	}
}

// WithMapping sets the mapping strategy for numeric inks.
func WithMapping(strategy MappingStrategy) InkOption {
	return func(c *inkConfig) {
		c.strategy = strategy
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
