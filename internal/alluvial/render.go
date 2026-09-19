package alluvial

import (
	"hash/fnv"
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var (
	alluvialBackground = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	alluvialGuide      = color.RGBA{R: 0xD0, G: 0xD0, B: 0xD0, A: 0xFF}
	alluvialLabel      = color.RGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xFF}
)

// RenderToCanvas draws readable release columns and the filled alluvial paths.
func RenderToCanvas(layout Layout, width, height int) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)
	addAlluvialBackground(cv, width, height)
	addAlluvialColumns(cv, layout)

	for _, flow := range layout.Flows {
		if !validFlow(flow) {
			continue
		}

		cv.AddFilledPath(canvas.LayerContent, canvas.FilledPath{
			Loops: [][]geometry.Point{{
				{X: flow.FromX, Y: flow.FromTop},
				{X: flow.ToX, Y: flow.ToTop},
				{X: flow.ToX, Y: flow.ToBottom},
				{X: flow.FromX, Y: flow.FromBottom},
			}},
			Fill: flowColour(flow.Path),
		})
	}

	return cv
}

func addAlluvialBackground(cv *canvas.Canvas, width, height int) {
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(alluvialBackground),
			Border:      inks.FixedInk(alluvialBackground),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: spec,
		Bounds: geometry.RectFromPositionSize(
			geometry.OriginPoint,
			geometry.NewSize(float64(width), float64(height)),
		),
		Focus: canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
	})
}

func addAlluvialColumns(cv *canvas.Canvas, layout Layout) {
	lineSpec := &canvas.LineSpec{Stroke: inks.FixedInk(alluvialGuide), StrokeWidth: 1}
	textSpec := &canvas.TextSpec{Ink: inks.FixedInk(alluvialLabel), FontSize: 13, Anchor: canvas.AnchorMiddle}

	for _, column := range layout.Columns {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.Point{X: column.X, Y: layout.Top},
			To:   geometry.Point{X: column.X, Y: layout.Bottom},
		})
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     textSpec,
			Position: geometry.Point{X: column.X, Y: layout.Top - 12},
			Content:  column.Reference,
		})
	}
}

func validFlow(flow Flow) bool {
	if flow.FromX >= flow.ToX {
		return false
	}

	if flow.FromBottom < flow.FromTop || flow.ToBottom < flow.ToTop {
		return false
	}

	if flow.FromBottom == flow.FromTop && flow.ToBottom == flow.ToTop {
		return false
	}

	for _, value := range []float64{
		flow.FromX, flow.ToX, flow.FromTop, flow.FromBottom, flow.ToTop, flow.ToBottom,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}

	return true
}

func flowColour(path string) color.RGBA {
	colours := palette.GetPalette(palette.Categorization).Colours
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(path))

	return colours[int(hasher.Sum32())%len(colours)]
}
