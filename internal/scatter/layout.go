package scatter

import (
	"cmp"
	"math"
	"slices"
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	scatterPlotTopMargin    = 48.0
	scatterPlotRightMargin  = 32.0
	scatterPlotBottomMargin = 96.0
	scatterPlotLeftMargin   = 96.0
	scatterMinRadius        = 12.0
	scatterMaxRadiusFactor  = 0.45
	scatterMinNumericSlots  = 8.0
	scatterTickCount        = 5
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
		XAxis: resolveAxis(dataset.Points, plot, xAxis, true),
		YAxis: resolveAxis(dataset.Points, plot, yAxis, false),
	}

	minSize, maxSize := sizeExtent(dataset.Points)
	maxRadius := math.Max(scatterMinRadius, maxPointRadius(layout, len(dataset.Points)))
	minRadius := scatterMinRadius

	layout.Points = make([]ScatterPoint, 0, len(dataset.Points))
	for _, point := range dataset.Points {
		layout.Points = append(layout.Points, ScatterPoint{
			File:   point.File,
			X:      positionForValue(point.X, layout.XAxis, plot, true),
			Y:      positionForValue(point.Y, layout.YAxis, plot, false),
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
	for i := range layout.XAxis.NumericTicks() {
		layout.XAxis.Numeric.Ticks[i].Position += dx
	}
	for i := range layout.YAxis.NumericTicks() {
		layout.YAxis.Numeric.Ticks[i].Position += dy
	}
	for i := range layout.XAxis.CategoricalBands() {
		layout.XAxis.Categorical.Bands[i].Start += dx
		layout.XAxis.Categorical.Bands[i].End += dx
		layout.XAxis.Categorical.Bands[i].Center += dx
	}
	for i := range layout.YAxis.CategoricalBands() {
		layout.YAxis.Categorical.Bands[i].Start += dy
		layout.YAxis.Categorical.Bands[i].End += dy
		layout.YAxis.Categorical.Bands[i].Center += dy
	}
	for i := range layout.Points {
		layout.Points[i].X += dx
		layout.Points[i].Y += dy
	}
}

func resolveAxis(points []PointDatum, plot PlotRect, spec AxisSpec, horizontal bool) ResolvedAxis {
	axis := ResolvedAxis{Spec: spec, Title: string(spec.Metric)}
	if spec.Kind == metric.Classification {
		axis.Categorical = &CategoricalAxis{Bands: categoricalBands(points, plot, spec, horizontal)}
		return axis
	}

	minValue, maxValue := numericExtent(points, spec, horizontal)
	axis.Numeric = &NumericAxis{
		Min:   minValue,
		Max:   maxValue,
		Ticks: numericTicks(minValue, maxValue, plot, horizontal),
	}

	return axis
}

func (a ResolvedAxis) NumericTicks() []AxisTick {
	if a.Numeric == nil {
		return nil
	}

	return a.Numeric.Ticks
}

func (a ResolvedAxis) CategoricalBands() []AxisBand {
	if a.Categorical == nil {
		return nil
	}

	return a.Categorical.Bands
}

func numericExtent(points []PointDatum, spec AxisSpec, horizontal bool) (float64, float64) {
	if len(points) == 0 {
		return 0, 0
	}

	first := axisNumeric(points[0], spec, horizontal)
	minValue := first
	maxValue := first
	for _, point := range points[1:] {
		value := axisNumeric(point, spec, horizontal)
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}

	return minValue, maxValue
}

func categoricalBands(points []PointDatum, plot PlotRect, spec AxisSpec, horizontal bool) []AxisBand {
	labels := make([]string, 0, len(points))
	seen := map[string]bool{}
	for _, point := range points {
		label := axisCategory(point, spec, horizontal)
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}

	slices.Sort(labels)
	if len(labels) == 0 {
		return nil
	}

	bands := make([]AxisBand, len(labels))
	span := plot.W
	origin := plot.X
	if !horizontal {
		span = plot.H
		origin = plot.Y
	}
	bandSize := span / float64(len(labels))
	for i, label := range labels {
		start := origin + float64(i)*bandSize
		bands[i] = AxisBand{
			Label:  label,
			Start:  start,
			End:    start + bandSize,
			Center: start + bandSize/2,
		}
	}

	return bands
}

func numericTicks(minValue, maxValue float64, plot PlotRect, horizontal bool) []AxisTick {
	if minValue == maxValue {
		position := plot.X + plot.W/2
		if !horizontal {
			position = plot.Y + plot.H/2
		}

		return []AxisTick{{Value: minValue, Label: formatTick(minValue), Position: position}}
	}

	ticks := make([]AxisTick, scatterTickCount)
	for i := range scatterTickCount {
		norm := float64(i) / float64(scatterTickCount-1)
		value := minValue + (maxValue-minValue)*norm
		position := plot.X + plot.W*norm
		if !horizontal {
			position = plot.Y + plot.H*(1-norm)
		}
		ticks[i] = AxisTick{Value: value, Label: formatTick(value), Position: position}
	}

	return ticks
}

func formatTick(value float64) string {
	return strconv.FormatFloat(value, 'g', 3, 64)
}

func positionForValue(value AxisValue, axis ResolvedAxis, plot PlotRect, horizontal bool) float64 {
	if axis.Categorical != nil {
		for _, band := range axis.Categorical.Bands {
			if band.Label == value.Category {
				return band.Center
			}
		}
		if horizontal {
			return plot.X + plot.W/2
		}

		return plot.Y + plot.H/2
	}

	minValue := axis.Numeric.Min
	maxValue := axis.Numeric.Max
	if minValue == maxValue {
		if horizontal {
			return plot.X + plot.W/2
		}

		return plot.Y + plot.H/2
	}

	norm := (value.Numeric - minValue) / (maxValue - minValue)
	if horizontal {
		return plot.X + plot.W*norm
	}

	return plot.Y + plot.H*(1-norm)
}

func sizeExtent(points []PointDatum) (float64, float64) {
	if len(points) == 0 {
		return 0, 0
	}

	minSize := points[0].Size
	maxSize := points[0].Size
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

func axisSlotSize(axis ResolvedAxis, span float64, pointCount int) float64 {
	if axis.Categorical != nil && len(axis.Categorical.Bands) > 0 {
		return span / float64(len(axis.Categorical.Bands))
	}

	return span / math.Max(scatterMinNumericSlots, math.Sqrt(float64(max(pointCount, 1))))
}

func scaleRadius(value, minValue, maxValue, minRadius, maxRadius float64) float64 {
	if maxRadius <= minRadius || minValue == maxValue {
		return maxRadius
	}

	norm := (value - minValue) / (maxValue - minValue)
	return minRadius + (maxRadius-minRadius)*norm
}

func axisNumeric(point PointDatum, spec AxisSpec, horizontal bool) float64 {
	if horizontal {
		return point.X.Numeric
	}

	return point.Y.Numeric
}

func axisCategory(point PointDatum, spec AxisSpec, horizontal bool) string {
	if horizontal {
		return point.X.Category
	}

	return point.Y.Category
}
