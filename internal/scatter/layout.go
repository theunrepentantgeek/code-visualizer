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
	scatterMinTickGaps      = 4
	scatterMaxTickGaps      = 10
	scatterTargetTickGaps   = 7
	scatterZeroGapRatio     = 0.2
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

type axisDirection int

const (
	horizontalAxis axisDirection = iota
	verticalAxis
)

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

func resolveAxis(points []PointDatum, plot PlotRect, spec AxisSpec, direction axisDirection) ResolvedAxis {
	axis := ResolvedAxis{Spec: spec, Title: string(spec.Metric)}
	if spec.Kind == metric.Classification {
		axis.Categorical = &CategoricalAxis{Bands: categoricalBands(points, plot, direction)}

		return axis
	}

	minValue, maxValue := numericExtent(points, direction)
	axis.Numeric = &NumericAxis{
		Min:   minValue,
		Max:   maxValue,
		Ticks: numericTicks(minValue, maxValue, plot, direction),
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

func numericExtent(points []PointDatum, direction axisDirection) (minValue, maxValue float64) {
	if len(points) == 0 {
		return 0, 0
	}

	first := direction.numericValue(points[0])
	minValue = first
	maxValue = first

	for _, point := range points[1:] {
		value := direction.numericValue(point)
		if value < minValue {
			minValue = value
		}

		if value > maxValue {
			maxValue = value
		}
	}

	return minValue, maxValue
}

func categoricalBands(points []PointDatum, plot PlotRect, direction axisDirection) []AxisBand {
	labels := make([]string, 0, len(points))
	seen := map[string]bool{}

	for _, point := range points {
		label := direction.categoryValue(point)
		if !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}

	slices.Sort(labels)

	if len(labels) == 0 {
		return nil
	}

	origin, span := direction.span(plot)
	bands := make([]AxisBand, len(labels))
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

func numericTicks(minValue, maxValue float64, plot PlotRect, direction axisDirection) []AxisTick {
	if minValue == maxValue {
		return []AxisTick{{
			Value:    minValue,
			Label:    formatTick(minValue, 0),
			Position: direction.center(plot),
		}}
	}

	niceMin, niceMax := includeNearZero(minValue, maxValue)
	step, niceStart, niceEnd := niceTickStep(niceMin, niceMax)
	gapCount := int(math.Round((niceEnd - niceStart) / step))
	ticks := make([]AxisTick, gapCount+1)

	for i := range gapCount + 1 {
		value := niceStart + step*float64(i)
		if i == gapCount {
			value = niceEnd
		}

		norm := (value - niceStart) / (niceEnd - niceStart)
		ticks[i] = AxisTick{
			Value:    value,
			Label:    formatTick(value, step),
			Position: direction.position(plot, norm),
		}
	}

	return ticks
}

func includeNearZero(
	minValue float64,
	maxValue float64,
) (lowerBound float64, upperBound float64) {
	span := maxValue - minValue
	if span <= 0 {
		return minValue, maxValue
	}

	snapMargin := span * scatterZeroGapRatio

	if minValue > 0 && minValue <= snapMargin {
		minValue = 0
	}

	if maxValue < 0 && -maxValue <= snapMargin {
		maxValue = 0
	}

	return minValue, maxValue
}

type tickCandidate struct {
	step     float64
	start    float64
	end      float64
	gaps     int
	gapDelta float64
	padding  float64
}

// betterThan reports whether c is a preferable tick layout to other.
func (c tickCandidate) betterThan(other tickCandidate) bool {
	if c.gapDelta != other.gapDelta {
		return c.gapDelta < other.gapDelta
	}

	if c.gaps != other.gaps {
		return c.gaps > other.gaps
	}

	return c.padding < other.padding
}

// makeTickCandidate evaluates a candidate step against [minValue, maxValue].
// Returns false if the resulting gap count is outside the allowed range.
func makeTickCandidate(
	minValue float64,
	maxValue float64,
	candidateStep float64,
) (tickCandidate, bool) {
	start := math.Floor(minValue/candidateStep) * candidateStep
	end := math.Ceil(maxValue/candidateStep) * candidateStep
	gaps := int(math.Round((end - start) / candidateStep))

	if gaps < scatterMinTickGaps || gaps > scatterMaxTickGaps {
		return tickCandidate{}, false
	}

	return tickCandidate{
		step:     candidateStep,
		start:    start,
		end:      end,
		gaps:     gaps,
		gapDelta: math.Abs(float64(gaps - scatterTargetTickGaps)),
		padding:  (minValue-start)/candidateStep + (end-maxValue)/candidateStep,
	}, true
}

// bestTickCandidate searches the anchor/exponent grid for the best tick layout.
func bestTickCandidate(
	minValue float64,
	maxValue float64,
) (tickCandidate, bool) {
	span := maxValue - minValue
	rawStep := span / float64(scatterTargetTickGaps)
	baseExponent := math.Floor(math.Log10(rawStep))
	anchors := []float64{1, 2, 2.5, 5, 10}

	var (
		best  tickCandidate
		found bool
	)

	for exponent := baseExponent - 1; exponent <= baseExponent+1; exponent++ {
		scale := math.Pow(10, exponent)
		for _, anchor := range anchors {
			cand, ok := makeTickCandidate(minValue, maxValue, anchor*scale)
			if !ok {
				continue
			}

			if !found || cand.betterThan(best) {
				best = cand
				found = true
			}
		}
	}

	return best, found
}

func niceTickStep(
	minValue float64,
	maxValue float64,
) (step float64, start float64, end float64) {
	span := maxValue - minValue
	if span <= 0 {
		return 1, minValue, maxValue
	}

	if best, ok := bestTickCandidate(minValue, maxValue); ok {
		return best.step, best.start, best.end
	}

	return span / float64(scatterTargetTickGaps), minValue, maxValue
}

func formatTick(value, step float64) string {
	if step <= 0 {
		return strconv.FormatFloat(value, 'g', 6, 64)
	}

	decimals := 0
	for decimals < 6 {
		scaled := step * math.Pow(10, float64(decimals))
		if math.Abs(scaled-math.Round(scaled)) < 1e-9 {
			break
		}

		decimals++
	}

	return strconv.FormatFloat(value, 'f', decimals, 64)
}

func positionForValue(value AxisValue, axis ResolvedAxis, plot PlotRect, direction axisDirection) float64 {
	if axis.Categorical != nil {
		for _, band := range axis.Categorical.Bands {
			if band.Label == value.Category {
				return band.Center
			}
		}

		return direction.center(plot)
	}

	minValue := axis.Numeric.Min
	maxValue := axis.Numeric.Max

	if minValue == maxValue {
		return direction.center(plot)
	}

	norm := (value.Numeric - minValue) / (maxValue - minValue)

	return direction.position(plot, norm)
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

func (d axisDirection) center(plot PlotRect) float64 {
	if d == horizontalAxis {
		return plot.X + plot.W/2
	}

	return plot.Y + plot.H/2
}

func (d axisDirection) position(plot PlotRect, norm float64) float64 {
	if d == horizontalAxis {
		return plot.X + plot.W*norm
	}

	return plot.Y + plot.H*(1-norm)
}

func (d axisDirection) span(plot PlotRect) (origin, span float64) {
	if d == horizontalAxis {
		return plot.X, plot.W
	}

	return plot.Y, plot.H
}

func (d axisDirection) numericValue(point PointDatum) float64 {
	if d == horizontalAxis {
		return point.X.Numeric
	}

	return point.Y.Numeric
}

func (d axisDirection) categoryValue(point PointDatum) string {
	if d == horizontalAxis {
		return point.X.Category
	}

	return point.Y.Category
}
