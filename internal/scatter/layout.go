package scatter

import (
	"cmp"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
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
	File      *model.File
	Directory *model.Directory
	Geometry  geometry.Circle
	Label     string
}

// ScatterLayout is the rendered geometry for a scatter plot.
type ScatterLayout struct {
	Plot   geometry.Rect
	XAxis  ResolvedAxis
	YAxis  ResolvedAxis
	Points []ScatterPoint
}

// Layout converts the dataset into absolute plot geometry.
func Layout(dataset Dataset, width, height int, xAxis, yAxis AxisSpec) ScatterLayout {
	plotW := math.Max(1, float64(width)-scatterPlotLeftMargin-scatterPlotRightMargin)
	plotH := math.Max(1, float64(height)-scatterPlotTopMargin-scatterPlotBottomMargin)
	plot := geometry.Rect{
		Min: geometry.Point{X: scatterPlotLeftMargin, Y: scatterPlotTopMargin},
		Max: geometry.Point{X: scatterPlotLeftMargin + plotW, Y: scatterPlotTopMargin + plotH},
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
			File:      point.File,
			Directory: point.Directory,
			Geometry: geometry.Circle{
				Center: geometry.Point{
					X: positionForValue(point.X, layout.XAxis, plot, horizontalAxis),
					Y: positionForValue(point.Y, layout.YAxis, plot, verticalAxis),
				},
				Radius: scaleRadius(point.Size, minSize, maxSize, minRadius, maxRadius),
			},
			Label: point.Name(),
		})
	}

	slices.SortFunc(layout.Points, func(a, b ScatterPoint) int {
		if cmp := cmp.Compare(b.Geometry.Radius, a.Geometry.Radius); cmp != 0 {
			return cmp
		}

		return cmp.Compare(a.Label, b.Label)
	})

	return layout
}

// OffsetLayout shifts the layout when legend space has been reserved.
func OffsetLayout(layout *ScatterLayout, offset geometry.Vector) {
	layout.Plot = layout.Plot.Translate(offset)
	layout.XAxis.Offset(offset.X)
	layout.YAxis.Offset(offset.Y)

	for i := range layout.Points {
		layout.Points[i].Geometry.Center = layout.Points[i].Geometry.Center.Translate(offset)
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
	cellW := axisSlotSize(layout.XAxis, layout.Plot.Width(), pointCount)
	cellH := axisSlotSize(layout.YAxis, layout.Plot.Height(), pointCount)
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
