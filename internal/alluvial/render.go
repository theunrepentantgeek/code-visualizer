package alluvial

import (
	"fmt"
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

var (
	alluvialBackground = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	alluvialGuide      = color.RGBA{R: 0xD0, G: 0xD0, B: 0xD0, A: 0xFF}
	alluvialLabel      = color.RGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xFF}
)

// RenderToCanvas draws readable release columns and the filled alluvial paths.
func RenderToCanvas(layout Layout, width, height int, fillInk inks.Ink, labelFillMetric metric.Name) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)
	addAlluvialBackground(cv, width, height)
	addAlluvialColumns(cv, layout)

	leftExtension, rightExtension := alluvialEdgeBandExtensions(layout.Columns, labelFillMetric)
	addAlluvialEdgeBands(cv, layout.Columns, leftExtension, rightExtension, fillInk)

	for _, flow := range layout.Flows {
		if !validFlow(flow) {
			continue
		}

		cv.AddFilledPath(canvas.LayerContent, canvas.FilledPath{
			Loops: [][]geometry.Point{sweptFlowPoints(flow)},
			Fill:  alluvialFillInk(fillInk, flow.HasFillValue).Dip(inks.MeasureValue(flow.FillValue)),
		})
	}

	addAlluvialBandLabels(cv, layout, labelFillMetric, fillInk)

	return cv
}

func sweptFlowPoints(flow Flow) []geometry.Point {
	const segments = 12

	points := make([]geometry.Point, 0, 2*segments+2)

	for index := range segments + 1 {
		points = append(points, cubicPoint(
			geometry.Point{X: flow.FromX, Y: flow.FromTop},
			geometry.Point{X: flow.FromX + (flow.ToX-flow.FromX)/3, Y: flow.FromTop},
			geometry.Point{X: flow.ToX - (flow.ToX-flow.FromX)/3, Y: flow.ToTop},
			geometry.Point{X: flow.ToX, Y: flow.ToTop},
			float64(index)/segments,
		))
	}

	for index := segments; index >= 0; index-- {
		points = append(points, cubicPoint(
			geometry.Point{X: flow.FromX, Y: flow.FromBottom},
			geometry.Point{X: flow.FromX + (flow.ToX-flow.FromX)/3, Y: flow.FromBottom},
			geometry.Point{X: flow.ToX - (flow.ToX-flow.FromX)/3, Y: flow.ToBottom},
			geometry.Point{X: flow.ToX, Y: flow.ToBottom},
			float64(index)/segments,
		))
	}

	return points
}

func cubicPoint(start, controlOne, controlTwo, end geometry.Point, t float64) geometry.Point {
	inverse := 1 - t

	return geometry.Point{
		X: inverse*inverse*inverse*start.X + 3*inverse*inverse*t*controlOne.X +
			3*inverse*t*t*controlTwo.X + t*t*t*end.X,
		Y: inverse*inverse*inverse*start.Y + 3*inverse*inverse*t*controlOne.Y +
			3*inverse*t*t*controlTwo.Y + t*t*t*end.Y,
	}
}

func addAlluvialBandLabels(
	cv *canvas.Canvas,
	layout Layout,
	labelFillMetric metric.Name,
	fillInk inks.Ink,
) {
	for _, column := range layout.Columns {
		for _, band := range column.Bands {
			addAlluvialBandLabel(cv, column.X, band, labelFillMetric, fillInk)
		}
	}
}

func alluvialEdgeBandExtensions(
	columns []ColumnLayout,
	labelFillMetric metric.Name,
) (left, right float64) {
	if len(columns) < 2 {
		return 0, 0
	}

	return alluvialColumnLabelExtension(columns[0], labelFillMetric),
		alluvialColumnLabelExtension(columns[len(columns)-1], labelFillMetric)
}

func alluvialColumnLabelExtension(
	column ColumnLayout,
	labelFillMetric metric.Name,
) float64 {
	maximumWidth := 0.0

	for _, band := range column.Bands {
		lines := alluvialBandLabelLines(band, labelFillMetric)

		fontSize, ok := alluvialBandLabelFontSize(band, len(lines))
		if !ok {
			continue
		}

		widths, _ := textlayout.MeasureStrings(lines, fontSize)
		for _, width := range widths {
			maximumWidth = max(maximumWidth, width)
		}
	}

	const edgeInset = 2.0

	return maximumWidth/2 + edgeInset
}

func addAlluvialEdgeBands(
	cv *canvas.Canvas,
	columns []ColumnLayout,
	leftExtension, rightExtension float64,
	fillInk inks.Ink,
) {
	if len(columns) < 2 {
		return
	}

	addAlluvialEdgeColumnBands(cv, columns[0], columns[0].X-leftExtension, columns[0].X, fillInk)

	last := columns[len(columns)-1]
	addAlluvialEdgeColumnBands(cv, last, last.X, last.X+rightExtension, fillInk)
}

func addAlluvialEdgeColumnBands(
	cv *canvas.Canvas,
	column ColumnLayout,
	left, right float64,
	fillInk inks.Ink,
) {
	for _, band := range column.Bands {
		if band.Bottom <= band.Top {
			continue
		}

		cv.AddFilledPath(canvas.LayerContent, canvas.FilledPath{
			Loops: [][]geometry.Point{{
				{X: left, Y: band.Top},
				{X: right, Y: band.Top},
				{X: right, Y: band.Bottom},
				{X: left, Y: band.Bottom},
			}},
			Fill: alluvialFillInk(fillInk, band.HasFillValue).Dip(inks.MeasureValue(band.FillValue)),
		})
	}
}

func addAlluvialBandLabel(
	cv *canvas.Canvas,
	x float64,
	band Band,
	labelFillMetric metric.Name,
	fillInk inks.Ink,
) {
	lines := alluvialBandLabelLines(band, labelFillMetric)

	fontSize, ok := alluvialBandLabelFontSize(band, len(lines))
	if !ok {
		return
	}

	center := (band.Top + band.Bottom) / 2
	labelColour := canvas.TextColourFor(
		alluvialFillInk(fillInk, band.HasFillValue).Dip(inks.MeasureValue(band.FillValue)),
	)
	spec := &canvas.TextSpec{
		Ink:      inks.FixedInk(labelColour),
		FontSize: fontSize,
		Anchor:   canvas.AnchorMiddle,
	}

	start := center - float64(len(lines)-1)*fontSize/2
	for index, line := range lines {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     spec,
			Position: geometry.Point{X: x, Y: start + float64(index)*fontSize},
			Content:  line,
		})
	}
}

func alluvialBandLabelLines(band Band, labelFillMetric metric.Name) []string {
	lines := []string{band.Path, fmt.Sprintf("%g", band.Width)}
	if labelFillMetric != "" {
		if band.HasFillValue {
			lines = append(lines, fmt.Sprintf("%g", band.FillValue))
		} else {
			lines = append(lines, "-")
		}
	}

	return lines
}

func alluvialFillInk(fillInk inks.Ink, hasFillValue bool) inks.Ink {
	if !hasFillValue {
		return inks.FixedInk(alluvialGuide)
	}

	return fillInk
}

func alluvialBandLabelFontSize(band Band, lineCount int) (float64, bool) {
	fontSize := min(13, (band.Bottom-band.Top)/float64(lineCount+1))
	if fontSize < 8 {
		return 0, false
	}

	return fontSize, true
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
