// Package radialtree implements data types and layout algorithms for radial tree visualizations.
package radialtree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	margin             = 40.0
	dirDiscFactor      = 0.06
	minDirDisc         = 4.0
	maxDiscFactor      = 0.40
	minFileDisc        = 3.0
	minFileLabelWidth  = 72.0 // px: ~10 chars × 12pt × 0.6 char-width ratio
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

	totalLeaves := computeLeafCount(root)

	// Stage 1 + 2: make room for file labels by expanding arc allocation and
	// increasing ring spacing when needed.
	//
	// Stage 1 (arc expansion): files at shallower depths receive a higher
	// virtual weight so they are allocated a proportionally larger angular arc.
	// Deep directories — whose files already have large arcs at large radii —
	// give up some angular arc to benefit shallow directories.
	//
	// Stage 2 (ring-spacing scale): after arc expansion, the shallowest files
	// get arc ≈ minFileLabelWidth * totalLeaves / virtualTotal pixels.  If that
	// is less than minFileLabelWidth, scale ringSpacing up by the ratio
	// virtualTotal/totalLeaves — effectively pushing all discs outward until
	// labels fit.  The scale is ≤ the pure Stage-2 factor (no virtual weights)
	// because Stage 1 has already reduced the required increase.
	if totalLeaves > 0 && labels == LabelAll {
		minDepth := computeMinFileDepth(root, 0)
		if minDepth > 0 {
			virtualTotal := computeVirtualLeafCount(root, 0, ringSpacing, float64(totalLeaves))
			neededRS := ringSpacing * virtualTotal / float64(totalLeaves)
			if neededRS > ringSpacing {
				ringSpacing = neededRS
			}
		}
	}

	effectiveMaxDiscFactor := adjustedDiscFactor(n1, ringSpacing, maxDiscFactor)
	dp := buildDiscParams(root, discMetric, minFileDisc, ringSpacing*effectiveMaxDiscFactor)

	// Start at top (−π/2) and sweep the full circle clockwise.
	return layoutDir(root, 0, -math.Pi/2, 2*math.Pi, ringSpacing, discMetric, labels, dp, float64(totalLeaves))
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
	totalLeaves float64,
) RadialNode {
	// Place this directory at the midpoint of its angular sector.
	angle := startAngle + sweepAngle/2
	radius := float64(depth) * ringSpacing

	dirDisc := ringSpacing * dirDiscFactor
	if dirDisc < minDirDisc {
		dirDisc = minDirDisc
	}

	node := RadialNode{
		X:           radius * math.Cos(angle),
		Y:           radius * math.Sin(angle),
		DiscRadius:  dirDisc,
		Angle:       angle,
		Label:       dir.Name,
		IsDirectory: true,
		ShowLabel:   labels == LabelAll || labels == LabelFoldersOnly,
	}

	// Use virtual leaf counts for arc allocation so that directories with many
	// direct file children receive proportionally more arc than their raw leaf
	// count would give them, compensating for the smaller arc-per-pixel at
	// shallower radii.
	parentVirtualLeaves := computeVirtualLeafCount(dir, depth, ringSpacing, totalLeaves)
	if parentVirtualLeaves <= 0 {
		parentVirtualLeaves = 1
	}

	childStart := startAngle
	childRadius := float64(depth+1) * ringSpacing

	// Files first: each file is a leaf with virtual weight fileVirtualWeight.
	fileVW := fileVirtualWeight(depth+1, ringSpacing, totalLeaves)

	for _, f := range dir.Files {
		childSweep := sweepAngle * fileVW / parentVirtualLeaves
		childAngle := childStart + childSweep/2

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
		childStart += childSweep
	}

	// Subdirs: each gets arc proportional to its virtual leaf count.
	for _, d := range dir.Dirs {
		childVirtualLeaves := computeVirtualLeafCount(d, depth+1, ringSpacing, totalLeaves)
		if childVirtualLeaves <= 0 {
			childVirtualLeaves = 1
		}

		childSweep := sweepAngle * childVirtualLeaves / parentVirtualLeaves
		child := layoutDir(d, depth+1, childStart, childSweep, ringSpacing, discMetric, labels, dp, totalLeaves)
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

	model.WalkFiles(root, func(f *model.File) {
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

// computeMinFileDepth returns the depth of the shallowest file node in the tree
// rooted at dir, where dir itself is at the given depth.  Returns -1 if no
// files exist anywhere under dir.
func computeMinFileDepth(dir *model.Directory, depth int) int {
	if len(dir.Files) > 0 {
		return depth + 1
	}

	minD := -1

	for _, d := range dir.Dirs {
		if md := computeMinFileDepth(d, depth+1); md >= 0 {
			if minD < 0 || md < minD {
				minD = md
			}
		}
	}

	return minD
}

// fileVirtualWeight returns the virtual arc weight for a single file node placed
// at the given depth ring.  The weight is at least 1 and grows when the
// standard arc allocation (2π/totalLeaves × depth × ringSpacing) would be
// narrower than minFileLabelWidth, ensuring shallow files receive enough arc.
func fileVirtualWeight(depth int, ringSpacing, totalLeaves float64) float64 {
	if totalLeaves <= 0 || ringSpacing <= 0 || depth <= 0 {
		return 1.0
	}

	arcPerRawLeaf := 2 * math.Pi / totalLeaves * float64(depth) * ringSpacing
	if arcPerRawLeaf <= 0 {
		return 1.0
	}

	w := minFileLabelWidth / arcPerRawLeaf
	if w < 1.0 {
		return 1.0
	}

	return w
}

// computeVirtualLeafCount returns the sum of virtual arc weights for all leaves
// under dir.  depth is dir's depth (root = 0).  Files at shallower depths
// receive higher virtual weight; files already at adequate arc width receive 1.
func computeVirtualLeafCount(dir *model.Directory, depth int, ringSpacing, totalLeaves float64) float64 {
	w := fileVirtualWeight(depth+1, ringSpacing, totalLeaves)
	count := w * float64(len(dir.Files))

	for _, d := range dir.Dirs {
		count += computeVirtualLeafCount(d, depth+1, ringSpacing, totalLeaves)
	}

	return count
}
