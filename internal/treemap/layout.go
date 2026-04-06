// Package treemap implements squarified treemap layout using the
// nikolaydubina/treemap library.
package treemap

import (
	"github.com/nikolaydubina/treemap/layout"

	"github.com/bevan/code-visualizer/internal/scan"
)

const (
	HeaderHeight = 20.0 // pixels for directory header bar
	padding      = 4.0  // pixels between groups
	siblingGap   = 2.0  // pixels between sibling rectangles
	minFileSize  = 1.0  // minimum area for zero-size files (FR-013)
)

// Layout computes a squarified treemap layout from a DirectoryNode tree.
// Returns a root TreemapRectangle with nested children.
func Layout(root scan.DirectoryNode, width, height int) TreemapRectangle {
	box := layout.Box{X: 0, Y: 0, W: float64(width), H: float64(height)}
	rect := layoutDir(root, box)

	return rect
}

func layoutDir(dir scan.DirectoryNode, box layout.Box) TreemapRectangle {
	rect := TreemapRectangle{
		X:           box.X,
		Y:           box.Y,
		W:           box.W,
		H:           box.H,
		Label:       dir.Name,
		IsDirectory: true,
	}

	children := collectChildren(dir)

	if len(children) == 0 {
		return rect
	}

	contentBox := contentArea(box)
	if contentBox.W <= 0 || contentBox.H <= 0 {
		return rect
	}

	areas := make([]float64, len(children))
	for i, c := range children {
		areas[i] = c.area
	}

	boxes := layout.Squarify(contentBox, areas)

	for i, c := range children {
		b := insetBox(boxes[i], siblingGap/2)
		rect.Children = append(rect.Children, layoutChild(dir, c, b))
	}

	return rect
}

type child struct {
	isDir   bool
	fileIdx int
	dirIdx  int
	area    float64
}

func collectChildren(dir scan.DirectoryNode) []child {
	children := make([]child, 0, len(dir.Files)+len(dir.Dirs))

	for i, f := range dir.Files {
		area := float64(f.Size)
		if area <= 0 {
			area = minFileSize
		}

		children = append(children, child{isDir: false, fileIdx: i, area: area})
	}

	for i, d := range dir.Dirs {
		area := dirTotalSize(d)
		if area <= 0 {
			area = minFileSize
		}

		children = append(children, child{isDir: true, dirIdx: i, area: area})
	}

	return children
}

func contentArea(box layout.Box) layout.Box {
	return layout.Box{
		X: box.X + padding,
		Y: box.Y + HeaderHeight,
		W: box.W - 2*padding,
		H: box.H - HeaderHeight - padding,
	}
}

func layoutChild(dir scan.DirectoryNode, c child, b layout.Box) TreemapRectangle {
	if c.isDir {
		return layoutDir(dir.Dirs[c.dirIdx], b)
	}

	f := dir.Files[c.fileIdx]

	return TreemapRectangle{
		X:     b.X,
		Y:     b.Y,
		W:     b.W,
		H:     b.H,
		Label: f.Name,
	}
}

func insetBox(b layout.Box, inset float64) layout.Box {
	// Only apply inset if the box is large enough to remain positive
	if b.W <= 2*inset || b.H <= 2*inset {
		return b
	}

	return layout.Box{
		X: b.X + inset,
		Y: b.Y + inset,
		W: b.W - 2*inset,
		H: b.H - 2*inset,
	}
}

func dirTotalSize(dir scan.DirectoryNode) float64 {
	var total float64

	for _, f := range dir.Files {
		s := float64(f.Size)
		if s <= 0 {
			s = minFileSize
		}

		total += s
	}

	for _, d := range dir.Dirs {
		total += dirTotalSize(d)
	}

	return total
}
