package scatter

import (
	"cmp"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	scatterPlotTopMargin    = 48.0
	scatterPlotRightMargin  = 32.0
	scatterPlotBottomMargin = 96.0
	scatterPlotLeftMargin   = 96.0
	scatterMinRadius        = 12.0
	scatterMaxRadiusFactor  = 0.45
)

// ScatterPoint is one fully laid-out disc.
type ScatterPoint struct {
	File   *model.File
	X      float64
	Y      float64
	Radius float64
	Label  string
}

// ScatterLayout is the rendered geometry for a scatter plot.
type ScatterLayout struct {
	Plot   PlotRect
	XAxis  ResolvedAxis
	YAxis  ResolvedAxis
	Points []ScatterPoint
}

// Layout converts the dataset into absolute plot geometry.
func Layout(dataset Dataset, width, height int, xAxis, yAxis AxisSpec) ScatterLayout {
	plot := PlotRect{
		X: scatterPlotLeftMargin,
		Y: scatterPlotTopMargin,
		W: math.Max(1, float64(width)-scatterPlotLeftMargin-scatterPlotRightMargin),
		H: math.Max(1, float64(height)-scatterPlotTopMargin-scatterPlotBottomMargin),
	}

	layout := ScatterLayout{
		Plot:  plot,
		XAxis: resolveAxis(dataset.Points, plot, xAxis, horizontalAxis),
		YAxis: resolveAxis(dataset.Points, plot, yAxis, verticalAxis),
	}

	minSize, maxSize := sizeExtent(dataset.Points)
	maxRadius := math.Max(scatterMinRadius, maxPointRadius(layout, len(dataset.Points)))
	minRadius := scatterMinRadius

	layout.Points = make([]ScatterPoint, 0, len(dataset.Points))
	for _, point := range dataset.Points {
		layout.Points = append(layout.Points, ScatterPoint{
			File:   point.File,
			X:      positionForValue(point.X, layout.XAxis, plot, horizontalAxis),
			Y:      positionForValue(point.Y, layout.YAxis, plot, verticalAxis),
			Radius: scaleRadius(point.Size, minSize, maxSize, minRadius, maxRadius),
			Label:  point.File.Name,
		})
	}

	slices.SortFunc(layout.Points, func(a, b ScatterPoint) int {
		if cmp := cmp.Compare(b.Radius, a.Radius); cmp != 0 {
			return cmp
		}

		return cmp.Compare(a.Label, b.Label)
	})

	return layout
}

// OffsetLayout shifts the layout when legend space has been reserved.
func OffsetLayout(layout *ScatterLayout, dx, dy float64) {
	layout.Plot.X += dx
	layout.Plot.Y += dy
	layout.XAxis.Offset(dx)
	layout.YAxis.Offset(dy)

	for i := range layout.Points {
		layout.Points[i].X += dx
		layout.Points[i].Y += dy
	}
}

func sizeExtent(points []PointDatum) (minSize, maxSize float64) {
	if len(points) == 0 {
		return 0, 0
	}

	minSize = points[0].Size
	maxSize = points[0].Size

	for _, point := range points[1:] {
		if point.Size < minSize {
			minSize = point.Size
		}

		if point.Size > maxSize {
			maxSize = point.Size
		}
	}

	return minSize, maxSize
}

func maxPointRadius(layout ScatterLayout, pointCount int) float64 {
	cellW := axisSlotSize(layout.XAxis, layout.Plot.W, pointCount)
	cellH := axisSlotSize(layout.YAxis, layout.Plot.H, pointCount)
	maxRadius := math.Min(cellW, cellH) * scatterMaxRadiusFactor

	if maxRadius < 4 {
		return 4
	}

	return maxRadius
}

func scaleRadius(value, minValue, maxValue, minRadius, maxRadius float64) float64 {
	if maxRadius <= minRadius || minValue == maxValue {
		return maxRadius
	}

	norm := (value - minValue) / (maxValue - minValue)

	return minRadius + (maxRadius-minRadius)*norm
}
