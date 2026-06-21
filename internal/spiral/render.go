package spiral

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

var (
	trackColour = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	labelColour = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bgColour    = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	trackWidth    = 1.0
	labelGap      = 4.0
	trackMinSteps = 500
)

// RenderToCanvas builds a Canvas from a spiral layout and time buckets.
func RenderToCanvas(
	layout SpiralLayout,
	buckets []TimeBucket,
	width, height int,
	is Inks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addBackground(cv, width, height)
	addTrack(cv, layout)
	addDiscs(cv, layout.Nodes, buckets, is)
	addLabels(cv, layout.Nodes)

	return cv
}

// addBackground adds the white background rectangle.
func addBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(bgColour),
			Border:      inks.FixedInk(bgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec:  bgSpec,
		W:     float64(width),
		H:     float64(height),
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}

// addTrack adds the faint guide curve as a Path on the Structure layer.
func addTrack(cv *canvas.Canvas, layout SpiralLayout) {
	if len(layout.Nodes) < 2 {
		return
	}

	steps := trackSteps(len(layout.Nodes))
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
		Stroke:      inks.FixedInk(trackColour),
		StrokeWidth: trackWidth,
	}

	cv.AddPath(canvas.LayerStructure, canvas.Path{
		Spec:   trackSpec,
		Points: points,
	})
}

// addDiscs adds filled circles with borders for each active node.
func addDiscs(
	cv *canvas.Canvas,
	nodes []SpiralNode,
	buckets []TimeBucket,
	is Inks,
) {
	// Pre-allocate the two spec variants (borderWidth is either 2.0 or 3.0)
	// so they are not re-created for every disc in the loop.
	smallSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        is.Fill,
			Border:      is.Border,
			BorderWidth: 2.0,
		},
	}
	largeSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        is.Fill,
			Border:      is.Border,
			BorderWidth: 3.0,
		},
	}

	for i, n := range nodes {
		if n.DiscRadius <= 0 {
			continue
		}

		fillMV := metricValue(buckets[i].FillValue, buckets[i].FillLabel, is.Fill)
		borderMV := metricValue(buckets[i].BorderValue, buckets[i].BorderLabel, is.Border)

		spec := smallSpec
		if borderWidth(n.DiscRadius) == 3.0 {
			spec = largeSpec
		}

		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   spec,
			X:      n.X,
			Y:      n.Y,
			Radius: n.DiscRadius,
			Angle:  n.Angle,
			Fill:   fillMV,
			Border: borderMV,
		})
	}
}

// addLabels adds rotated text labels tangent to the spiral.
// Pre-allocates a shared labelInk to avoid recreating it for every label.
func addLabels(cv *canvas.Canvas, nodes []SpiralNode) {
	labelInk := inks.FixedInk(labelColour)

	for _, n := range nodes {
		if !n.ShowLabel || n.Label == "" {
			continue
		}

		addLabel(cv, n, labelInk)
	}
}

// addLabel adds a single rotated label for a spiral node.
func addLabel(cv *canvas.Canvas, n SpiralNode, labelInk inks.Ink) {
	labelR := n.DiscRadius + labelGap
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
		Ink:      labelInk,
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

// metricValue builds a MetricValue from time-bucket data for the given ink.
func metricValue(numericVal float64, categoryVal string, ink inks.Ink) inks.MetricValue {
	info := ink.Info()

	switch info.Kind {
	case inks.KindNumeric:
		return inks.MeasureValue(numericVal)
	case inks.KindCategorical:
		return inks.CategoryValue(categoryVal)
	default:
		return inks.MetricValue{}
	}
}

// borderWidth returns the border stroke width for a spiral disc.
func borderWidth(discRadius float64) float64 {
	if discRadius < 8 {
		return 2.0
	}

	return 3.0
}

// trackSteps returns the number of interpolation steps for the track curve.
func trackSteps(nodeCount int) int {
	steps := 3 * nodeCount
	if steps < trackMinSteps {
		return trackMinSteps
	}

	return steps
}
