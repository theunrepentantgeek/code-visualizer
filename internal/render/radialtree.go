package render

import (
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/radialtree"
)

var (
	radialDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	radialDefaultDirFill  = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	radialDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	radialEdgeColour      = color.RGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF}
)

const radialLabelGap = 4.0

// RenderRadialPNG renders the radial tree layout to a PNG file at the given path.
// Drawing is done in three passes — edges, then discs, then labels — to ensure
// correct z-ordering across the entire tree.
func RenderRadialPNG(root radialtree.RadialNode, canvasSize int, outputPath string) error {
	dc := gg.NewContext(canvasSize, canvasSize)

	dc.SetColor(color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	dc.Clear()

	cx, cy := float64(canvasSize)/2.0, float64(canvasSize)/2.0

	drawEdges(dc, root, cx, cy)
	drawDiscs(dc, root, cx, cy)
	drawLabels(dc, root, cx, cy)

	return eris.Wrap(dc.SavePNG(outputPath), "failed to save PNG")
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
		dc.Stroke()
		drawEdges(dc, child, cx, cy)
	}
}

// drawDiscs draws the filled circle and border for each node, recursively.
func drawDiscs(dc *gg.Context, node radialtree.RadialNode, cx, cy float64) {
	if node.DiscRadius <= 0 {
		for _, child := range node.Children {
			drawDiscs(dc, child, cx, cy)
		}

		return
	}

	sx := cx + node.X
	sy := cy + node.Y

	// Fill
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

	// Border
	border := radialDefaultBorder
	if node.BorderColour != nil {
		border = *node.BorderColour
	}

	dc.SetColor(border)
	dc.SetLineWidth(1.0)
	dc.DrawCircle(sx, sy, node.DiscRadius)
	dc.Stroke()

	for _, child := range node.Children {
		drawDiscs(dc, child, cx, cy)
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
			labelRadius := dist + node.DiscRadius + radialLabelGap
			lx := cx + labelRadius*math.Cos(node.Angle)
			ly := cy + labelRadius*math.Sin(node.Angle)

			// Normalise angle to [0, 2π) for half-plane check.
			angle := node.Angle
			for angle < 0 {
				angle += 2 * math.Pi
			}
			for angle >= 2*math.Pi {
				angle -= 2 * math.Pi
			}

			fill := effectiveFill(node)
			dc.SetColor(TextColourFor(fill))

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
	}

	for _, child := range node.Children {
		drawLabels(dc, child, cx, cy)
	}
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
