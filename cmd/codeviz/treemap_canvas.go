package main

import (
	"image/color"

	"github.com/bevan/code-visualizer/internal/canvas"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/treemap"
)

const (
	treemapHeaderHeight = treemap.HeaderHeight
	treemapMinBorderDim = 20.0
	treemapMidBorderDim = 100.0
)

var (
	treemapStructuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	treemapHeaderFill       = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	treemapDefaultFill      = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	treemapBgColour         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	treemapWhiteText        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// treemapInks holds the Ink instances for a treemap render pass.
type treemapInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildTreemapInks creates fill and border inks from metric configuration.
func buildTreemapInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) treemapInks {
	inks := treemapInks{
		fill:   canvas.FixedInk(treemapDefaultFill),
		border: canvas.FixedInk(treemapStructuralBorder),
	}

	inks.fill = buildMetricInk(root, fillMetric, fillPaletteName, treemapDefaultFill)
	if borderMetric != "" {
		inks.border = buildMetricInk(root, borderMetric, borderPaletteName, treemapStructuralBorder)
	}

	return inks
}

// buildMetricInk creates an Ink for a given metric, using the appropriate
// constructor based on the metric kind (numeric vs categorical).
func buildMetricInk(
	root *model.Directory,
	m metric.Name,
	palName palette.PaletteName,
	fallback color.RGBA,
) canvas.Ink {
	p, ok := provider.Get(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, m)
		if len(values) == 0 {
			return canvas.FixedInk(fallback)
		}

		return canvas.NumericInk(m, values, pal)
	}

	types := collectDistinctTypes(root, m)

	return canvas.CategoricalInk(m, types, pal)
}

// renderTreemapToCanvas walks the layout tree and model tree in parallel,
// adding shapes to the canvas. Returns the populated canvas.
func renderTreemapToCanvas(
	rects treemap.TreemapRectangle,
	root *model.Directory,
	width, height int,
	inks treemapInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	// Background
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(treemapBgColour),
			Border:      canvas.FixedInk(treemapBgColour),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		X:    0, Y: 0,
		W: float64(width), H: float64(height),
	})

	addTreemapRect(cv, rects, root, inks)

	return cv
}

// addTreemapRect recursively adds shapes for a single treemap node.
func addTreemapRect(
	cv *canvas.Canvas,
	rect treemap.TreemapRectangle,
	node *model.Directory,
	inks treemapInks,
) {
	if rect.IsDirectory {
		addDirectoryShapes(cv, rect, inks)
	} else {
		addFileRectForFile(cv, rect, nil, inks)
		return
	}

	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addTreemapRect(cv, child, node.Dirs[dirIdx], inks)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			addFileRectForFile(cv, child, node.Files[fileIdx], inks)
			fileIdx++
		}
	}
}

func addDirectoryShapes(
	cv *canvas.Canvas,
	rect treemap.TreemapRectangle,
	inks treemapInks,
) {
	// Header bar fill
	headerSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(treemapHeaderFill),
			Border:      canvas.FixedInk(treemapHeaderFill),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec: headerSpec,
		X:    rect.X, Y: rect.Y,
		W: rect.W, H: treemapHeaderHeight,
	})

	// Header label
	if rect.Label != "" {
		labelSpec := &canvas.TextSpec{
			Ink:      canvas.FixedInk(treemapWhiteText),
			FontSize: 0,
			Anchor:   canvas.AnchorStart,
		}
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    labelSpec,
			X:       rect.X + 4,
			Y:       rect.Y + treemapHeaderHeight/2,
			Content: rect.Label,
		})
	}

	// Directory border
	borderSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(color.RGBA{A: 0}),
			Border:      canvas.FixedInk(treemapStructuralBorder),
			BorderWidth: treemapDynBorderWidth(rect.W, rect.H, true),
		},
	}
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec: borderSpec,
		X:    rect.X, Y: rect.Y,
		W: rect.W, H: rect.H,
	})
}

func addFileRectForFile(
	cv *canvas.Canvas,
	rect treemap.TreemapRectangle,
	file *model.File,
	inks treemapInks,
) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	hasBorder := inks.border.Info().Kind != canvas.InkFixed

	fillMV := metricValueForFile(file, inks.fill)
	borderMV := metricValueForFile(file, inks.border)

	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.fill,
			Border:      inks.border,
			BorderWidth: treemapDynBorderWidth(rect.W, rect.H, hasBorder),
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
		Label:  rect.Label,
	})
}

// metricValueForFile builds a MetricValue from a file's data for the given ink.
func metricValueForFile(file *model.File, ink canvas.Ink) canvas.MetricValue {
	if file == nil {
		return canvas.MetricValue{}
	}

	info := ink.Info()

	switch info.Kind {
	case canvas.InkFixed:
		return canvas.MetricValue{}
	case canvas.InkNumeric:
		m := info.MetricName
		if v, ok := file.Quantity(m); ok {
			return canvas.MetricValue{Kind: metric.Quantity, Quantity: int(v)}
		}

		if v, ok := file.Measure(m); ok {
			return canvas.MetricValue{Kind: metric.Measure, Measure: v}
		}

		return canvas.MetricValue{}
	case canvas.InkCategorical:
		m := info.MetricName
		if v, ok := file.Classification(m); ok {
			return canvas.MetricValue{Kind: metric.Classification, Category: v}
		}

		return canvas.MetricValue{}
	default:
		return canvas.MetricValue{}
	}
}

// treemapDynBorderWidth returns a dynamic border width based on rectangle
// size and whether a border metric is configured.
func treemapDynBorderWidth(w, h float64, hasBorderMetric bool) float64 {
	if !hasBorderMetric {
		return 0.5
	}

	minDim := min(w, h)

	switch {
	case minDim < treemapMinBorderDim:
		return 1.0
	case minDim >= treemapMidBorderDim:
		return 3.0
	default:
		return 2.0
	}
}
