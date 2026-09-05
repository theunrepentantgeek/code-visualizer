package treemap

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Directory chrome specs are constant across every directory node in a render
// pass. Pre-allocating them avoids repeated heap allocations in the recursive
// walk.
//
//nolint:gochecknoglobals // pre-allocated render-phase specs
var (
	dirTopLabelSpec = &canvas.TextSpec{
		Ink:      inks.FixedInk(palette.White),
		FontSize: directoryLabelFontSize,
		Anchor:   canvas.AnchorMiddle,
		Rotation: 0,
	}
	dirLeftLabelSpec = &canvas.TextSpec{
		Ink:      inks.FixedInk(palette.White),
		FontSize: directoryLabelFontSize,
		Anchor:   canvas.AnchorMiddle,
		Rotation: -math.Pi / 2,
	}
	dirBorderFillInk = inks.FixedInk(color.RGBA{A: 0})
	dirBorderLineInk = inks.FixedInk(structuralBorder)
)

// dynBorderWidths lists every value DynBorderWidth can return, in ascending order.
// The index maps directly to specIndex().
//
//nolint:gochecknoglobals // read-only lookup table for DynBorderWidth results
var dynBorderWidths = [4]float64{0.5, 1.0, 2.0, 3.0}

// specIndex returns the pre-allocation table index for a DynBorderWidth result.
// Returns -1 for unexpected values.
func specIndex(bw float64) int {
	for i, w := range dynBorderWidths {
		if bw == w {
			return i
		}
	}

	return -1
}

// dirBorderSpecs holds one pre-allocated RectangleSpec per possible border width
// (indices 0–3 correspond to dynBorderWidths). Avoids a per-directory allocation.
// Built once per render pass in buildDirBorderSpecs.
type dirBorderSpecs [4]*canvas.RectangleSpec

func buildDirBorderSpecs() dirBorderSpecs {
	var table dirBorderSpecs
	for i, bw := range dynBorderWidths {
		table[i] = &canvas.RectangleSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        dirBorderFillInk,
				Border:      dirBorderLineInk,
				BorderWidth: bw,
			},
		}
	}

	return table
}

// dirRailSpecs holds one pre-allocated RectangleSpec per depth-palette colour
// (indices 0-len(headerFills)-1). Avoids a per-directory allocation.
// Built once per render pass in buildDirRailSpecs.
type dirRailSpecs [len(headerFills)]*canvas.RectangleSpec

func buildDirRailSpecs() dirRailSpecs {
	var table dirRailSpecs

	for i, fill := range headerFills {
		ink := inks.FixedInk(fill)
		table[i] = &canvas.RectangleSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        ink,
				Border:      ink,
				BorderWidth: 0,
			},
		}
	}

	return table
}

// dirRailSpecForDepth selects the pre-allocated rail spec for a directory's
// VisibleDepth, wrapping around the palette every len(specs) levels. Total
// over all ints: negative depths (notably the root's VisibleDepth of -1)
// clamp to specs[0], the darkest fill, rather than panicking or wrapping via
// Go's negative-operand modulo.
func dirRailSpecForDepth(specs dirRailSpecs, visibleDepth int) *canvas.RectangleSpec {
	if visibleDepth < 0 {
		return specs[0]
	}

	return specs[visibleDepth%len(specs)]
}

// fileRectSpecs holds one pre-allocated RectangleSpec per possible border width,
// sharing a single set of Fill and Border inks across the whole render pass.
// Built once per render pass in buildFileRectSpecs.
type fileRectSpecs [4]*canvas.RectangleSpec

func buildFileRectSpecs(is Inks) fileRectSpecs {
	var table fileRectSpecs
	for i, bw := range dynBorderWidths {
		table[i] = &canvas.RectangleSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        is.Fill,
				Border:      is.Border,
				BorderWidth: bw,
			},
		}
	}

	return table
}

// RenderToCanvas walks the layout tree and model tree in parallel,
// adding shapes to the canvas. Returns the populated canvas.
func RenderToCanvas(
	rects TreemapRectangle,
	root *model.Directory,
	width, height int,
	is Inks,
	sizeMetric metric.Name,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	// Background
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(palette.White),
			Border:      inks.FixedInk(palette.White),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec:   bgSpec,
		Bounds: geometry.Rect{Max: geometry.NewPoint(float64(width), float64(height))},
		Focus:  canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
	})

	dirSpecs := buildDirBorderSpecs()
	fileSpecs := buildFileRectSpecs(is)
	railSpecs := buildDirRailSpecs()
	addRect(cv, rects, root, is, sizeMetric, dirSpecs, fileSpecs, railSpecs)

	return cv
}

// addRect recursively adds shapes for a single treemap node.
func addRect(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	node *model.Directory,
	is Inks,
	sizeMetric metric.Name,
	dirSpecs dirBorderSpecs,
	fileSpecs fileRectSpecs,
	railSpecs dirRailSpecs,
) {
	if !rect.IsDirectory {
		addFileRectForFile(cv, rect, nil, is, rect, 0, fileSpecs)

		return
	}

	addDirectoryShapes(cv, rect, dirSpecs, railSpecs)

	dirTotal := directoryTotalWeight(node, sizeMetric)
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addRect(cv, child, node.Dirs[dirIdx], is, sizeMetric, dirSpecs, fileSpecs, railSpecs)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			fileWeight := fileMetricWeight(node.Files[fileIdx], sizeMetric)
			addFileRectForFile(cv, child, node.Files[fileIdx], is, rect, fileWeight/dirTotal, fileSpecs)
			fileIdx++
		}
	}
}

func addDirectoryShapes(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	dirSpecs dirBorderSpecs,
	railSpecs dirRailSpecs,
) {
	if rect.Chrome.Orientation != DirectoryLabelNone {
		rail := rect.Chrome.Rail
		cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
			Spec:   dirRailSpecForDepth(railSpecs, rect.VisibleDepth),
			Bounds: rail,
			Focus:  canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
		})
	}

	if rect.Chrome.Orientation != DirectoryLabelNone && rect.Chrome.Text != "" {
		spec := dirTopLabelSpec
		rail := rect.Chrome.Rail
		center := rail.Center()

		if rect.Chrome.Orientation == DirectoryLabelLeft {
			spec = dirLeftLabelSpec
		}

		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:     spec,
			Position: center,
			Content:  rect.Chrome.Text,
		})
	}

	// Directory border - BorderWidth varies per directory; look up the pre-allocated spec.
	size := rect.size()
	bw := DynBorderWidth(size.Width, size.Height, inks.KindNumeric)
	idx := specIndex(bw)

	var borderSpec *canvas.RectangleSpec
	if idx >= 0 {
		borderSpec = dirSpecs[idx]
	} else {
		borderSpec = &canvas.RectangleSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        dirBorderFillInk,
				Border:      dirBorderLineInk,
				BorderWidth: bw,
			},
		}
	}

	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:   borderSpec,
		Bounds: rect.Bounds,
		Focus:  canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
	})
}

func addFileRectForFile(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	file *model.File,
	is Inks,
	parentDir TreemapRectangle,
	weightFraction float64,
	fileSpecs fileRectSpecs,
) {
	if rect.Bounds.Width() <= 0 || rect.Bounds.Height() <= 0 {
		return
	}

	focus := computeFocus(rect, parentDir, weightFraction)
	hasBorder := is.Border.Info().Kind
	fillMV := inks.MetricValueForFile(file, is.Fill)
	borderMV := inks.MetricValueForFile(file, is.Border)

	size := rect.size()
	bw := DynBorderWidth(size.Width, size.Height, hasBorder)
	idx := specIndex(bw)

	var spec *canvas.RectangleSpec
	if idx >= 0 {
		spec = fileSpecs[idx]
	} else {
		spec = &canvas.RectangleSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        is.Fill,
				Border:      is.Border,
				BorderWidth: bw,
			},
		}
	}

	cv.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec:   spec,
		Bounds: rect.Bounds,
		Fill:   fillMV,
		Border: borderMV,
		Focus:  focus,
	})
}

func computeFocus(fileRect, dirRect TreemapRectangle, weightFraction float64) canvasmodel.GradientPoint {
	if fileRect.Bounds.Width() <= 0 || fileRect.Bounds.Height() <= 0 {
		return canvasmodel.GradientPoint{X: 0.5, Y: 0.5}
	}

	fileSize := fileRect.size()
	dirSize := dirRect.size()
	fileCX := fileRect.Bounds.Min.X + fileSize.Width/2
	fileCY := fileRect.Bounds.Min.Y + fileSize.Height/2
	dirCX := dirRect.Bounds.Min.X + dirSize.Width/2
	dirCY := dirRect.Bounds.Min.Y + dirSize.Height/2
	focusX := fileCX + (dirCX-fileCX)*weightFraction
	focusY := fileCY + (dirCY-fileCY)*weightFraction

	// Normalize against the Rect extent consumed by the canvas backend so it
	// reconstructs the focus computed from the original layout-box sizes.
	return canvasmodel.GradientPoint{
		X: (focusX - fileRect.Bounds.Min.X) / fileRect.Bounds.Width(),
		Y: (focusY - fileRect.Bounds.Min.Y) / fileRect.Bounds.Height(),
	}
}

func directoryTotalWeight(dir *model.Directory, sizeMetric metric.Name) float64 {
	total := 0.0
	for _, f := range dir.Files {
		total += fileMetricWeight(f, sizeMetric)
	}

	if total <= 0 {
		total = float64(len(dir.Files))
	}

	return total
}

func fileMetricWeight(file *model.File, sizeMetric metric.Name) float64 {
	if file == nil || sizeMetric == "" {
		return 1.0
	}

	if v, ok := file.Quantity(sizeMetric); ok {
		return float64(v)
	}

	if v, ok := file.Measure(sizeMetric); ok {
		return v
	}

	return 1.0
}

// DynBorderWidth returns a dynamic border width based on rectangle
// size and the kind of border ink configured.
func DynBorderWidth(w, h float64, borderKind inks.Kind) float64 {
	if borderKind == inks.KindFixed {
		return 0.5
	}

	minDim := min(w, h)

	switch {
	case minDim < minBorderDim:
		return 1.0
	case minDim >= midBorderDim:
		return 3.0
	default:
		return 2.0
	}
}
