package treemap

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// dirHeaderSpec and dirLabelSpec are constant across every directory node in a
// render pass. Pre-allocating them avoids repeated heap allocations in the
// recursive walk.
//
//nolint:gochecknoglobals // pre-allocated render-phase specs
var (
	dirHeaderSpec = &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(headerFill),
			Border:      inks.FixedInk(headerFill),
			BorderWidth: 0,
		},
	}
	dirLabelSpec = &canvas.TextSpec{
		Ink:      inks.FixedInk(palette.White),
		FontSize: 0,
		Anchor:   canvas.AnchorStart,
	}
	dirBorderFillInk = inks.FixedInk(color.RGBA{A: 0})
	dirBorderLineInk = inks.FixedInk(structuralBorder)
)

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
		Spec:  bgSpec,
		X:     0,
		Y:     0,
		W:     float64(width),
		H:     float64(height),
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})

	addRect(cv, rects, root, is, sizeMetric)

	return cv
}

// addRect recursively adds shapes for a single treemap node.
func addRect(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	node *model.Directory,
	is Inks,
	sizeMetric metric.Name,
) {
	if !rect.IsDirectory {
		addFileRectForFile(cv, rect, nil, is, rect, 0)

		return
	}

	addDirectoryShapes(cv, rect)

	dirTotal := directoryTotalWeight(node, sizeMetric)
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addRect(cv, child, node.Dirs[dirIdx], is, sizeMetric)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			fileWeight := fileMetricWeight(node.Files[fileIdx], sizeMetric)
			addFileRectForFile(cv, child, node.Files[fileIdx], is, rect, fileWeight/dirTotal)
			fileIdx++
		}
	}
}

func addDirectoryShapes(
	cv *canvas.Canvas,
	rect TreemapRectangle,
) {
	// Header bar fill - spec is constant across all directories in this render pass.
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  dirHeaderSpec,
		X:     rect.X,
		Y:     rect.Y,
		W:     rect.W,
		H:     headerHeight,
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})

	// Header label - spec is constant; only position and content vary.
	if rect.Label != "" {
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    dirLabelSpec,
			X:       rect.X + 4,
			Y:       rect.Y + headerHeight/2,
			Content: rect.Label,
		})
	}

	// Directory border - BorderWidth varies per directory, so only the inks are shared.
	borderSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        dirBorderFillInk,
			Border:      dirBorderLineInk,
			BorderWidth: DynBorderWidth(rect.W, rect.H, inks.KindNumeric),
		},
	}
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  borderSpec,
		X:     rect.X,
		Y:     rect.Y,
		W:     rect.W,
		H:     rect.H,
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}

func addFileRectForFile(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	file *model.File,
	is Inks,
	parentDir TreemapRectangle,
	weightFraction float64,
) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	focus := computeFocus(rect, parentDir, weightFraction)
	hasBorder := is.Border.Info().Kind
	fillMV := inks.MetricValueForFile(file, is.Fill)
	borderMV := inks.MetricValueForFile(file, is.Border)

	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        is.Fill,
			Border:      is.Border,
			BorderWidth: DynBorderWidth(rect.W, rect.H, hasBorder),
		},
	}

	cv.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec:   spec,
		X:      rect.X,
		Y:      rect.Y,
		W:      rect.W,
		H:      rect.H,
		Fill:   fillMV,
		Border: borderMV,
		Focus:  focus,
	})
}

func computeFocus(fileRect, dirRect TreemapRectangle, weightFraction float64) canvasmodel.Point {
	if fileRect.W <= 0 || fileRect.H <= 0 {
		return canvasmodel.Point{X: 0.5, Y: 0.5}
	}

	fileCX := fileRect.X + fileRect.W/2
	fileCY := fileRect.Y + fileRect.H/2
	dirCX := dirRect.X + dirRect.W/2
	dirCY := dirRect.Y + dirRect.H/2
	focusX := fileCX + (dirCX-fileCX)*weightFraction
	focusY := fileCY + (dirCY-fileCY)*weightFraction

	return canvasmodel.Point{
		X: (focusX - fileRect.X) / fileRect.W,
		Y: (focusY - fileRect.Y) / fileRect.H,
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
