// Package treemap implements squarified treemap layout using the
// nikolaydubina/treemap library.
package treemap

import (
	"github.com/nikolaydubina/treemap/layout"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	siblingGap  = 2.0
	minFileSize = 1.0
)

// boxToRect converts a third-party layout.Box into a geometry.Rect.
func boxToRect(box layout.Box) geometry.Rect {
	return geometry.RectFromPositionSize(
		geometry.Point{X: box.X, Y: box.Y},
		geometry.Size{Width: box.W, Height: box.H},
	)
}

// Layout computes a squarified treemap layout from a Directory tree.
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name) TreemapRectangle {
	box := layout.Box{X: 0, Y: 0, W: float64(width), H: float64(height)}

	return layoutRoot(root, box, sizeMetric)
}

func layoutRoot(root *model.Directory, box layout.Box, sizeMetric metric.Name) TreemapRectangle {
	return layoutDirectory(root, box, sizeMetric, -1, directoryChromeBorderOnly(box))
}

func layoutDir(dir *model.Directory, box layout.Box, sizeMetric metric.Name, visibleDepth int) TreemapRectangle {
	return layoutDirectory(dir, box, sizeMetric, visibleDepth, resolveDirectoryChrome(box, dir.Name))
}

func layoutDirectory(
	dir *model.Directory,
	box layout.Box,
	sizeMetric metric.Name,
	visibleDepth int,
	chrome DirectoryChrome,
) TreemapRectangle {
	rect := TreemapRectangle{
		Bounds:       boxToRect(box),
		Label:        dir.Name,
		IsDirectory:  true,
		VisibleDepth: visibleDepth,
	}
	rect.Chrome = chrome

	children := collectChildren(dir, sizeMetric)
	if len(children) == 0 {
		return rect
	}

	// Recomputed from box (not derived from rect.Chrome.Content) so Squarify
	// receives the same floating-point values as chrome resolution used,
	// with no Rect Min/Max round-trip to perturb descendant layout.
	contentBox := directoryContentBox(box, chrome.Orientation)
	if contentBox.W <= 0 || contentBox.H <= 0 {
		return rect
	}

	areas := make([]float64, len(children))
	for i, c := range children {
		areas[i] = c.area
	}

	boxes := layout.Squarify(contentBox, areas)

	rect.Children = make([]TreemapRectangle, 0, len(children))

	for i, c := range children {
		b := insetBox(boxes[i], siblingGap/2)
		rect.Children = append(rect.Children, layoutChild(dir, c, b, sizeMetric, visibleDepth))
	}

	return rect
}

type child struct {
	isDir   bool
	fileIdx int
	dirIdx  int
	area    float64
}

func collectChildren(dir *model.Directory, sizeMetric metric.Name) []child {
	children := make([]child, 0, len(dir.Files)+len(dir.Dirs))

	for i, f := range dir.Files {
		area := fileSize(f, sizeMetric)
		if area <= 0 {
			area = minFileSize
		}

		children = append(children, child{isDir: false, fileIdx: i, area: area})
	}

	for i, d := range dir.Dirs {
		area := dirTotalSize(d, sizeMetric)
		if area <= 0 {
			area = minFileSize
		}

		children = append(children, child{isDir: true, dirIdx: i, area: area})
	}

	return children
}

// fileSize returns the size-metric value for f as a float64.
// Quantity is checked first, then Measure. Returns 0 if absent.
func fileSize(f *model.File, sizeMetric metric.Name) float64 {
	if v, ok := f.Quantity(sizeMetric); ok {
		return float64(v)
	}

	if v, ok := f.Measure(sizeMetric); ok {
		return v
	}

	return 0
}

func layoutChild(
	dir *model.Directory,
	c child,
	b layout.Box,
	sizeMetric metric.Name,
	parentVisibleDepth int,
) TreemapRectangle {
	if c.isDir {
		return layoutDir(dir.Dirs[c.dirIdx], b, sizeMetric, parentVisibleDepth+1)
	}

	f := dir.Files[c.fileIdx]

	return TreemapRectangle{
		Bounds: boxToRect(b),
		Label:  f.Name,
	}
}

func insetBox(b layout.Box, inset float64) layout.Box {
	if b.W <= 2*inset || b.H <= 2*inset {
		return b
	}

	return layout.Box{
		X: b.X + inset, Y: b.Y + inset,
		W: b.W - 2*inset, H: b.H - 2*inset,
	}
}

// OffsetRects shifts all rectangle coordinates by the provided offset, recursively
// adjusting every child in the tree.
func OffsetRects(rect *TreemapRectangle, offset geometry.Vector) {
	rect.Bounds = rect.Bounds.Translate(offset)

	if rect.IsDirectory {
		if rect.Chrome.Orientation != DirectoryLabelNone {
			rect.Chrome.Rail = rect.Chrome.Rail.Translate(offset)
		}

		rect.Chrome.Content = rect.Chrome.Content.Translate(offset)
	}

	for i := range rect.Children {
		OffsetRects(&rect.Children[i], offset)
	}
}

func dirTotalSize(dir *model.Directory, sizeMetric metric.Name) float64 {
	var total float64

	for _, f := range dir.Files {
		s := fileSize(f, sizeMetric)
		if s <= 0 {
			s = minFileSize
		}

		total += s
	}

	for _, d := range dir.Dirs {
		total += dirTotalSize(d, sizeMetric)
	}

	return total
}
