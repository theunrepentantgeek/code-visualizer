package spiral

import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

var (
	trackColour = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	bgColour    = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	trackWidth    = 1.0
	trackMinSteps = 500
)

// RenderToCanvas builds a Canvas from a spiral layout and time buckets.
func RenderToCanvas(
	layout SpiralLayout,
	buckets []TimeBucket,
	width, height int,
	is Inks,
	triangles []surface.Triangle,
	surfaceInk inks.Ink,
	discLabels []canvas.BlockLabel,
	format canvas.ImageFormat,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addBackground(cv, width, height)
	addSurface(cv, triangles, surfaceInk)
	addTrack(cv, layout)
	addDiscs(cv, layout.Nodes, buckets, is)
	for _, label := range discLabels {
		cv.AddBlockLabel(canvas.LayerOverlay, label, format)
	}

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

// addSurface adds interpolated metric triangles behind the spiral foreground.
func addSurface(cv *canvas.Canvas, triangles []surface.Triangle, surfaceInk inks.Ink) {
	if len(triangles) == 0 || surfaceInk == nil {
		return
	}

	if surfaceInk.Info().Kind == inks.KindNumeric {
		addBandedSurface(cv, triangles, surfaceInk, inks.NumericBreakpoints(surfaceInk))

		return
	}

	addFlatSurface(cv, triangles, surfaceInk)
}

func addFlatSurface(cv *canvas.Canvas, triangles []surface.Triangle, surfaceInk inks.Ink) {
	loopsByColour := make(map[color.RGBA][][]canvas.Position)

	for _, triangle := range triangles {
		fill := metricValue(triangle.Value, "", surfaceInk)
		colour := surfaceInk.Dip(fill)
		loopsByColour[colour] = append(
			loopsByColour[colour],
			surfacePolygonPoints(triangle.Points[:]),
		)
	}

	addSurfaceFillPaths(cv, loopsByColour)
}

func addBandedSurface(
	cv *canvas.Canvas,
	triangles []surface.Triangle,
	surfaceInk inks.Ink,
	breakpoints []float64,
) {
	loopsByColour := make(map[color.RGBA][][]canvas.Position)

	for _, triangle := range triangles {
		fragments := surface.SubdivideTriangle(triangle, breakpoints)
		if fragments == nil {
			continue
		}

		for _, fragment := range fragments {
			fill := metricValue(fragment.Value, "", surfaceInk)
			colour := surfaceInk.Dip(fill)
			loopsByColour[colour] = append(
				loopsByColour[colour],
				surfacePolygonPoints(fragment.Points),
			)
		}
	}

	addSurfaceFillPaths(cv, loopsByColour)
}

func addSurfaceFillPaths(cv *canvas.Canvas, loopsByColour map[color.RGBA][][]canvas.Position) {
	colours := make([]color.RGBA, 0, len(loopsByColour))
	for colour := range loopsByColour {
		colours = append(colours, colour)
	}

	slices.SortFunc(colours, func(left, right color.RGBA) int {
		return cmp.Compare(rgbaKey(left), rgbaKey(right))
	})

	for _, colour := range colours {
		cv.AddFilledPath(canvas.LayerSurface, canvas.FilledPath{
			Loops: loopsByColour[colour],
			Fill:  colour,
		})
	}
}

func rgbaKey(colour color.RGBA) uint32 {
	return uint32(colour.R)<<24 | uint32(colour.G)<<16 | uint32(colour.B)<<8 | uint32(colour.A)
}

func surfacePolygonPoints(points []surface.Point) []canvas.Position {
	positions := make([]canvas.Position, len(points))
	for index, point := range points {
		positions[index] = canvas.Position{X: point.X, Y: point.Y}
	}

	return positions
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
