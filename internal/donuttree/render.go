package donuttree

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const donutSectorBorderWidth = 1.0

var (
	donutBackgroundColour = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	donutAnchorFillColour = color.RGBA{R: 0xE5, G: 0xE7, B: 0xEB, A: 0xFF}
	donutAnchorBorder     = color.RGBA{R: 0x4B, G: 0x55, B: 0x63, A: 0xFF}
	donutLabelColour      = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	transparentInk        = inks.FixedInk(color.RGBA{})
)

// RenderToCanvas renders directory sectors and the fixed root anchor.
func RenderToCanvas(
	layout LayoutResult,
	root *model.Directory,
	width, height int,
	is Inks,
	labelMetrics LabelMetrics,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)
	addDonutBackground(cv, width, height)
	addRootAnchor(cv, layout, root)
	addDonutSectors(cv, layout.Children, layout.Center, is, labelMetrics)

	return cv
}

func addDonutBackground(cv *canvas.Canvas, width, height int) {
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(donutBackgroundColour),
			Border:      inks.FixedInk(donutBackgroundColour),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: spec, W: float64(width), H: float64(height), Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}

func addRootAnchor(cv *canvas.Canvas, layout LayoutResult, root *model.Directory) {
	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(donutAnchorFillColour),
			Border:      inks.FixedInk(donutAnchorBorder),
			BorderWidth: 1,
		},
	}
	cv.AddDisc(canvas.LayerContent, canvas.Disc{
		Spec: discSpec, X: layout.Center.X, Y: layout.Center.Y, Radius: layout.AnchorRadius,
	})

	labelSpec := &canvas.TextSpec{
		Ink: inks.FixedInk(donutLabelColour), Anchor: canvas.AnchorMiddle, FontSize: 14,
	}

	rootName := layout.RootName
	if root != nil {
		rootName = root.Name
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec: labelSpec, X: layout.Center.X, Y: layout.Center.Y, Content: rootName,
	})
}

func addDonutSectors(
	cv *canvas.Canvas,
	nodes []DonutNode,
	center canvas.Position,
	is Inks,
	labelMetrics LabelMetrics,
) {
	fillSpec := &canvas.PolygonSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill: is.Fill, Border: is.Border,
		},
	}
	for _, node := range nodes {
		fillValue := inks.MetricValueForDirectory(node.Directory, is.Fill)
		cv.AddPolygon(canvas.LayerContent, canvas.Polygon{
			Spec:   fillSpec,
			Points: sectorPoints(node, center),
			Fill:   fillValue,
			Border: inks.MetricValueForDirectory(node.Directory, is.Border),
		})

		if is.HasBorderMetric {
			borderWidth := sectorBorderWidth(node)
			borderSpec := &canvas.PolygonSpec{
				ShapeStyle: canvas.ShapeStyle{
					Fill: transparentInk, Border: is.Border, BorderWidth: borderWidth,
				},
			}
			cv.AddPolygon(canvas.LayerContent, canvas.Polygon{
				Spec:   borderSpec,
				Points: insetSectorPoints(node, center, borderWidth),
				Border: inks.MetricValueForDirectory(node.Directory, is.Border),
			})
		}

		labelInk := inks.FixedInk(canvas.TextColourFor(is.Fill.Dip(fillValue)))
		addSectorLabel(cv, node, center, buildDirectoryLabel(node.Directory, labelMetrics), labelInk)
		addDonutSectors(cv, node.Children, center, is, labelMetrics)
	}
}

func sectorPoints(node DonutNode, center canvas.Position) []canvas.Position {
	steps := sectorSteps(node.SweepAngle)
	points := make([]canvas.Position, 0, 2*steps+3)

	for step := range steps + 1 {
		angle := node.StartAngle + node.SweepAngle*float64(step)/float64(steps)
		points = append(points, polarPosition(center, node.OuterRadius, angle))
	}

	for step := range steps + 1 {
		angle := node.EndAngle() - node.SweepAngle*float64(step)/float64(steps)
		points = append(points, polarPosition(center, node.InnerRadius, angle))
	}

	points = append(points, points[0])

	return points
}

func insetSectorPoints(node DonutNode, center canvas.Position, borderWidth float64) []canvas.Position {
	halfWidth := borderWidth / 2
	innerRadius := node.InnerRadius + halfWidth
	outerRadius := node.OuterRadius - halfWidth
	halfSweep := node.SweepAngle / 2
	maxInset := halfSweep * (1 - 1e-9)
	innerInset := radialEdgeInset(innerRadius, halfWidth, maxInset)
	outerInset := radialEdgeInset(outerRadius, halfWidth, maxInset)
	steps := sectorSteps(node.SweepAngle)
	points := make([]canvas.Position, 0, 2*steps+3)

	for step := range steps + 1 {
		angle := node.StartAngle + outerInset +
			(node.SweepAngle-2*outerInset)*float64(step)/float64(steps)
		points = append(points, polarPosition(center, outerRadius, angle))
	}

	for step := range steps + 1 {
		angle := node.EndAngle() - innerInset -
			(node.SweepAngle-2*innerInset)*float64(step)/float64(steps)
		points = append(points, polarPosition(center, innerRadius, angle))
	}

	points = append(points, points[0])

	return points
}

func sectorBorderWidth(node DonutNode) float64 {
	sine := math.Sin(node.SweepAngle / 2)
	if sine >= 1 {
		return donutSectorBorderWidth
	}

	maxWidth := 2 * node.InnerRadius * sine / (1 - sine)

	return math.Min(donutSectorBorderWidth, maxWidth)
}

func radialEdgeInset(radius, halfWidth, maxInset float64) float64 {
	if radius <= 0 || halfWidth <= 0 || maxInset <= 0 {
		return 0
	}

	return math.Min(math.Asin(math.Min(halfWidth/radius, 1)), maxInset)
}

func sectorSteps(sweepAngle float64) int {
	return max(2, int(math.Ceil(sweepAngle/(2*math.Pi)*64)))
}

func polarPosition(center canvas.Position, radius, angle float64) canvas.Position {
	return canvas.Position{
		X: center.X + radius*math.Cos(angle),
		Y: center.Y + radius*math.Sin(angle),
	}
}
