package donuttree

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

var (
	donutBackgroundColour = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	donutAnchorFillColour = color.RGBA{R: 0xE5, G: 0xE7, B: 0xEB, A: 0xFF}
	donutAnchorBorder     = color.RGBA{R: 0x4B, G: 0x55, B: 0x63, A: 0xFF}
	donutLabelColour      = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
)

// RenderToCanvas renders directory sectors and the fixed root anchor.
func RenderToCanvas(layout LayoutResult, root *model.Directory, width, height int, is Inks) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)
	addDonutBackground(cv, width, height)
	addRootAnchor(cv, layout, root)
	addDonutSectors(cv, layout.Children, layout.Center, is)

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

func addDonutSectors(cv *canvas.Canvas, nodes []DonutNode, center canvas.Position, is Inks) {
	borderWidth := 0.0
	if is.HasBorderMetric {
		borderWidth = 1
	}

	spec := &canvas.PolygonSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill: is.Fill, Border: is.Border, BorderWidth: borderWidth,
		},
	}

	labelInk := inks.FixedInk(donutLabelColour)

	for _, node := range nodes {
		cv.AddPolygon(canvas.LayerContent, canvas.Polygon{
			Spec:   spec,
			Points: sectorPoints(node, center),
			Fill:   inks.MetricValueForDirectory(node.Directory, is.Fill),
			Border: inks.MetricValueForDirectory(node.Directory, is.Border),
		})
		addSectorLabel(cv, node, center, buildDirectoryLabel(node.Directory, is.LabelMetrics), labelInk)
		addDonutSectors(cv, node.Children, center, is)
	}
}

func sectorPoints(node DonutNode, center canvas.Position) []canvas.Position {
	steps := max(2, int(math.Ceil(node.SweepAngle/(2*math.Pi)*64)))
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

func polarPosition(center canvas.Position, radius, angle float64) canvas.Position {
	return canvas.Position{
		X: center.X + radius*math.Cos(angle),
		Y: center.Y + radius*math.Sin(angle),
	}
}
