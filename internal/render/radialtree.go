package render

import (
	"image/color"
	"math"
	"sort"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/radialtree"
)

var (
	radialDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	radialDefaultDirFill  = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	radialDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	radialEdgeColour      = color.RGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF}
	radialLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
)

const radialLabelGap = 4.0

// RenderRadial renders the radial tree layout to an image file at the given path.
// The output format is determined by the file extension (png, jpg/jpeg, svg).
// Drawing is done in three passes — edges, then discs, then labels — to ensure
// correct z-ordering across the entire tree.
// If legend is non-nil and has a valid position, a legend overlay is drawn.
func RenderRadial(root *radialtree.RadialNode, canvasSize int, outputPath string, legend *LegendInfo) error {
	format, err := FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	if format == FormatSVG {
		return renderRadialSVG(root, canvasSize, outputPath, legend)
	}

	dc := renderRadialImage(root, canvasSize)
	drawLegend(dc, legend, canvasSize, canvasSize)

	switch format {
	case FormatPNG:
		return saveContextPNG(dc, outputPath)
	case FormatJPG:
		return saveContextJPG(dc, outputPath)
	default:
		return eris.Errorf("unexpected format: %d", format)
	}
}

// renderRadialImage draws the radial tree to a gg context.
func renderRadialImage(root *radialtree.RadialNode, canvasSize int) *gg.Context {
	dc := gg.NewContext(canvasSize, canvasSize)

	dc.SetColor(color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	dc.Clear()

	cx, cy := float64(canvasSize)/2.0, float64(canvasSize)/2.0

	drawEdges(dc, *root, cx, cy)
	drawDiscs(dc, *root, cx, cy)
	drawLabels(dc, *root, cx, cy)

	return dc
}

// drawEdges draws a straight line from each node to each of its children, recursively.
func drawEdges(dc *gg.Context, node radialtree.RadialNode, cx, cy float64) {
	px := cx + node.X
	py := cy + node.Y

	dc.SetColor(radialEdgeColour)
	dc.SetLineWidth(0.5)

	for _, child := range node.Children {
		chx := cx + child.X
		chy := cy + child.Y
		dc.DrawLine(px, py, chx, chy)
		drawEdges(dc, child, cx, cy)
	}

	dc.Stroke()
}

// discEntry holds a node and its computed screen position for deferred drawing.
type discEntry struct {
	node   radialtree.RadialNode
	sx, sy float64
}

// collectDiscs recursively gathers all nodes with a positive DiscRadius,
// computing their absolute screen position from the canvas centre (cx, cy).
func collectDiscs(node radialtree.RadialNode, cx, cy float64) []discEntry {
	if node.DiscRadius <= 0 {
		result := make([]discEntry, 0, len(node.Children))
		for _, child := range node.Children {
			result = append(result, collectDiscs(child, cx, cy)...)
		}

		return result
	}

	entry := discEntry{
		node: node,
		sx:   cx + node.X,
		sy:   cy + node.Y,
	}

	result := make([]discEntry, 0, 1+len(node.Children))
	result = append(result, entry)

	for _, child := range node.Children {
		result = append(result, collectDiscs(child, cx, cy)...)
	}

	return result
}

// drawSingleDisc draws the filled circle and border for a single node at (sx, sy).
func drawSingleDisc(dc *gg.Context, node radialtree.RadialNode, sx, sy float64) {
	fill := radialDefaultFileFill
	if node.IsDirectory {
		fill = radialDefaultDirFill
	}

	if node.FillColour.A > 0 {
		fill = node.FillColour
	}

	dc.SetColor(fill)
	dc.DrawCircle(sx, sy, node.DiscRadius)
	dc.Fill()

	border := radialDefaultBorder
	if node.BorderColour != nil {
		border = *node.BorderColour
	}

	dc.SetColor(border)
	dc.SetLineWidth(1.0)
	dc.DrawCircle(sx, sy, node.DiscRadius)
	dc.Stroke()
}

// drawDiscs draws all disc nodes sorted by radius descending so smaller nodes
// are never obscured by larger ones drawn later.
func drawDiscs(dc *gg.Context, node radialtree.RadialNode, cx, cy float64) {
	entries := collectDiscs(node, cx, cy)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].node.DiscRadius > entries[j].node.DiscRadius
	})

	for _, e := range entries {
		drawSingleDisc(dc, e.node, e.sx, e.sy)
	}
}

// drawLabels draws the label for each node that has ShowLabel set, recursively.
// Labels are oriented radially: text runs outward from the canvas centre, with
// characters kept upright by flipping the rotation in the left half of the canvas.
func drawLabels(dc *gg.Context, node radialtree.RadialNode, cx, cy float64) {
	if node.ShowLabel && node.Label != "" {
		sx := cx + node.X
		sy := cy + node.Y

		dist := math.Sqrt(node.X*node.X + node.Y*node.Y)

		if dist == 0 {
			// Root node: centre the label on the disc.
			fill := effectiveFill(node)
			dc.SetColor(TextColourFor(fill))
			dc.DrawStringAnchored(node.Label, sx, sy, 0.5, 0.5)
		} else {
			drawExternalLabel(dc, node, cx, cy)
		}
	}

	for _, child := range node.Children {
		drawLabels(dc, child, cx, cy)
	}
}

// drawExternalLabel draws a radially-oriented label outside the disc for non-root nodes.
func drawExternalLabel(dc *gg.Context, node radialtree.RadialNode, cx, cy float64) {
	dist := math.Sqrt(node.X*node.X + node.Y*node.Y)
	labelRadius := dist + node.DiscRadius + radialLabelGap
	lx := cx + labelRadius*math.Cos(node.Angle)
	ly := cy + labelRadius*math.Sin(node.Angle)

	// Normalise angle to [0, 2π) for half-plane check.
	angle := math.Mod(node.Angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	dc.SetColor(radialLabelColour)
	dc.Push()

	if angle <= math.Pi/2 || angle > 3*math.Pi/2 {
		// Right half: rotate by the raw angle, anchor at left of text.
		dc.RotateAbout(node.Angle, lx, ly)
		dc.DrawStringAnchored(node.Label, lx, ly, 0.0, 0.5)
	} else {
		// Left half: add π to flip so characters stay upright, anchor at right.
		dc.RotateAbout(node.Angle+math.Pi, lx, ly)
		dc.DrawStringAnchored(node.Label, lx, ly, 1.0, 0.5)
	}

	dc.Pop()
}

// effectiveFill returns the fill colour that will be used for a node,
// resolving defaults for files and directories.
func effectiveFill(node radialtree.RadialNode) color.RGBA {
	if node.FillColour.A > 0 {
		return node.FillColour
	}

	if node.IsDirectory {
		return radialDefaultDirFill
	}

	return radialDefaultFileFill
}
