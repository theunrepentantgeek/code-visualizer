// Package radialtree implements data types and layout algorithms for radial tree visualizations.
package radialtree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/walk"
)

const (
	margin        = 40.0
	dirDiscFactor = 0.06
	minDirDisc    = 4.0
	maxDiscFactor = 0.40
	minFileDisc   = 3.0
)

// Layout builds a radial tree layout for root.
// canvasSize is the width and height of the square canvas in pixels.
// discMetric is the metric used to scale file node disc sizes.
// labels controls which labels are shown.
func Layout(root *model.Directory, canvasSize int, discMetric metric.Name, labels LabelMode) RadialNode {
	maxDepth := computeMaxDepth(root)

	var ringSpacing float64
	if maxDepth == 0 {
		// Degenerate case: root has no children; use a fixed ring radius.
		ringSpacing = float64(canvasSize) / 4.0
	} else {
		ringSpacing = (float64(canvasSize)/2.0 - margin) / float64(maxDepth+1)
	}

	n1 := len(root.Files) + len(root.Dirs)
	if n1 > 0 && maxDepth > 0 {
		// Ensure ring 1 has enough circumference for n1 nodes at minimum disc size.
		const minGapPixels = 4.0

		minCircumference := float64(n1) * (2*minFileDisc + minGapPixels)

		minRingSpacing := minCircumference / (2 * math.Pi)
		if minRingSpacing > ringSpacing {
			ringSpacing = minRingSpacing
		}
	}

	effectiveMaxDiscFactor := adjustedDiscFactor(n1, ringSpacing, maxDiscFactor)
	dp := buildDiscParams(root, discMetric, minFileDisc, ringSpacing*effectiveMaxDiscFactor)

	// Start at top (−π/2) and sweep the full circle clockwise.
	return layoutDir(root, 0, -math.Pi/2, 2*math.Pi, ringSpacing, discMetric, labels, dp)
}

// discParams holds the precomputed parameters used to scale file disc radii.
type discParams struct {
	fileMin   float64 // minimum pixel disc radius for file nodes
	fileMax   float64 // maximum pixel disc radius for file nodes
	metricMin float64 // minimum non-zero metric value across all files
	metricMax float64 // maximum non-zero metric value across all files
	useEqual  bool    // true when all metric values are equal or no values exist
}

func buildDiscParams(root *model.Directory, discMetric metric.Name, fileMin, fileMax float64) discParams {
	dp := discParams{fileMin: fileMin, fileMax: fileMax}

	vals := collectFileMetricValues(root, discMetric)
	if len(vals) == 0 {
		return dp
	}

	minVal, maxVal := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < minVal {
			minVal = v
		}

		if v > maxVal {
			maxVal = v
		}
	}

	dp.metricMin = minVal
	dp.metricMax = maxVal
	dp.useEqual = minVal == maxVal

	return dp
}

func layoutDir(
	dir *model.Directory,
	depth int,
	startAngle, sweepAngle, ringSpacing float64,
	discMetric metric.Name,
	labels LabelMode,
	dp discParams,
) RadialNode {
	// Place this directory at the midpoint of its angular sector.
	angle := startAngle + sweepAngle/2
	radius := float64(depth) * ringSpacing

	dirDisc := math.Max(ringSpacing*dirDiscFactor, minDirDisc)

	node := RadialNode{
		X:           radius * math.Cos(angle),
		Y:           radius * math.Sin(angle),
		DiscRadius:  dirDisc,
		Angle:       angle,
		Label:       dir.Name,
		IsDirectory: true,
		ShowLabel:   labels == LabelAll || labels == LabelFoldersOnly,
	}

	allocationUnits := childAllocationUnits(dir)
	if allocationUnits == 0 {
		return node
	}

	contentSweep := sweepAngle
	childStart := startAngle

	if depth > 0 {
		// Reserve one child-sized blank slot on each side of non-root directory
		// groups so sibling folders don't run directly into each other.
		paddingSweep := sweepAngle / float64(allocationUnits+2)
		contentSweep -= 2 * paddingSweep
		childStart += paddingSweep
	}

	childRadius := float64(depth+1) * ringSpacing
	fileSweep := contentSweep / float64(allocationUnits)

	// Files first: each file occupies one allocation unit of the padded sweep.
	for _, f := range dir.Files {
		childAngle := childStart + fileSweep/2

		fileNode := RadialNode{
			X:           childRadius * math.Cos(childAngle),
			Y:           childRadius * math.Sin(childAngle),
			DiscRadius:  fileDiscRadius(f, discMetric, dp),
			Angle:       childAngle,
			Label:       f.Name,
			IsDirectory: false,
			ShowLabel:   labels == LabelAll,
		}

		node.Children = append(node.Children, fileNode)
		childStart += fileSweep
	}

	// Subdirs: each gets a proportional slice of the padded sweep based on its
	// file-leaf weight, with empty directories still reserving one unit.
	for _, d := range dir.Dirs {
		weight := childWeight(d)
		childSweep := float64(weight) / float64(allocationUnits) * contentSweep
		child := layoutDir(d, depth+1, childStart, childSweep, ringSpacing, discMetric, labels, dp)
		node.Children = append(node.Children, child)
		childStart += childSweep
	}

	return node
}

// fileDiscRadius returns the disc pixel radius for f, scaled by the disc metric.
func fileDiscRadius(f *model.File, discMetric metric.Name, dp discParams) float64 {
	val := fileMetricValue(f, discMetric)
	if val <= 0 {
		return dp.fileMin
	}

	if dp.useEqual {
		// Single or uniform metric value: use the midpoint size.
		return (dp.fileMin + dp.fileMax) / 2
	}

	scaled := dp.fileMin + (val-dp.metricMin)/(dp.metricMax-dp.metricMin)*(dp.fileMax-dp.fileMin)

	return clamp(scaled, dp.fileMin, dp.fileMax)
}

// fileMetricValue returns the disc-metric value for f as a float64.
// Quantity is checked first (int64), then Measure (float64). Returns 0 if absent.
func fileMetricValue(f *model.File, discMetric metric.Name) float64 {
	if q, ok := f.Quantity(discMetric); ok {
		return float64(q)
	}

	if m, ok := f.Measure(discMetric); ok {
		return m
	}

	return 0
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// computeLeafCount returns the total number of file leaves under dir.
// Returns 0 for empty directories; callers are responsible for handling the
// zero case to avoid division by zero in sector calculations.
func computeLeafCount(dir *model.Directory) int {
	count := len(dir.Files)
	for _, d := range dir.Dirs {
		count += computeLeafCount(d)
	}

	return count
}

func childAllocationUnits(dir *model.Directory) int {
	units := len(dir.Files)
	for _, d := range dir.Dirs {
		units += childWeight(d)
	}

	return units
}

func childWeight(dir *model.Directory) int {
	leafCount := computeLeafCount(dir)
	if leafCount == 0 {
		return 1
	}

	return leafCount
}

// computeMaxDepth returns the maximum depth of any node in the tree rooted at dir.
// Root is at depth 0; its direct children (files or dirs) are at depth 1, etc.
func computeMaxDepth(dir *model.Directory) int {
	depth := 0

	if len(dir.Files) > 0 {
		depth = 1
	}

	for _, d := range dir.Dirs {
		if child := 1 + computeMaxDepth(d); child > depth {
			depth = child
		}
	}

	return depth
}

// collectFileMetricValues returns all non-zero disc-metric values across every file under root.
func collectFileMetricValues(root *model.Directory, discMetric metric.Name) []float64 {
	var vals []float64

	walk.Files(root, func(f *model.File) {
		v := fileMetricValue(f, discMetric)
		if v > 0 {
			vals = append(vals, v)
		}
	})

	return vals
}

// adjustedDiscFactor returns a maxDiscFactor scaled down so that n nodes
// fit on a ring of radius ringSpacing without their full-size discs overlapping.
// This ensures readable layout even when directories have many children.
func adjustedDiscFactor(n int, ringSpacing, baseMaxDiscFactor float64) float64 {
	if n <= 0 {
		return baseMaxDiscFactor
	}

	// Each node needs arc >= 2*discRadius + minGap pixels.
	// With n nodes on circumference 2π*ringSpacing:
	// maxDiscRadius = (π * ringSpacing / n) - minGap/2
	const minGap = 4.0

	maxR := (math.Pi * ringSpacing / float64(n)) - minGap/2
	if maxR <= 0 {
		return baseMaxDiscFactor * 0.1 // hard minimum
	}

	factor := maxR / ringSpacing
	if factor < baseMaxDiscFactor {
		return factor
	}

	return baseMaxDiscFactor
}
