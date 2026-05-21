package treemap

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// RenderToCanvas walks the layout tree and model tree in parallel,
// adding shapes to the canvas. Returns the populated canvas.
func RenderToCanvas(
	rects TreemapRectangle,
	root *model.Directory,
	width, height int,
	inks Inks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	// Background
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(bgColour),
			Border:      canvas.FixedInk(bgColour),
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

	addRect(cv, rects, root, inks)

	return cv
}

// addRect recursively adds shapes for a single treemap node.
func addRect(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	node *model.Directory,
	inks Inks,
) {
	if !rect.IsDirectory {
		addFileRectForFile(cv, rect, nil, inks)

		return
	}

	addDirectoryShapes(cv, rect)

	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addRect(cv, child, node.Dirs[dirIdx], inks)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			addFileRectForFile(cv, child, node.Files[fileIdx], inks)
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
			Fill:        canvas.FixedInk(headerFill),
			Border:      canvas.FixedInk(headerFill),
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
			Ink:      canvas.FixedInk(whiteText),
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
			Fill:        canvas.FixedInk(color.RGBA{A: 0}),
			Border:      canvas.FixedInk(structuralBorder),
			BorderWidth: DynBorderWidth(rect.W, rect.H, canvas.InkNumeric),
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
) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

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
		Focus:  canvasmodel.Point{X: 0.5, Y: 0.5},
	})

	if rect.Label != "" && rect.W >= 40 && rect.H >= 16 {
		fillColour := inks.Fill.Dip(fillMV)
		labelColour := canvas.TextColourFor(fillColour)

		labelSpec := &canvas.TextSpec{
			Ink:      canvas.FixedInk(labelColour),
			FontSize: 0,
			Anchor:   canvas.AnchorMiddle,
		}
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    labelSpec,
			X:       rect.X + rect.W/2,
			Y:       rect.Y + rect.H/2,
			Content: rect.Label,
		})
	}
}

// DynBorderWidth returns a dynamic border width based on rectangle
// size and the kind of border ink configured.
func DynBorderWidth(w, h float64, borderKind canvas.InkKind) float64 {
	if borderKind == canvas.InkFixed {
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
