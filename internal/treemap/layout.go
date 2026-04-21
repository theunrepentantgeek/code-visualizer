// Package treemap implements squarified treemap layout using the
// nikolaydubina/treemap library.
package treemap

import (
	"github.com/nikolaydubina/treemap/layout"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

const (
	HeaderHeight = 20.0
	padding      = 4.0
	siblingGap   = 2.0
	minFileSize  = 1.0
)

// Layout computes a squarified treemap layout from a Directory tree.
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name) TreemapRectangle {
	box := layout.Box{X: 0, Y: 0, W: float64(width), H: float64(height)}

	return layoutDir(root, box, sizeMetric)
}

func layoutDir(dir *model.Directory, box layout.Box, sizeMetric metric.Name) TreemapRectangle {
	rect := TreemapRectangle{
		X: box.X, Y: box.Y, W: box.W, H: box.H,
		Label: dir.Name, IsDirectory: true,
	}

	children := collectChildren(dir, sizeMetric)
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
		rect.Children = append(rect.Children, layoutChild(dir, c, b, sizeMetric))
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

func fileSize(f *model.File, sizeMetric metric.Name) float64 {
	v, ok := f.Quantity(sizeMetric)
	if !ok {
		return 0
	}

	return float64(v)
}

func contentArea(box layout.Box) layout.Box {
	return layout.Box{
		X: box.X + padding,
		Y: box.Y + HeaderHeight,
		W: box.W - 2*padding,
		H: box.H - HeaderHeight - padding,
	}
}

func layoutChild(dir *model.Directory, c child, b layout.Box, sizeMetric metric.Name) TreemapRectangle {
	if c.isDir {
		return layoutDir(dir.Dirs[c.dirIdx], b, sizeMetric)
	}

	f := dir.Files[c.fileIdx]

	return TreemapRectangle{
		X: b.X, Y: b.Y, W: b.W, H: b.H,
		Label: f.Name,
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

// OffsetRects shifts all rectangle coordinates by (dx, dy), recursively
// adjusting every child in the tree.
func OffsetRects(rect *TreemapRectangle, dx, dy float64) {
	rect.X += dx
	rect.Y += dy

	for i := range rect.Children {
		OffsetRects(&rect.Children[i], dx, dy)
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
