package spiral

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

var (
	defaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	defaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
)

// Inks pairs the fill and border Ink instances for a spiral render pass.
// Alias for inks.ShapeInks so other viz packages share the same struct.
type Inks = inks.ShapeInks

// BuildInks creates fill and border inks from aggregated time-bucket data.
func BuildInks(
	buckets []TimeBucket,
	requested stages.RequestedMetrics,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	is := Inks{
		Fill:   inks.FixedInk(defaultFill),
		Border: inks.FixedInk(defaultBorder),
	}

	if fillMetric != "" {
		is.Fill = buildBucketInk(
			buckets, requested, fillMetric, fillPaletteName,
			func(b *TimeBucket) float64 { return b.FillValue },
			func(b *TimeBucket) string { return b.FillLabel },
			defaultFill,
		)
	}

	if borderMetric != "" {
		is.Border = buildBucketInk(
			buckets, requested, borderMetric, borderPaletteName,
			func(b *TimeBucket) float64 { return b.BorderValue },
			func(b *TimeBucket) string { return b.BorderLabel },
			defaultBorder,
		)
	}

	return is
}

// buildBucketInk creates an Ink from time-bucket-aggregated metric values.
// It takes accessor functions because spiral uses pre-aggregated time-bucket
// data, unlike treemap's per-file model.
func buildBucketInk(
	buckets []TimeBucket,
	requested stages.RequestedMetrics,
	m metric.Name,
	palName palette.PaletteName,
	numericFn func(*TimeBucket) float64,
	categoryFn func(*TimeBucket) string,
	fallback color.RGBA,
) inks.Ink {
	d, ok := requested.DescriptorFor(m)
	if !ok {
		return inks.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		values := make([]float64, len(buckets))
		for i := range buckets {
			values[i] = numericFn(&buckets[i])
		}

		return inks.NumericInk(m, values, pal)
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

	return inks.CategoricalInk(m, categories, pal)
}
