package treemap

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// RenderToCanvas walks the layout tree and model tree in parallel,
// adding shapes to the canvas. Returns the populated canvas.
func RenderToCanvas(
	rects TreemapRectangle,
	root *model.Directory,
	width, height int,
	inks Inks,
	sizeMetric metric.Name,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	// Background
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        pkginks.FixedInk(palette.White),
			Border:      pkginks.FixedInk(palette.White),
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

	addRect(cv, rects, root, inks, sizeMetric)

	return cv
}

// addRect recursively adds shapes for a single treemap node.
func addRect(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	node *model.Directory,
	inks Inks,
	sizeMetric metric.Name,
) {
	if !rect.IsDirectory {
		addFileRectForFile(cv, rect, nil, inks, rect, 0)

		return
	}

	addDirectoryShapes(cv, rect)

	dirTotal := directoryTotalWeight(node, sizeMetric)
	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addRect(cv, child, node.Dirs[dirIdx], inks, sizeMetric)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			fileWeight := fileMetricWeight(node.Files[fileIdx], sizeMetric)
			addFileRectForFile(cv, child, node.Files[fileIdx], inks, rect, fileWeight/dirTotal)
			fileIdx++
		}
	}
}

func addDirectoryShapes(
	cv *canvas.Canvas,
	rect TreemapRectangle,
) {
	// Header bar fill
	headerSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        pkginks.FixedInk(headerFill),
			Border:      pkginks.FixedInk(headerFill),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  headerSpec,
		X:     rect.X,
		Y:     rect.Y,
		W:     rect.W,
		H:     headerHeight,
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})

	// Header label
	if rect.Label != "" {
		labelSpec := &canvas.TextSpec{
			Ink:      pkginks.FixedInk(palette.White),
			FontSize: 0,
			Anchor:   canvas.AnchorStart,
		}
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    labelSpec,
			X:       rect.X + 4,
			Y:       rect.Y + headerHeight/2,
			Content: rect.Label,
		})
	}

	// Directory border
	borderSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        pkginks.FixedInk(color.RGBA{A: 0}),
			Border:      pkginks.FixedInk(structuralBorder),
			BorderWidth: DynBorderWidth(rect.W, rect.H, pkginks.KindNumeric),
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
	inks Inks,
	parentDir TreemapRectangle,
	weightFraction float64,
) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	focus := computeFocus(rect, parentDir, weightFraction)
	hasBorder := inks.Border.Info().Kind
	fillMV := pkginks.MetricValueForFile(file, inks.Fill)
	borderMV := pkginks.MetricValueForFile(file, inks.Border)

	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.Fill,
			Border:      inks.Border,
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
func DynBorderWidth(w, h float64, borderKind pkginks.Kind) float64 {
	if borderKind == pkginks.KindFixed {
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
