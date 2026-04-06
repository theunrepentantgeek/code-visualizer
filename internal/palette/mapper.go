package palette

import (
	"image/color"
	"log/slog"
)

// MapNumericToColour maps a bucket index (from BucketBoundaries.BucketIndex) to a palette colour.
// bucketIdx is 0-based, numBuckets is the total number of buckets (len(Boundaries)+1).
func MapNumericToColour(bucketIdx, numBuckets int, p ColourPalette) color.RGBA {
	if len(p.Colours) == 0 {
		return color.RGBA{A: 255}
	}

	if numBuckets <= 1 {
		mid := len(p.Colours) / 2

		return p.Colours[mid]
	}

	// Scale bucket range (0..numBuckets-1) to palette range (0..len(Colours)-1)
	paletteIdx := bucketIdx * (len(p.Colours) - 1) / (numBuckets - 1)
	if paletteIdx >= len(p.Colours) {
		paletteIdx = len(p.Colours) - 1
	}

	return p.Colours[paletteIdx]
}

// CategoricalMapper maps string values to palette colours.
type CategoricalMapper struct {
	mapping map[string]color.RGBA
}

// NewCategoricalMapper creates a mapper that assigns each distinct value a palette colour.
// If there are more values than colours, colours wrap around with a warning.
func NewCategoricalMapper(values []string, p ColourPalette) *CategoricalMapper {
	if len(p.Colours) == 0 {
		// Fallback: generate a single grey colour
		mapping := make(map[string]color.RGBA, len(values))
		for _, v := range values {
			mapping[v] = color.RGBA{R: 128, G: 128, B: 128, A: 255}
		}

		return &CategoricalMapper{mapping: mapping}
	}

	if len(values) > len(p.Colours) {
		slog.Warn("distinct values exceed palette capacity; some values will share colours",
			"values", len(values), "palette_capacity", len(p.Colours))
	}

	mapping := make(map[string]color.RGBA, len(values))
	for i, v := range values {
		idx := i % len(p.Colours)
		mapping[v] = p.Colours[idx]
	}

	return &CategoricalMapper{mapping: mapping}
}

// Map returns the colour for the given value.
func (m *CategoricalMapper) Map(value string) color.RGBA {
	if c, ok := m.mapping[value]; ok {
		return c
	}

	return color.RGBA{R: 128, G: 128, B: 128, A: 255}
}
