// Package render produces images of treemap and radial-tree layouts.
// Raster formats (PNG, JPG) use the fogleman/gg graphics library;
// SVG is generated directly as XML.
package render

import (
	"image/color"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/treemap"
)

var (
	structuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	headerFill       = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	defaultFill      = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bgColour         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Render renders the treemap layout to an image file at the given path.
// The output format is determined by the file extension (png, jpg/jpeg, svg).
// If legend is non-nil and has a valid position, a legend overlay is drawn.
func Render(root treemap.TreemapRectangle, width, height int, outputPath string, legend *LegendInfo) error {
	format, err := FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	if format == FormatSVG {
		return renderTreemapSVG(root, width, height, outputPath, legend)
	}

	dc := renderTreemapImage(root, width, height)
	drawLegend(dc, legend, width, height)

	switch format {
	case FormatPNG:
		return saveContextPNG(dc, outputPath)
	case FormatJPG:
		return saveContextJPG(dc, outputPath)
	default:
		return eris.Errorf("unexpected format: %d", format)
	}
}

// renderTreemapImage draws the treemap to a gg context.
func renderTreemapImage(root treemap.TreemapRectangle, width, height int) *gg.Context {
	dc := gg.NewContext(width, height)

	dc.SetColor(bgColour)
	dc.Clear()

	drawRect(dc, root)

	return dc
}

func drawRect(dc *gg.Context, rect treemap.TreemapRectangle) {
	if rect.IsDirectory {
		drawDirectoryHeader(dc, rect)
	} else {
		drawFileRect(dc, rect)
	}

	for _, child := range rect.Children {
		drawRect(dc, child)
	}
}

func drawDirectoryHeader(dc *gg.Context, rect treemap.TreemapRectangle) {
	// Draw header bar
	dc.SetColor(headerFill)
	dc.DrawRectangle(rect.X, rect.Y, rect.W, treemap.HeaderHeight)
	dc.Fill()

	// Header label
	dc.SetColor(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	dc.DrawStringAnchored(rect.Label, rect.X+4, rect.Y+treemap.HeaderHeight/2, 0, 0.5)

	// Border around entire directory group
	dc.SetColor(structuralBorder)
	dc.SetLineWidth(1)
	dc.DrawRectangle(rect.X, rect.Y, rect.W, rect.H)
	dc.Stroke()
}

func drawFileRect(dc *gg.Context, rect treemap.TreemapRectangle) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	// Fill
	fill := defaultFill
	if rect.FillColour.A > 0 {
		fill = rect.FillColour
	}

	dc.SetColor(fill)
	dc.DrawRectangle(rect.X, rect.Y, rect.W, rect.H)
	dc.Fill()

	// Border
	if rect.BorderColour != nil {
		dc.SetColor(*rect.BorderColour)
	} else {
		dc.SetColor(structuralBorder)
	}

	dc.SetLineWidth(1)
	dc.DrawRectangle(rect.X, rect.Y, rect.W, rect.H)
	dc.Stroke()

	// Label
	if ShouldShowLabel(rect) {
		textCol := TextColourFor(fill)
		dc.SetColor(textCol)
		dc.DrawStringAnchored(rect.Label, rect.X+rect.W/2, rect.Y+rect.H/2, 0.5, 0.5)
	}
}
