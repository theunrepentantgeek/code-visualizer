package render

import (
	"fmt"
	"html"
	"image/color"
	"math"
	"os"
	"sort"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/radialtree"
)

// renderRadialSVG generates an SVG file from the radial tree layout.
func renderRadialSVG(root *radialtree.RadialNode, canvasSize int, legend *LegendInfo, outputPath string) (err error) {
	legendH := ComputeLegendHeight(legend)
	totalHeight := canvasSize + legendH

	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	cs := float64(canvasSize)

	fmt.Fprintf(f, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
`, canvasSize, totalHeight, canvasSize, totalHeight)

	fmt.Fprintf(f,
		"<rect x=\"0\" y=\"0\" width=\"%d\" height=\"%d\""+
			" fill=\"#ffffff\"/>\n",
		canvasSize, totalHeight)

	cx, cy := cs/2.0, cs/2.0

	writeSVGEdges(f, *root, cx, cy)
	writeSVGDiscs(f, *root, cx, cy)
	writeSVGLabels(f, *root, cx, cy)

	writeSVGLegend(f, legend, cs, cs)

	fmt.Fprint(f, "</svg>\n")

	return nil
}

func writeSVGEdges(f *os.File, node radialtree.RadialNode, cx, cy float64) {
	px := cx + node.X
	py := cy + node.Y

	for _, child := range node.Children {
		chx := cx + child.X
		chy := cy + child.Y

		fmt.Fprintf(f,
			"<line x1=\"%.2f\" y1=\"%.2f\" x2=\"%.2f\" y2=\"%.2f\""+
				" stroke=\"%s\" stroke-width=\"0.5\"/>\n",
			px, py, chx, chy,
			colourToHex(radialEdgeColour))

		writeSVGEdges(f, child, cx, cy)
	}
}

type svgDisc struct {
	node   radialtree.RadialNode
	sx, sy float64
}

func collectSVGDiscs(
	n radialtree.RadialNode, cx, cy float64,
) []svgDisc {
	var discs []svgDisc

	if n.DiscRadius > 0 {
		discs = append(discs, svgDisc{
			node: n,
			sx:   cx + n.X,
			sy:   cy + n.Y,
		})
	}

	for _, child := range n.Children {
		discs = append(discs, collectSVGDiscs(child, cx, cy)...)
	}

	return discs
}

func writeSVGDiscs(f *os.File, node radialtree.RadialNode, cx, cy float64) {
	discs := collectSVGDiscs(node, cx, cy)

	sort.Slice(discs, func(i, j int) bool {
		return discs[i].node.DiscRadius > discs[j].node.DiscRadius
	})

	for _, d := range discs {
		fill := radialDefaultFileFill
		if d.node.IsDirectory {
			fill = radialDefaultDirFill
		}

		if d.node.FillColour.A > 0 {
			fill = d.node.FillColour
		}

		border := radialDefaultBorder
		if d.node.BorderColour != nil {
			border = *d.node.BorderColour
		}

		fmt.Fprintf(f,
			"<circle cx=\"%.2f\" cy=\"%.2f\" r=\"%.2f\""+
				" fill=\"%s\" stroke=\"%s\" stroke-width=\"1\"/>\n",
			d.sx, d.sy, d.node.DiscRadius,
			colourToHex(fill), colourToHex(border))
	}
}

func writeSVGLabels(f *os.File, node radialtree.RadialNode, cx, cy float64) {
	if node.ShowLabel && node.Label != "" {
		dist := math.Sqrt(node.X*node.X + node.Y*node.Y)

		if dist == 0 {
			fill := svgEffectiveFill(node)
			textCol := TextColourFor(fill)

			writeSVGText(f,
				cx+node.X, cy+node.Y,
				colourToHex(textCol),
				"middle",
				html.EscapeString(node.Label))
		} else {
			writeSVGExternalLabel(f, node, cx, cy)
		}
	}

	for _, child := range node.Children {
		writeSVGLabels(f, child, cx, cy)
	}
}

func writeSVGExternalLabel(
	f *os.File, node radialtree.RadialNode, cx, cy float64,
) {
	dist := math.Sqrt(node.X*node.X + node.Y*node.Y)
	labelRadius := dist + node.DiscRadius + radialLabelGap
	lx := cx + labelRadius*math.Cos(node.Angle)
	ly := cy + labelRadius*math.Sin(node.Angle)

	// Normalise angle to [0, 2π) for half-plane check.
	angle := math.Mod(node.Angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	rotDeg := angle * 180.0 / math.Pi

	var anchor string

	if angle <= math.Pi/2 || angle > 3*math.Pi/2 {
		anchor = "start"
	} else {
		rotDeg += 180.0
		anchor = "end"
	}

	writeSVGTextRotated(f,
		lx, ly,
		colourToHex(radialLabelColour),
		anchor,
		rotDeg,
		html.EscapeString(node.Label))
}

func svgEffectiveFill(node radialtree.RadialNode) color.RGBA {
	if node.FillColour.A > 0 {
		return node.FillColour
	}

	if node.IsDirectory {
		return radialDefaultDirFill
	}

	return radialDefaultFileFill
}
