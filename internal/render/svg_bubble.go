package render

import (
	"fmt"
	"html"
	"os"
	"sort"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/bubbletree"
)

// renderBubbleSVG generates an SVG file from the bubble tree layout.
func renderBubbleSVG(
	root *bubbletree.BubbleNode, width, height int, legend *LegendInfo, outputPath string,
) (err error) {
	legendH := ComputeLegendHeight(legend)
	totalHeight := height + legendH

	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	fmt.Fprintf(f,
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"+
			"<svg xmlns=\"http://www.w3.org/2000/svg\""+
			" xmlns:xlink=\"http://www.w3.org/1999/xlink\""+
			" width=\"%d\" height=\"%d\""+
			" viewBox=\"0 0 %d %d\">\n",
		width, totalHeight, width, totalHeight)

	fmt.Fprintf(f,
		"<rect x=\"0\" y=\"0\" width=\"%d\" height=\"%d\" fill=\"#ffffff\"/>\n",
		width, totalHeight)

	labelledDirs := collectLabelledDirs(*root)
	writeSVGArcDefs(f, labelledDirs)

	writeSVGBubbleDirs(f, *root)
	writeSVGBubbleFiles(f, *root)
	writeSVGBubbleDirLabels(f, labelledDirs)
	writeSVGBubbleFileLabels(f, *root)

	writeSVGLegend(f, legend, 0, float64(height), float64(width))

	fmt.Fprint(f, "</svg>\n")

	return nil
}

// writeSVGBubbleDirs writes directory circle elements, outermost (largest) first.
func writeSVGBubbleDirs(f *os.File, root bubbletree.BubbleNode) {
	dirs := collectBubbleDirs(root)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Radius > dirs[j].Radius
	})

	dirOpacity := float64(bubbleDirAlpha) / 255.0

	for _, n := range dirs {
		fill := bubbleDefaultDirFill
		if n.FillColour.A > 0 {
			fill = n.FillColour
		}

		border := resolveBorder(n)

		fmt.Fprintf(f,
			"<circle cx=\"%.2f\" cy=\"%.2f\" r=\"%.2f\""+
				" fill=\"%s\" fill-opacity=\"%.2f\""+
				" stroke=\"%s\" stroke-width=\"0.5\"/>\n",
			n.X, n.Y, n.Radius,
			colourToHex(fill), dirOpacity,
			colourToHex(border))
	}
}

// writeSVGBubbleFiles writes file circle elements with solid fills.
func writeSVGBubbleFiles(f *os.File, root bubbletree.BubbleNode) {
	files := collectBubbleFiles(root)

	for _, n := range files {
		fill := resolveFileFill(n)
		border := resolveBorder(n)

		fmt.Fprintf(f,
			"<circle cx=\"%.2f\" cy=\"%.2f\" r=\"%.2f\""+
				" fill=\"%s\" stroke=\"%s\" stroke-width=\"0.5\"/>\n",
			n.X, n.Y, n.Radius,
			colourToHex(fill), colourToHex(border))
	}
}

// collectLabelledDirs recursively gathers directory nodes that have a visible label.
func collectLabelledDirs(node bubbletree.BubbleNode) []bubbletree.BubbleNode {
	var result []bubbletree.BubbleNode

	if node.IsDirectory && node.ShowLabel && node.Label != "" {
		result = append(result, node)
	}

	for _, child := range node.Children {
		result = append(result, collectLabelledDirs(child)...)
	}

	return result
}

// writeSVGArcDefs writes a <defs> block with one arc <path> per labelled directory.
// Each arc traces the top semicircle from left to right.
func writeSVGArcDefs(f *os.File, dirs []bubbletree.BubbleNode) {
	if len(dirs) == 0 {
		return
	}

	fmt.Fprint(f, "<defs>\n")

	for idx, n := range dirs {
		arcR := n.Radius - bubbleLabelInset
		if arcR <= 0 {
			continue
		}

		// Arc from (cx - arcR, cy) to (cx + arcR, cy) — top semicircle.
		fmt.Fprintf(f,
			"<path id=\"arc-%d\" d=\"M %.2f,%.2f A %.2f,%.2f 0 0,1 %.2f,%.2f\" fill=\"none\"/>\n",
			idx,
			n.X-arcR, n.Y,
			arcR, arcR,
			n.X+arcR, n.Y)
	}

	fmt.Fprint(f, "</defs>\n")
}

// writeSVGBubbleDirLabels writes curved <textPath> labels for directory nodes.
func writeSVGBubbleDirLabels(f *os.File, dirs []bubbletree.BubbleNode) {
	for idx, n := range dirs {
		fontSize := computeArcFontSize(n.Label, n.Radius)
		if fontSize == 0 {
			continue
		}

		fmt.Fprintf(f,
			"<text font-size=\"%.1f\" font-family=\"sans-serif\" fill=\"%s\">"+
				"<textPath href=\"#arc-%d\" startOffset=\"50%%\" text-anchor=\"middle\">%s</textPath>"+
				"</text>\n",
			fontSize,
			colourToHex(bubbleLabelColour),
			idx,
			html.EscapeString(n.Label))
	}
}

// writeSVGBubbleFileLabels writes centred text labels for file nodes.
func writeSVGBubbleFileLabels(f *os.File, root bubbletree.BubbleNode) {
	writeSVGBubbleFileLabelRecursive(f, root)
}

func writeSVGBubbleFileLabelRecursive(f *os.File, node bubbletree.BubbleNode) {
	if !node.IsDirectory && node.ShowLabel && node.Label != "" {
		writeSVGText(f,
			node.X, node.Y,
			colourToHex(bubbleLabelColour),
			"middle",
			html.EscapeString(node.Label))
	}

	for _, child := range node.Children {
		writeSVGBubbleFileLabelRecursive(f, child)
	}
}
