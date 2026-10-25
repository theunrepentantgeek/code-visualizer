// Package bubbletree implements data types and layout algorithms
// for circle-packing bubble tree visualizations.
package bubbletree

import (
	"cmp"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	minFileRadius    = 2.0                   // minimum circle radius for any file node
	siblingPadding   = 3.0                   // gap between sibling circles at the same level
	parentPadding    = 6.0                   // inset from parent circle edge
	LabelReservation = bubbleDefaultFontSize // occupied radius reserved above labelled directory bubbles
	labelFitSteps    = 20
)

// Layout builds a bubble tree from root, positioning circles to fit within
// width × height pixels. sizeMetric controls the relative size of file circles
// and labels controls which labels are shown.
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	if root == nil {
		return BubbleNode{}
	}

	node := layoutRoot(root, sizeMetric, labels)
	scaleToFit(&node, float64(width), float64(height))

	if labels != LabelNone && len(root.Dirs) > 0 && node.layoutScale > 0 && node.layoutScale < 1 {
		node = fitReadableLabels(width, height, node)
	}

	return node
}

func layoutRoot(dir *model.Directory, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	node := layoutDir(dir, sizeMetric, labels)

	// The root directory's disc is never rendered, so its label is never
	// shown either. Strip the label reservation so the children fill the
	// canvas instead of leaving whitespace for a label that won't appear.
	if node.ShowLabel {
		node.Geometry.Radius -= LabelReservation
		node.ShowLabel = false
	}

	return node
}

// fitReadableLabels budgets readable labels largest-first, then fits proportional
// file radii around them. Gaps retain their original scale so tiny directories do
// not exhaust the canvas before larger directories can receive labels.
func fitReadableLabels(width, height int, original BubbleNode) BubbleNode {
	candidates := collectReadableLabelCandidates(&original, nil)
	if len(candidates) == 0 {
		return original
	}

	slices.SortStableFunc(candidates, func(a, b *BubbleNode) int {
		return cmp.Compare(b.Geometry.Radius, a.Geometry.Radius)
	})

	ranks := make(map[*BubbleNode]int, len(candidates))
	for i, node := range candidates {
		ranks[node] = i
	}

	baseScale := original.layoutScale
	makeCandidate := func(fileScale float64, labelCount int) BubbleNode {
		node := repackReadableLabels(&original, baseScale, fileScale, labelCount, ranks)
		scaleToFit(&node, float64(width), float64(height))

		return node
	}

	// Keep every file radius positive, including while finding the label budget.
	minFileScale := 1 / float64(1<<labelFitSteps)
	lowLabels, highLabels := 0, len(candidates)

	for lowLabels < highLabels {
		count := (lowLabels + highLabels + 1) / 2
		candidate := makeCandidate(minFileScale, count)

		if candidate.layoutScale >= baseScale {
			lowLabels = count
		} else {
			highLabels = count - 1
		}
	}

	if lowLabels == 0 {
		return original
	}

	low, high := minFileScale, 1.0
	best := makeCandidate(low, lowLabels)

	for range labelFitSteps {
		fileScale := (low + high) / 2
		candidate := makeCandidate(fileScale, lowLabels)

		if candidate.layoutScale < baseScale {
			high = fileScale
		} else {
			low = fileScale
			best = candidate
		}
	}

	return best
}

func collectReadableLabelCandidates(node *BubbleNode, candidates []*BubbleNode) []*BubbleNode {
	if node.IsDirectory && node.ShowLabel && node.Label != "" &&
		(bubbleDirLabelFontSize(*node) > 0 ||
			node.Geometry.Radius >= LabelReservation+bubbleMinLabelDiscRadius(node.Label)) {
		candidates = append(candidates, node)
	}

	for i := range node.Children {
		candidates = collectReadableLabelCandidates(&node.Children[i], candidates)
	}

	return candidates
}

func repackReadableLabels(
	original *BubbleNode,
	baseScale, fileScale float64,
	labelCount int,
	ranks map[*BubbleNode]int,
) BubbleNode {
	node := *original
	if !node.IsDirectory {
		node.Geometry.Radius *= fileScale / baseScale

		return node
	}

	node.Children = make([]BubbleNode, len(original.Children))
	for i := range original.Children {
		node.Children[i] = repackReadableLabels(&original.Children[i], baseScale, fileScale, labelCount, ranks)
	}

	rank, eligible := ranks[original]
	node.ShowLabel = eligible && rank < labelCount
	node.labelBand = 0

	if node.ShowLabel {
		node.labelBand = LabelReservation / baseScale
	}

	node = packDirectory(node)
	if node.ShowLabel {
		node.Geometry.Radius = max(node.Geometry.Radius,
			node.labelBand+bubbleMinLabelDiscRadius(node.Label)/baseScale)
	}

	return node
}

// layoutDir recursively builds a BubbleNode for dir. Children are packed
// using the front-chain algorithm and enclosed. All coordinates are relative
// to the parent centre (local frame).
func layoutDir(dir *model.Directory, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	children := make([]BubbleNode, 0, len(dir.Dirs)+len(dir.Files))

	for _, d := range dir.Dirs {
		children = append(children, layoutDir(d, sizeMetric, labels))
	}

	for _, f := range dir.Files {
		children = append(children, layoutFile(f, sizeMetric, labels))
	}

	node := BubbleNode{
		Path:        dir.Path,
		Label:       dir.Name,
		IsDirectory: true,
		ShowLabel:   labels == LabelAll || labels == LabelFoldersOnly,
		Children:    children,
	}
	if node.ShowLabel {
		node.labelBand = LabelReservation
	}

	return packDirectory(node)
}

func packDirectory(node BubbleNode) BubbleNode {
	children := node.Children
	if len(children) == 0 {
		node.Geometry.Radius = minFileRadius
		if node.ShowLabel {
			node.Geometry.Radius += node.labelBand
		}

		return node
	}

	// Sort by radius descending — improves packing density.
	slices.SortFunc(children, func(a, b BubbleNode) int {
		return cmp.Compare(b.Geometry.Radius, a.Geometry.Radius)
	})

	packCircles(children)

	enc := computeEnclosing(children)

	// Re-centre so the enclosing circle's centre becomes local origin.
	offset := enc.Center.VectorTo(geometry.OriginPoint)
	for i := range children {
		children[i].Geometry.Center = children[i].Geometry.Center.Translate(offset)
	}

	node.Geometry.Radius = enc.Radius + parentPadding
	if node.ShowLabel {
		node.Geometry.Radius += node.labelBand
	}

	return node
}

func layoutFile(f *model.File, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	r := math.Sqrt(fileMetricValue(f, sizeMetric))
	if r < minFileRadius {
		r = minFileRadius
	}

	return BubbleNode{
		Geometry:  geometry.NewCircle(geometry.OriginPoint, r),
		Path:      f.Path,
		Label:     f.Name,
		ShowLabel: labels == LabelAll,
	}
}

// fileMetricValue returns the metric value for f as a float64.
// Quantity is checked first, then Measure. Returns 0 if absent.
func fileMetricValue(f *model.File, m metric.Name) float64 {
	if q, ok := f.Quantity(m); ok {
		return float64(q)
	}

	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}
