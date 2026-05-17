package spiral

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

var (
	defaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	defaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
)

// Inks holds the Ink instances for a spiral render pass.
type Inks struct {
	Fill   canvas.Ink
	Border canvas.Ink
}

// BuildInks creates fill and border inks from aggregated time-bucket data.
func BuildInks(
	buckets []TimeBucket,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Fill:   canvas.FixedInk(defaultFill),
		Border: canvas.FixedInk(defaultBorder),
	}

	if fillMetric != "" {
		inks.Fill = buildBucketInk(
			buckets, fillMetric, fillPaletteName,
			func(b *TimeBucket) float64 { return b.FillValue },
			func(b *TimeBucket) string { return b.FillLabel },
			defaultFill,
		)
	}

	if borderMetric != "" {
		inks.Border = buildBucketInk(
			buckets, borderMetric, borderPaletteName,
			func(b *TimeBucket) float64 { return b.BorderValue },
			func(b *TimeBucket) string { return b.BorderLabel },
			defaultBorder,
		)
	}

	return inks
}

// buildBucketInk creates an Ink from time-bucket-aggregated metric values.
// It takes accessor functions because spiral uses pre-aggregated time-bucket
// data, unlike treemap's per-file model.
func buildBucketInk(
	buckets []TimeBucket,
	m metric.Name,
	palName palette.PaletteName,
	numericFn func(*TimeBucket) float64,
	categoryFn func(*TimeBucket) string,
	fallback color.RGBA,
) canvas.Ink {
	d, ok := provider.GetDescriptor(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		values := make([]float64, len(buckets))
		for i := range buckets {
			values[i] = numericFn(&buckets[i])
		}

		return canvas.NumericInk(m, values, pal)
	}

	seen := map[string]bool{}

	var categories []string

	for i := range buckets {
		cat := categoryFn(&buckets[i])
		if cat != "" && !seen[cat] {
			seen[cat] = true
			categories = append(categories, cat)
		}
	}

	return canvas.CategoricalInk(m, categories, pal)
}
