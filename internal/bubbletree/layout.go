// Package bubbletree implements data types and layout algorithms
// for circle-packing bubble tree visualizations.
package bubbletree

import (
	"cmp"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	minFileRadius    = 2.0                   // minimum circle radius for any file node
	siblingPadding   = 3.0                   // gap between sibling circles at the same level
	parentPadding    = 6.0                   // inset from parent circle edge
	LabelReservation = bubbleDefaultFontSize // occupied radius reserved above labelled directory bubbles
)

// Layout builds a bubble tree from root, positioning circles to fit within
// width × height pixels. sizeMetric controls the relative size of file circles
// and labels controls which labels are shown.
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	if root == nil {
		return BubbleNode{}
	}

	node := layoutDir(root, sizeMetric, labels)

	// The root directory's disc is never rendered, so its label is never
	// shown either. Strip the label reservation so the children fill the
	// canvas instead of leaving whitespace for a label that won't appear.
	if node.ShowLabel {
		node.Radius -= LabelReservation
		node.ShowLabel = false
	}

	scaleToFit(&node, float64(width), float64(height))

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
	}

	if len(children) == 0 {
		node.Radius = minFileRadius
		if node.ShowLabel {
			node.Radius += LabelReservation
		}

		return node
	}

	// Sort by radius descending — improves packing density.
	slices.SortFunc(children, func(a, b BubbleNode) int {
		return cmp.Compare(b.Radius, a.Radius)
	})

	packCircles(children)

	enc := computeEnclosing(children)

	// Re-centre so the enclosing circle's centre becomes local origin.
	for i := range children {
		children[i].X -= enc.x
		children[i].Y -= enc.y
	}

	node.Radius = enc.radius + parentPadding
	if node.ShowLabel {
		node.Radius += LabelReservation
	}

	node.Children = children

	return node
}

func layoutFile(f *model.File, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	r := math.Sqrt(fileMetricValue(f, sizeMetric))
	if r < minFileRadius {
		r = minFileRadius
	}

	return BubbleNode{
		Radius:    r,
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
