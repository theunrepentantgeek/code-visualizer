package viz

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// ColourEncoding identifies the metric and palette used to encode a visual
// channel as colour.
type ColourEncoding struct {
	Metric  metric.Name
	Palette palette.PaletteName
}

// IsSet reports whether the encoding selects a metric.
func (e ColourEncoding) IsSet() bool {
	return e.Metric != ""
}
