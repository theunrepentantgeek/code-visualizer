package scatter

import (
	"math"
	"unicode/utf8"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

const (
	scatterAxisStrokeWidth = 1.5
	scatterGridStrokeWidth = 1.0
	scatterBorderWidth     = 1.0
	scatterMetricBorder    = 2.0
	scatterLabelMinFont    = 8.0
	scatterLabelMaxFont    = 14.0
)

// RenderToCanvas converts a scatter layout into a populated canvas.
func RenderToCanvas(layout ScatterLayout, width, height int, is Inks) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)
	addScatterBackground(cv, width, height)
	addScatterStructure(cv, layout)
	addScatterPoints(cv, layout.Points, is)

	return cv
}

func addScatterBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(scatterBgColour),
			Border:      inks.FixedInk(scatterBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec:  bgSpec,
		W:     float64(width),
		H:     float64(height),
		Focus: canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
	})
}

func addScatterStructure(cv *canvas.Canvas, layout ScatterLayout) {
	addScatterPlotBorder(cv, layout.Plot)
	addScatterAxisGuides(cv, layout)
	addScatterAxisLabels(cv, layout)
}

func addScatterPlotBorder(cv *canvas.Canvas, plot PlotRect) {
	plotSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(scatterBgColour),
			Border:      inks.FixedInk(scatterAxisColour),
			BorderWidth: scatterAxisStrokeWidth,
		},
	}

	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  plotSpec,
		X:     plot.X,
		Y:     plot.Y,
		W:     plot.W,
		H:     plot.H,
		Focus: canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
	})
}

func addScatterAxisGuides(cv *canvas.Canvas, layout ScatterLayout) {
	lineSpec := &canvas.LineSpec{Stroke: inks.FixedInk(scatterGridColour), StrokeWidth: scatterGridStrokeWidth}
	for _, tick := range layout.XAxis.NumericTicks() {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.NewPoint(tick.Position, layout.Plot.Y),
			To:   geometry.NewPoint(tick.Position, layout.Plot.Y+layout.Plot.H),
		})
	}

	for _, tick := range layout.YAxis.NumericTicks() {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.NewPoint(layout.Plot.X, tick.Position),
			To:   geometry.NewPoint(layout.Plot.X+layout.Plot.W, tick.Position),
		})
	}

	xBands := layout.XAxis.CategoricalBands()
	for _, band := range xBands {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.NewPoint(band.Start, layout.Plot.Y),
			To:   geometry.NewPoint(band.Start, layout.Plot.Y+layout.Plot.H),
		})
	}

	if len(xBands) > 0 {
		last := xBands[len(xBands)-1]
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.NewPoint(last.End, layout.Plot.Y),
			To:   geometry.NewPoint(last.End, layout.Plot.Y+layout.Plot.H),
		})
	}

	yBands := layout.YAxis.CategoricalBands()
	for _, band := range yBands {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.NewPoint(layout.Plot.X, band.Start),
			To:   geometry.NewPoint(layout.Plot.X+layout.Plot.W, band.Start),
		})
	}

	if len(yBands) > 0 {
		last := yBands[len(yBands)-1]
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			From: geometry.NewPoint(layout.Plot.X, last.End),
			To:   geometry.NewPoint(layout.Plot.X+layout.Plot.W, last.End),
		})
	}
}

func addScatterAxisLabels(cv *canvas.Canvas, layout ScatterLayout) {
	labelInk := inks.FixedInk(scatterLabelColour)

	titleSpec := &canvas.TextSpec{Ink: labelInk, FontSize: 12, Anchor: canvas.AnchorMiddle}
	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:     titleSpec,
		Position: geometry.NewPoint(layout.Plot.X+layout.Plot.W/2, layout.Plot.Y+layout.Plot.H+56),
		Content:  layout.XAxis.Title,
	})

	yTitleSpec := &canvas.TextSpec{
		Ink:      labelInk,
		FontSize: 12,
		Anchor:   canvas.AnchorMiddle,
		Rotation: -math.Pi / 2,
	}
	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:     yTitleSpec,
		Position: geometry.NewPoint(layout.Plot.X-72, layout.Plot.Y+layout.Plot.H/2),
		Content:  layout.YAxis.Title,
	})

	tickSpec := &canvas.TextSpec{Ink: labelInk, FontSize: 10, Anchor: canvas.AnchorMiddle}
	for _, tick := range layout.XAxis.NumericTicks() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     tickSpec,
			Position: geometry.NewPoint(tick.Position, layout.Plot.Y+layout.Plot.H+18),
			Content:  tick.Label,
		})
	}

	for _, band := range layout.XAxis.CategoricalBands() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     tickSpec,
			Position: geometry.NewPoint(band.Center, layout.Plot.Y+layout.Plot.H+18),
			Content:  band.Label,
		})
	}

	yTickSpec := &canvas.TextSpec{Ink: labelInk, FontSize: 10, Anchor: canvas.AnchorEnd}
	for _, tick := range layout.YAxis.NumericTicks() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     yTickSpec,
			Position: geometry.NewPoint(layout.Plot.X-8, tick.Position),
			Content:  tick.Label,
		})
	}

	for _, band := range layout.YAxis.CategoricalBands() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     yTickSpec,
			Position: geometry.NewPoint(layout.Plot.X-8, band.Center),
			Content:  band.Label,
		})
	}
}

func addScatterPoints(cv *canvas.Canvas, points []ScatterPoint, is Inks) {
	borderWidth := scatterBorderWidth
	if is.HasBorderMetric {
		borderWidth = scatterMetricBorder
	}

	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{Fill: is.Fill, Border: is.Border, BorderWidth: borderWidth},
	}

	// Pre-allocate both possible label inks so per-point rendering avoids heap allocations.
	// canvas.TextColourFor returns exactly one of two values (black or white).
	darkLabelColour := canvas.TextColourFor(scatterBgColour)
	darkLabelInk := inks.FixedInk(darkLabelColour)
	lightLabelInk := inks.FixedInk(canvas.TextColourFor(darkLabelColour))

	for _, point := range points {
		fillValue := metricValueForPoint(point, is.Fill)
		borderValue := metricValueForPoint(point, is.Border)
		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   discSpec,
			X:      point.Position.X,
			Y:      point.Position.Y,
			Radius: point.Radius,
			Fill:   fillValue,
			Border: borderValue,
		})

		label, fontSize := scatterLabel(point.Label, point.Radius)
		labelColour := canvas.TextColourFor(is.Fill.Dip(fillValue))

		var labelInk inks.Ink
		if labelColour == darkLabelColour {
			labelInk = darkLabelInk
		} else {
			labelInk = lightLabelInk
		}

		labelSpec := &canvas.TextSpec{
			Ink:      labelInk,
			FontSize: fontSize,
			Anchor:   canvas.AnchorMiddle,
		}
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     labelSpec,
			Position: point.Position,
			Content:  label,
		})
	}
}

func metricValueForPoint(point ScatterPoint, ink inks.Ink) inks.MetricValue {
	if point.Directory != nil {
		return inks.MetricValueForDirectory(point.Directory, ink)
	}

	return inks.MetricValueForFile(point.File, ink)
}

func scatterLabel(label string, radius float64) (string, float64) {
	fontSize := min(scatterLabelMaxFont, max(scatterLabelMinFont, radius*0.6))

	maxChars := int((2 * radius * 0.85) / (fontSize * 0.6))
	if maxChars <= 0 {
		maxChars = 1
	}

	if utf8.RuneCountInString(label) <= maxChars {
		return label, fontSize
	}

	runes := []rune(label)

	if maxChars == 1 {
		return string(runes[:1]), fontSize
	}

	return string(runes[:maxChars-1]) + "…", fontSize
}
