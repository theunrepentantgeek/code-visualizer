package main

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

var (
	spiralDefaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	spiralDefaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	spiralTrackColour   = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	spiralLabelColour   = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	spiralBgColour      = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	spiralTrackWidth    = 1.0
	spiralLabelGap      = 4.0
	spiralTrackMinSteps = 500
)

// spiralInks holds the Ink instances for a spiral render pass.
type spiralInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildSpiralInks creates fill and border inks from aggregated time-bucket data.
func buildSpiralInks(
	buckets []spiral.TimeBucket,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) spiralInks {
	inks := spiralInks{
		fill:   canvas.FixedInk(spiralDefaultFill),
		border: canvas.FixedInk(spiralDefaultBorder),
	}

	if fillMetric != "" {
		inks.fill = buildBucketInk(
			buckets, fillMetric, fillPaletteName,
			func(b *spiral.TimeBucket) float64 { return b.FillValue },
			func(b *spiral.TimeBucket) string { return b.FillLabel },
			spiralDefaultFill,
		)
	}

	if borderMetric != "" {
		inks.border = buildBucketInk(
			buckets, borderMetric, borderPaletteName,
			func(b *spiral.TimeBucket) float64 { return b.BorderValue },
			func(b *spiral.TimeBucket) string { return b.BorderLabel },
			spiralDefaultBorder,
		)
	}

	return inks
}

// buildBucketInk creates an Ink from time-bucket-aggregated metric values.
// It takes accessor functions because spiral uses pre-aggregated time-bucket
// data, unlike treemap's per-file model.
func buildBucketInk(
	buckets []spiral.TimeBucket,
	m metric.Name,
	palName palette.PaletteName,
	numericFn func(*spiral.TimeBucket) float64,
	categoryFn func(*spiral.TimeBucket) string,
	fallback color.RGBA,
) canvas.Ink {
	p, ok := provider.Get(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
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

// renderSpiralToCanvas builds a Canvas from a spiral layout and time buckets.
func renderSpiralToCanvas(
	layout spiral.SpiralLayout,
	buckets []spiral.TimeBucket,
	width, height int,
	inks spiralInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addSpiralBackground(cv, width, height)
	addSpiralTrack(cv, layout)
	addSpiralDiscs(cv, layout.Nodes, buckets, inks)
	addSpiralLabels(cv, layout.Nodes)

	return cv
}

// addSpiralBackground adds the white background rectangle.
func addSpiralBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(spiralBgColour),
			Border:      canvas.FixedInk(spiralBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    float64(width), H: float64(height),
	})
}

// addSpiralTrack adds the faint guide curve as a Path on the Structure layer.
func addSpiralTrack(cv *canvas.Canvas, layout spiral.SpiralLayout) {
	if len(layout.Nodes) < 2 {
		return
	}

	steps := spiralTrackSteps(len(layout.Nodes))
	points := make([]canvas.Position, steps)

	for i := range steps {
		t := float64(i) / float64(steps-1)
		theta := t * layout.MaxTheta
		r := layout.A + layout.B*theta
		points[i] = canvas.Position{
			X: layout.CX + r*math.Sin(theta),
			Y: layout.CY - r*math.Cos(theta),
		}
	}

	trackSpec := &canvas.LineSpec{
		Stroke:      canvas.FixedInk(spiralTrackColour),
		StrokeWidth: spiralTrackWidth,
	}

	cv.AddPath(canvas.LayerStructure, canvas.Path{
		Spec:   trackSpec,
		Points: points,
	})
}

// addSpiralDiscs adds filled circles with borders for each active node.
func addSpiralDiscs(
	cv *canvas.Canvas,
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	inks spiralInks,
) {
	for i, n := range nodes {
		if n.DiscRadius <= 0 {
			continue
		}

		fillMV := spiralMetricValue(buckets[i].FillValue, buckets[i].FillLabel, inks.fill)
		borderMV := spiralMetricValue(buckets[i].BorderValue, buckets[i].BorderLabel, inks.border)

		discSpec := &canvas.DiscSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        inks.fill,
				Border:      inks.border,
				BorderWidth: spiralBorderWidth(n.DiscRadius),
			},
		}

		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   discSpec,
			X:      n.X,
			Y:      n.Y,
			Radius: n.DiscRadius,
			Angle:  n.Angle,
			Fill:   fillMV,
			Border: borderMV,
		})
	}
}

// addSpiralLabels adds rotated text labels tangent to the spiral.
func addSpiralLabels(cv *canvas.Canvas, nodes []spiral.SpiralNode) {
	for _, n := range nodes {
		if !n.ShowLabel || n.Label == "" {
			continue
		}

		addSpiralLabel(cv, n)
	}
}

// addSpiralLabel adds a single rotated label for a spiral node.
func addSpiralLabel(cv *canvas.Canvas, n spiral.SpiralNode) {
	labelR := n.DiscRadius + spiralLabelGap
	lx := n.X + labelR*math.Sin(n.Angle)
	ly := n.Y - labelR*math.Cos(n.Angle)

	norm := math.Mod(n.Angle, 2*math.Pi)
	if norm < 0 {
		norm += 2 * math.Pi
	}

	var anchor canvas.TextAnchor

	var rotation float64

	if norm <= math.Pi {
		anchor = canvas.AnchorStart
		rotation = n.Angle
	} else {
		anchor = canvas.AnchorEnd
		rotation = n.Angle + math.Pi
	}

	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(spiralLabelColour),
		FontSize: 0,
		Anchor:   anchor,
		Rotation: rotation,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       lx,
		Y:       ly,
		Content: n.Label,
	})
}

// spiralMetricValue builds a MetricValue from time-bucket data for the given ink.
func spiralMetricValue(numericVal float64, categoryVal string, ink canvas.Ink) canvas.MetricValue {
	info := ink.Info()

	switch info.Kind {
	case canvas.InkNumeric:
		return canvas.MeasureValue(numericVal)
	case canvas.InkCategorical:
		return canvas.CategoryValue(categoryVal)
	default:
		return canvas.MetricValue{}
	}
}

// spiralBorderWidth returns the border stroke width for a spiral disc.
func spiralBorderWidth(discRadius float64) float64 {
	if discRadius < 8 {
		return 2.0
	}

	return 3.0
}

// spiralTrackSteps returns the number of interpolation steps for the track curve.
func spiralTrackSteps(nodeCount int) int {
	steps := 3 * nodeCount
	if steps < spiralTrackMinSteps {
		return spiralTrackMinSteps
	}

	return steps
}
