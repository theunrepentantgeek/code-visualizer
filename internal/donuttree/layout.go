// Package donuttree implements deterministic geometry for donut tree visualizations.
package donuttree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const donutRingWidthRatio = 0.9

// Layout builds a concentric directory-sector layout. The root is represented
// by the central anchor, while its directories begin in the first annular ring.
func Layout(root *model.Directory, canvasSize int, sizeMetric metric.Name) LayoutResult {
	center := float64(canvasSize) / 2
	result := LayoutResult{
		Center: geometry.NewPoint(center, center),
	}

	if root == nil {
		return result
	}

	result.RootName = root.Name
	maxDepth := maxDirectoryDepth(root)

	if maxDepth == 0 {
		result.AnchorRadius = math.Max(center, 0)

		return result
	}

	ringWidth := math.Max(center, 0) / float64(maxDepth+1)
	result.AnchorRadius = ringWidth
	result.Children = layoutChildren(root.Dirs, 1, -math.Pi/2, 2*math.Pi, ringWidth, sizeMetric)

	return result
}

func layoutChildren(
	dirs []*model.Directory,
	depth int,
	startAngle, sweepAngle, ringWidth float64,
	sizeMetric metric.Name,
) []DonutNode {
	if len(dirs) == 0 {
		return nil
	}

	values, sumPositive := allocationValues(dirs, sizeMetric)
	minimum := math.Min(math.Pi/180, sweepAngle/float64(len(dirs)))
	remaining := sweepAngle - minimum*float64(len(dirs))
	innerRadius := float64(depth) * ringWidth
	outerRadius := innerRadius + ringWidth*donutRingWidthRatio
	children := make([]DonutNode, 0, len(dirs))
	childStart := startAngle

	for i, dir := range dirs {
		childSweep := childSweep(values[i], sumPositive, minimum, remaining, sweepAngle, len(dirs))

		child := DonutNode{
			Directory:   dir,
			Depth:       depth,
			StartAngle:  childStart,
			SweepAngle:  childSweep,
			InnerRadius: innerRadius,
			OuterRadius: outerRadius,
		}
		if dir != nil {
			child.Children = layoutChildren(
				dir.Dirs,
				depth+1,
				child.StartAngle,
				child.SweepAngle,
				ringWidth,
				sizeMetric,
			)
		}

		children = append(children, child)
		childStart = child.EndAngle()
	}

	return children
}

func allocationValues(dirs []*model.Directory, sizeMetric metric.Name) ([]float64, float64) {
	values := make([]float64, len(dirs))

	var sumPositive float64

	for i, dir := range dirs {
		value := directoryMetricValue(dir, sizeMetric)

		if value > 0 {
			values[i] = value
			sumPositive += value
		}
	}

	return values, sumPositive
}

func childSweep(
	value, sumPositive, minimum, remaining, parentSweep float64,
	childCount int,
) float64 {
	if sumPositive == 0 {
		return parentSweep / float64(childCount)
	}

	if value > 0 {
		return minimum + remaining*value/sumPositive
	}

	return minimum
}

// directoryMetricValue returns a directory's allocation metric, preferring
// quantities to measures. Missing and non-positive values do not allocate area.
func directoryMetricValue(dir *model.Directory, sizeMetric metric.Name) float64 {
	if dir == nil {
		return 0
	}

	if quantity, ok := dir.Quantity(sizeMetric); ok {
		return float64(quantity)
	}

	if measure, ok := dir.Measure(sizeMetric); ok {
		return measure
	}

	return 0
}

func maxDirectoryDepth(dir *model.Directory) int {
	if dir == nil {
		return 0
	}

	maxDepth := 0

	for _, child := range dir.Dirs {
		depth := 1 + maxDirectoryDepth(child)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}
