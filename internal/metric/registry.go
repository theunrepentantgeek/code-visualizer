package metric

import "github.com/bevan/code-visualizer/internal/palette"

var metricDefaultPalette = map[MetricName]palette.PaletteName{
	FileSize:      palette.Neutral,
	FileLines:     palette.Neutral,
	FileAge:       palette.Temperature,
	FileFreshness: palette.Temperature,
	AuthorCount:   palette.GoodBad,
	FileType:      palette.Categorization,
}

// DefaultPaletteFor returns the default palette for a given metric.
// The second return value is false if the metric is unknown.
func DefaultPaletteFor(m MetricName) (palette.PaletteName, bool) {
	p, ok := metricDefaultPalette[m]

	return p, ok
}
