package scatter

import (
	"math"
	"unicode/utf8"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
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
func RenderToCanvas(layout ScatterLayout, width, height int, inks Inks) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)
	addScatterBackground(cv, width, height)
	addScatterStructure(cv, layout)
	addScatterPoints(cv, layout.Points, inks)

	return cv
}

func addScatterBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        pkginks.FixedInk(scatterBgColour),
			Border:      pkginks.FixedInk(scatterBgColour),
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

func addScatterStructure(cv *canvas.Canvas, layout ScatterLayout) {
	addScatterPlotBorder(cv, layout.Plot)
	addScatterAxisGuides(cv, layout)
	addScatterAxisLabels(cv, layout)
}

func addScatterPlotBorder(cv *canvas.Canvas, plot PlotRect) {
	plotSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        pkginks.FixedInk(scatterBgColour),
			Border:      pkginks.FixedInk(scatterAxisColour),
			BorderWidth: scatterAxisStrokeWidth,
		},
	}

	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  plotSpec,
		X:     plot.X,
		Y:     plot.Y,
		W:     plot.W,
		H:     plot.H,
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}

func addScatterAxisGuides(cv *canvas.Canvas, layout ScatterLayout) {
	lineSpec := &canvas.LineSpec{Stroke: pkginks.FixedInk(scatterGridColour), StrokeWidth: scatterGridStrokeWidth}
	for _, tick := range layout.XAxis.NumericTicks() {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			X1:   tick.Position,
			Y1:   layout.Plot.Y,
			X2:   tick.Position,
			Y2:   layout.Plot.Y + layout.Plot.H,
		})
	}

	for _, tick := range layout.YAxis.NumericTicks() {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			X1:   layout.Plot.X,
			Y1:   tick.Position,
			X2:   layout.Plot.X + layout.Plot.W,
			Y2:   tick.Position,
		})
	}

	for _, band := range layout.XAxis.CategoricalBands() {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			X1:   band.Start,
			Y1:   layout.Plot.Y,
			X2:   band.Start,
			Y2:   layout.Plot.Y + layout.Plot.H,
		})
	}

	if bands := layout.XAxis.CategoricalBands(); len(bands) > 0 {
		last := bands[len(bands)-1]
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			X1:   last.End,
			Y1:   layout.Plot.Y,
			X2:   last.End,
			Y2:   layout.Plot.Y + layout.Plot.H,
		})
	}

	for _, band := range layout.YAxis.CategoricalBands() {
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			X1:   layout.Plot.X,
			Y1:   band.Start,
			X2:   layout.Plot.X + layout.Plot.W,
			Y2:   band.Start,
		})
	}

	if bands := layout.YAxis.CategoricalBands(); len(bands) > 0 {
		last := bands[len(bands)-1]
		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: lineSpec,
			X1:   layout.Plot.X,
			Y1:   last.End,
			X2:   layout.Plot.X + layout.Plot.W,
			Y2:   last.End,
		})
	}
}

func addScatterAxisLabels(cv *canvas.Canvas, layout ScatterLayout) {
	titleSpec := &canvas.TextSpec{Ink: pkginks.FixedInk(scatterLabelColour), FontSize: 12, Anchor: canvas.AnchorMiddle}
	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    titleSpec,
		X:       layout.Plot.X + layout.Plot.W/2,
		Y:       layout.Plot.Y + layout.Plot.H + 56,
		Content: layout.XAxis.Title,
	})

	yTitleSpec := &canvas.TextSpec{
		Ink:      pkginks.FixedInk(scatterLabelColour),
		FontSize: 12,
		Anchor:   canvas.AnchorMiddle,
		Rotation: -math.Pi / 2,
	}
	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    yTitleSpec,
		X:       layout.Plot.X - 72,
		Y:       layout.Plot.Y + layout.Plot.H/2,
		Content: layout.YAxis.Title,
	})

	tickSpec := &canvas.TextSpec{Ink: pkginks.FixedInk(scatterLabelColour), FontSize: 10, Anchor: canvas.AnchorMiddle}
	for _, tick := range layout.XAxis.NumericTicks() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    tickSpec,
			X:       tick.Position,
			Y:       layout.Plot.Y + layout.Plot.H + 18,
			Content: tick.Label,
		})
	}

	for _, band := range layout.XAxis.CategoricalBands() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    tickSpec,
			X:       band.Center,
			Y:       layout.Plot.Y + layout.Plot.H + 18,
			Content: band.Label,
		})
	}

	yTickSpec := &canvas.TextSpec{Ink: pkginks.FixedInk(scatterLabelColour), FontSize: 10, Anchor: canvas.AnchorEnd}
	for _, tick := range layout.YAxis.NumericTicks() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    yTickSpec,
			X:       layout.Plot.X - 8,
			Y:       tick.Position,
			Content: tick.Label,
		})
	}

	for _, band := range layout.YAxis.CategoricalBands() {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    yTickSpec,
			X:       layout.Plot.X - 8,
			Y:       band.Center,
			Content: band.Label,
		})
	}
}

func addScatterPoints(cv *canvas.Canvas, points []ScatterPoint, inks Inks) {
	borderWidth := scatterBorderWidth
	if inks.HasBorderMetric {
		borderWidth = scatterMetricBorder
	}

	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{Fill: inks.Fill, Border: inks.Border, BorderWidth: borderWidth},
	}
	for _, point := range points {
		fillValue := pkginks.MetricValueForFile(point.File, inks.Fill)
		borderValue := pkginks.MetricValueForFile(point.File, inks.Border)
		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   discSpec,
			X:      point.X,
			Y:      point.Y,
			Radius: point.Radius,
			Fill:   fillValue,
			Border: borderValue,
		})

		label, fontSize := scatterLabel(point.Label, point.Radius)
		labelColour := canvas.TextColourFor(inks.Fill.Dip(fillValue))
		labelSpec := &canvas.TextSpec{
			Ink:      pkginks.FixedInk(labelColour),
			FontSize: fontSize,
			Anchor:   canvas.AnchorMiddle,
		}
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    labelSpec,
			X:       point.X,
			Y:       point.Y,
			Content: label,
		})
	}
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
