// Package treemap implements squarified treemap layout using the
// nikolaydubina/treemap library.
package treemap

import (
	"github.com/bevan/code-visualizer/internal/scan"
	"github.com/nikolaydubina/treemap/layout"
)

const (
	HeaderHeight  = 20.0 // pixels for directory header bar
	padding       = 4.0  // pixels between groups
	siblingGap    = 2.0  // pixels between sibling rectangles
	minFileSize   = 1.0  // minimum area for zero-size files (FR-013)
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

	// Collect all children (files + subdirs) with their areas
	type child struct {
		isDir   bool
		fileIdx int
		dirIdx  int
		area    float64
	}

	var children []child

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

	if len(children) == 0 {
		return rect
	}

	// Reserve space for header bar and padding
	contentBox := layout.Box{
		X: box.X + padding,
		Y: box.Y + HeaderHeight,
		W: box.W - 2*padding,
		H: box.H - HeaderHeight - padding,
	}

	if contentBox.W <= 0 || contentBox.H <= 0 {
		return rect
	}

	// Collect areas for squarify
	areas := make([]float64, len(children))
	for i, c := range children {
		areas[i] = c.area
	}

	boxes := layout.Squarify(contentBox, areas)

	for i, c := range children {
		b := boxes[i]
		// Inset each sibling rectangle by a small gap for visual separation
		b = insetBox(b, siblingGap/2)
		if c.isDir {
			childRect := layoutDir(dir.Dirs[c.dirIdx], b)
			rect.Children = append(rect.Children, childRect)
		} else {
			f := dir.Files[c.fileIdx]
			rect.Children = append(rect.Children, TreemapRectangle{
				X:     b.X,
				Y:     b.Y,
				W:     b.W,
				H:     b.H,
				Label: f.Name,
			})
		}
	}

	return rect
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
