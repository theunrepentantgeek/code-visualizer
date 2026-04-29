package render

import (
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/spiral"
)

var (
	spiralDefaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	spiralDefaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	spiralTrackColour   = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	spiralLabelColour   = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
)

const (
	spiralTrackWidth = 1.0
	spiralLabelGap   = 4.0
	spiralTrackSteps = 500
)

// RenderSpiral renders the spiral layout to an image file at the given path.
// The output format is determined by the file extension (png, jpg/jpeg, svg).
// Drawing uses three passes — background track, discs, then labels — to ensure
// correct z-ordering.
func RenderSpiral(
	nodes []spiral.SpiralNode,
	width int,
	height int,
	outputPath string,
	legend *LegendInfo,
) error {
	format, err := FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	if format == FormatSVG {
		return renderSpiralSVG(nodes, width, height, outputPath, legend)
	}

	dc := renderSpiralImage(nodes, width, height)
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

// renderSpiralImage draws the spiral to a gg context using three passes.
func renderSpiralImage(nodes []spiral.SpiralNode, width, height int) *gg.Context {
	dc := gg.NewContext(width, height)

	dc.SetColor(color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	dc.Clear()

	drawSpiralTrack(dc, nodes, width, height)
	drawSpiralDiscs(dc, nodes)
	drawSpiralLabels(dc, nodes)

	return dc
}

// drawSpiralTrack draws a faint guide curve through the spiral path (pass 1).
func drawSpiralTrack(dc *gg.Context, nodes []spiral.SpiralNode, width, height int) {
	if len(nodes) < 2 {
		return
	}

	params := inferTrackParams(nodes, width, height)

	dc.SetColor(spiralTrackColour)
	dc.SetLineWidth(spiralTrackWidth)

	for i := range spiralTrackSteps {
		t := float64(i) / float64(spiralTrackSteps-1)
		theta := t * params.maxTheta
		r := params.a + params.b*theta
		x := params.cx + r*math.Sin(theta)
		y := params.cy - r*math.Cos(theta)

		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}

	dc.Stroke()
}

// trackParams holds spiral geometry inferred from positioned nodes.
type trackParams struct {
	cx, cy   float64
	a, b     float64
	maxTheta float64
}

// inferTrackParams reconstructs Archimedean spiral parameters from nodes.
func inferTrackParams(nodes []spiral.SpiralNode, width, height int) trackParams {
	cx := float64(width) / 2
	cy := float64(height) / 2

	first := nodes[0]
	last := nodes[len(nodes)-1]

	return trackParams{
		cx:       cx,
		cy:       cy,
		a:        first.SpiralRadius,
		b:        spiralGrowthRate(first, last),
		maxTheta: last.Angle,
	}
}

// spiralGrowthRate computes the radial growth per radian from two nodes.
func spiralGrowthRate(first, last spiral.SpiralNode) float64 {
	if last.Angle == 0 {
		return 0
	}

	return (last.SpiralRadius - first.SpiralRadius) / last.Angle
}

// drawSpiralDiscs draws all spot discs in spiral order (pass 2).
func drawSpiralDiscs(dc *gg.Context, nodes []spiral.SpiralNode) {
	for _, n := range nodes {
		if n.DiscRadius <= 0 {
			continue
		}

		drawSingleSpot(dc, n)
	}
}

// drawSingleSpot draws the filled circle and border for one spiral node.
func drawSingleSpot(dc *gg.Context, n spiral.SpiralNode) {
	fill := spiralDefaultFill
	if n.FillColour.A > 0 {
		fill = n.FillColour
	}

	dc.SetColor(fill)
	dc.DrawCircle(n.X, n.Y, n.DiscRadius)
	dc.Fill()

	border := spiralDefaultBorder
	if n.BorderColour != nil {
		border = *n.BorderColour
	}

	dc.SetColor(border)
	dc.SetLineWidth(1.0)
	dc.DrawCircle(n.X, n.Y, n.DiscRadius)
	dc.Stroke()
}

// drawSpiralLabels draws labels tangent to the spiral (pass 3).
func drawSpiralLabels(dc *gg.Context, nodes []spiral.SpiralNode) {
	for _, n := range nodes {
		if !n.ShowLabel || n.Label == "" {
			continue
		}

		drawSpiralLabel(dc, n)
	}
}

// drawSpiralLabel draws a single label oriented tangent to the spiral,
// with upright-flipping so text is always readable.
func drawSpiralLabel(dc *gg.Context, n spiral.SpiralNode) {
	labelR := n.DiscRadius + spiralLabelGap
	lx := n.X + labelR*math.Sin(n.Angle)
	ly := n.Y - labelR*math.Cos(n.Angle)

	// Angle from north (clockwise) converted to gg rotation (CW from east).
	rotAngle := n.Angle

	// Normalise to [0, 2π) for half-plane check.
	norm := math.Mod(rotAngle, 2*math.Pi)
	if norm < 0 {
		norm += 2 * math.Pi
	}

	dc.SetColor(spiralLabelColour)
	dc.Push()

	if norm <= math.Pi {
		// Right half: rotate so text reads outward, anchor at start.
		dc.RotateAbout(rotAngle, lx, ly)
		dc.DrawStringAnchored(n.Label, lx, ly, 0.0, 0.5)
	} else {
		// Left half: flip 180° so characters stay upright, anchor at end.
		dc.RotateAbout(rotAngle+math.Pi, lx, ly)
		dc.DrawStringAnchored(n.Label, lx, ly, 1.0, 0.5)
	}

	dc.Pop()
}
