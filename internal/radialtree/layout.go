// Package radialtree implements data types and layout algorithms for radial tree visualizations.
package radialtree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	margin        = 40.0
	dirDiscFactor = 0.06
	minDirDisc    = 4.0
	maxDiscFactor = 0.40
	minFileDisc   = 3.0
	// maxDirDiscFactor scales the base directory disc size to give the largest
	// directory disc room to grow when a directory disc metric is available.
	maxDirDiscFactor = 2.0
)

// Layout builds a radial tree layout for root.
// canvasSize is the width and height of the square canvas in pixels.
// discMetric is the metric used to scale file node disc sizes.
// dirDiscMetric is the aggregated metric used to scale directory node disc
// sizes; when empty (or absent from the model) directories use a fixed size.
// labels controls which labels are shown.
// grain controls whether files are included alongside directories.
func Layout(
	root *model.Directory,
	canvasSize int,
	discMetric metric.Name,
	dirDiscMetric metric.Name,
	labels LabelMode,
	grain Grain,
) RadialNode {
	maxDepth := computeMaxDepth(root, grain)

	var ringSpacing float64
	if maxDepth == 0 {
		// Degenerate case: root has no children; use a fixed ring radius.
		ringSpacing = float64(canvasSize) / 4.0
	} else {
		ringSpacing = (float64(canvasSize)/2.0 - margin) / float64(maxDepth+1)
	}

	n1 := len(visibleFiles(root, grain)) + len(root.Dirs)
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
	dp.dir = buildDirDiscParams(root, dirDiscMetric, ringSpacing)

	opts := layoutOptions{
		ringSpacing: ringSpacing,
		discMetric:  discMetric,
		labels:      labels,
		grain:       grain,
		discParams:  dp,
	}

	// Start at top (−π/2) and sweep the full circle clockwise.
	return layoutDir(root, 0, -math.Pi/2, 2*math.Pi, opts)
}

// layoutOptions holds the parameters that remain constant for every node
// visited during a single layout pass.
type layoutOptions struct {
	ringSpacing float64
	discMetric  metric.Name
	labels      LabelMode
	grain       Grain
	discParams  discParams
}

// discParams holds the precomputed parameters used to scale file disc radii.
type discParams struct {
	fileMin   float64 // minimum pixel disc radius for file nodes
	fileMax   float64 // maximum pixel disc radius for file nodes
	metricMin float64 // minimum non-zero metric value across all files
	metricMax float64 // maximum non-zero metric value across all files
	useEqual  bool    // true when all metric values are equal or no values exist
	dir       dirDiscParams
}

// dirDiscParams holds the precomputed parameters used to scale directory disc
// radii from an aggregated (rolled up) metric.
type dirDiscParams struct {
	metricName metric.Name // aggregated metric scaling directory discs
	base       float64     // pixel disc radius used when no metric value applies
	dirMin     float64     // minimum pixel disc radius for directory nodes
	dirMax     float64     // maximum pixel disc radius for directory nodes
	metricMin  float64     // minimum non-zero metric value across all directories
	metricMax  float64     // maximum non-zero metric value across all directories
	found      bool        // true when directory metric values vary and can be scaled
}

func buildDiscParams(root *model.Directory, discMetric metric.Name, fileMin, fileMax float64) discParams {
	dp := discParams{fileMin: fileMin, fileMax: fileMax}

	// Compute min/max in a single tree walk to avoid allocating an
	// intermediate []float64 slice that mirrors every file metric value.
	var (
		found  bool
		minVal float64
		maxVal float64
	)

	model.WalkFiles(root, func(f *model.File) {
		v := fileMetricValue(f, discMetric)
		if !(v > 0) {
			return
		}

		if !found {
			minVal, maxVal = v, v
			found = true

			return
		}

		if v < minVal {
			minVal = v
		}

		if v > maxVal {
			maxVal = v
		}
	})

	if !found {
		return dp
	}

	dp.metricMin = minVal
	dp.metricMax = maxVal
	dp.useEqual = minVal == maxVal

	return dp
}

func layoutDir(
	dir *model.Directory,
	depth int,
	startAngle, sweepAngle float64,
	opts layoutOptions,
) RadialNode {
	// Place this directory at the midpoint of its angular sector.
	angle := startAngle + sweepAngle/2
	radius := float64(depth) * opts.ringSpacing

	dirDisc := directoryDiscRadius(dir, opts.discParams.dir)

	node := RadialNode{
		X:           radius * math.Cos(angle),
		Y:           radius * math.Sin(angle),
		DiscRadius:  dirDisc,
		Angle:       angle,
		Label:       dir.Name,
		IsDirectory: true,
		ShowLabel:   opts.labels == LabelAll || opts.labels == LabelFoldersOnly,
	}

	allocationUnits := childAllocationUnits(dir, opts.grain)
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

	childRadius := float64(depth+1) * opts.ringSpacing
	fileSweep := contentSweep / float64(allocationUnits)

	// Files first: each file occupies one allocation unit of the padded sweep.
	for _, f := range visibleFiles(dir, opts.grain) {
		childAngle := childStart + fileSweep/2

		fileNode := RadialNode{
			X:           childRadius * math.Cos(childAngle),
			Y:           childRadius * math.Sin(childAngle),
			DiscRadius:  fileDiscRadius(f, opts.discMetric, opts.discParams),
			Angle:       childAngle,
			Label:       f.Name,
			IsDirectory: false,
			ShowLabel:   opts.labels == LabelAll,
		}

		node.Children = append(node.Children, fileNode)
		childStart += fileSweep
	}

	// Subdirs: each gets a proportional slice of the padded sweep based on its
	// file-leaf weight, with empty directories still reserving one unit.
	for _, d := range dir.Dirs {
		weight := childWeight(d, opts.grain)
		childSweep := float64(weight) / float64(allocationUnits) * contentSweep
		child := layoutDir(d, depth+1, childStart, childSweep, opts)
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

// buildDirDiscParams precomputes the scaling parameters for directory discs
// from the aggregated (rolled up) directory metric. When the metric is absent
// or uniform across the tree, directories fall back to a fixed base size.
func buildDirDiscParams(root *model.Directory, discMetric metric.Name, ringSpacing float64) dirDiscParams {
	base := math.Max(ringSpacing*dirDiscFactor, minDirDisc)
	dp := dirDiscParams{
		metricName: discMetric,
		base:       base,
		dirMin:     minDirDisc,
		dirMax:     base * maxDirDiscFactor,
	}

	if discMetric == "" {
		return dp
	}

	var minVal, maxVal float64

	model.WalkDirectories(root, func(d *model.Directory) {
		v := directoryMetricValue(d, discMetric)
		if !(v > 0) {
			return
		}

		if !dp.found {
			minVal, maxVal = v, v
			dp.found = true

			return
		}

		minVal = math.Min(minVal, v)
		maxVal = math.Max(maxVal, v)
	})

	if minVal == maxVal {
		// No values, or every directory has the same value: nothing to scale.
		dp.found = false

		return dp
	}

	dp.metricMin, dp.metricMax = minVal, maxVal

	return dp
}

// directoryDiscRadius returns the disc pixel radius for dir, scaled so that
// disc area varies linearly with the rolled up directory metric.
func directoryDiscRadius(dir *model.Directory, dp dirDiscParams) float64 {
	if !dp.found {
		return dp.base
	}

	val := directoryMetricValue(dir, dp.metricName)
	if val <= 0 {
		return dp.dirMin
	}

	fraction := (val - dp.metricMin) / (dp.metricMax - dp.metricMin)
	minArea := dp.dirMin * dp.dirMin
	area := minArea + fraction*(dp.dirMax*dp.dirMax-minArea)

	return clamp(math.Sqrt(area), dp.dirMin, dp.dirMax)
}

// directoryMetricValue returns the disc-metric value for dir as a float64.
// Quantity is checked first (int64), then Measure (float64). Returns 0 if absent.
func directoryMetricValue(dir *model.Directory, discMetric metric.Name) float64 {
	if q, ok := dir.Quantity(discMetric); ok {
		return float64(q)
	}

	if m, ok := dir.Measure(discMetric); ok {
		return m
	}

	return 0
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

// visibleFiles returns the files of dir that are laid out for the given grain.
// Directory grain omits files entirely, leaving only the folder structure.
func visibleFiles(dir *model.Directory, grain Grain) []*model.File {
	if grain == GrainDirectory {
		return nil
	}

	return dir.Files
}

// computeLeafCount returns the total number of leaves under dir: file leaves
// for file grain, or leaf directories for directory grain.
// Returns 0 for empty directories; callers are responsible for handling the
// zero case to avoid division by zero in sector calculations.
func computeLeafCount(dir *model.Directory, grain Grain) int {
	if grain == GrainDirectory && len(dir.Dirs) == 0 {
		return 1
	}

	count := len(visibleFiles(dir, grain))
	for _, d := range dir.Dirs {
		count += computeLeafCount(d, grain)
	}

	return count
}

func childAllocationUnits(dir *model.Directory, grain Grain) int {
	units := len(visibleFiles(dir, grain))
	for _, d := range dir.Dirs {
		units += childWeight(d, grain)
	}

	return units
}

func childWeight(dir *model.Directory, grain Grain) int {
	leafCount := computeLeafCount(dir, grain)
	if leafCount == 0 {
		return 1
	}

	return leafCount
}

// computeMaxDepth returns the maximum depth of any node in the tree rooted at dir.
// Root is at depth 0; its direct children (files or dirs) are at depth 1, etc.
// Files are ignored when they are not laid out for the given grain.
func computeMaxDepth(dir *model.Directory, grain Grain) int {
	depth := 0

	if len(visibleFiles(dir, grain)) > 0 {
		depth = 1
	}

	for _, d := range dir.Dirs {
		if child := 1 + computeMaxDepth(d, grain); child > depth {
			depth = child
		}
	}

	return depth
}

// adjustedDiscFactor returns a maxDiscFactor scaled down so that n nodes
// fit on a ring of radius ringSpacing without their full-size discs overlapping.
// This ensures readable layout even when directories have many children.
func adjustedDiscFactor(
	n int,
	ringSpacing float64,
	//nolint:unparam // Constant used elsewhere for flexibility
	baseMaxDiscFactor float64,
) float64 {
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
