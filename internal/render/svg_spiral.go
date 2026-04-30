package render

import (
	"fmt"
	"html"
	"math"
	"os"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/spiral"
)

// renderSpiralSVG generates an SVG file from the spiral layout.
func renderSpiralSVG(
	nodes []spiral.SpiralNode,
	width, height int,
	outputPath string,
	legend *LegendInfo,
) (err error) {
	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	writeSpiralSVGHeader(f, width, height)
	writeSpiralSVGTrack(f, nodes, width, height)
	writeSpiralSVGDiscs(f, nodes)
	writeSpiralSVGLabels(f, nodes)

	writeSVGLegend(f, legend, width, height)

	fmt.Fprint(f, "</svg>\n")

	return nil
}

func writeSpiralSVGHeader(f *os.File, width, height int) {
	fmt.Fprintf(f,
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<svg xmlns=\"http://www.w3.org/2000/svg\""+
			" width=\"%d\" height=\"%d\""+
			" viewBox=\"0 0 %d %d\">\n",
		width, height, width, height)

	fmt.Fprintf(f,
		"<rect x=\"0\" y=\"0\" width=\"%d\" height=\"%d\" fill=\"#ffffff\"/>\n",
		width, height)
}

// writeSpiralSVGTrack writes the guide curve as an SVG <path>.
func writeSpiralSVGTrack(f *os.File, nodes []spiral.SpiralNode, width, height int) {
	if len(nodes) < 2 {
		return
	}

	params := inferTrackParams(nodes, width, height)
	steps := spiralTrackSteps(len(nodes))

	fmt.Fprint(f, "<path d=\"")

	for i := range steps {
		t := float64(i) / float64(steps-1)
		theta := t * params.maxTheta
		r := params.a + params.b*theta
		x := params.cx + r*math.Sin(theta)
		y := params.cy - r*math.Cos(theta)

		if i == 0 {
			fmt.Fprintf(f, "M %.2f %.2f", x, y)
		} else {
			fmt.Fprintf(f, " L %.2f %.2f", x, y)
		}
	}

	fmt.Fprintf(f,
		"\" fill=\"none\" stroke=\"%s\" stroke-width=\"%.1f\"/>\n",
		colourToHex(spiralTrackColour), spiralTrackWidth)
}

// writeSpiralSVGDiscs writes <circle> elements for each spiral node.
func writeSpiralSVGDiscs(f *os.File, nodes []spiral.SpiralNode) {
	for _, n := range nodes {
		if n.DiscRadius <= 0 {
			continue
		}

		fill := spiralDefaultFill
		if n.FillColour.A > 0 {
			fill = n.FillColour
		}

		border := spiralDefaultBorder
		if n.BorderColour != nil {
			border = *n.BorderColour
		}

		fmt.Fprintf(f,
			"<circle cx=\"%.2f\" cy=\"%.2f\" r=\"%.2f\""+
				" fill=\"%s\" stroke=\"%s\" stroke-width=\"1\"/>\n",
			n.X, n.Y, n.DiscRadius,
			colourToHex(fill), colourToHex(border))
	}
}

// writeSpiralSVGLabels writes rotated <text> elements for labelled nodes.
func writeSpiralSVGLabels(f *os.File, nodes []spiral.SpiralNode) {
	for _, n := range nodes {
		if !n.ShowLabel || n.Label == "" {
			continue
		}

		writeSpiralSVGLabel(f, n)
	}
}

func writeSpiralSVGLabel(f *os.File, n spiral.SpiralNode) {
	labelR := n.DiscRadius + spiralLabelGap
	lx := n.X + labelR*math.Sin(n.Angle)
	ly := n.Y - labelR*math.Cos(n.Angle)

	// Convert clockwise-from-north angle to SVG rotation degrees.
	rotDeg := n.Angle * 180.0 / math.Pi

	norm := math.Mod(n.Angle, 2*math.Pi)
	if norm < 0 {
		norm += 2 * math.Pi
	}

	var anchor string

	if norm <= math.Pi {
		anchor = "start"
	} else {
		rotDeg += 180.0
		anchor = "end"
	}

	writeSVGTextRotated(f,
		lx, ly,
		colourToHex(spiralLabelColour),
		anchor,
		rotDeg,
		html.EscapeString(n.Label))
}
