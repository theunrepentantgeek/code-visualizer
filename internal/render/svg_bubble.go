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
func renderBubbleSVG(root *bubbletree.BubbleNode, width, height int, outputPath string) (err error) {
	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	fmt.Fprintf(f, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
`, width, height, width, height)

	fmt.Fprintf(f,
		"<rect x=\"0\" y=\"0\" width=\"%d\" height=\"%d\" fill=\"#ffffff\"/>\n",
		width, height)

	writeSVGBubbleDirs(f, *root)
	writeSVGBubbleFiles(f, *root)
	writeSVGBubbleLabels(f, *root)

	fmt.Fprint(f, "</svg>\n")

	return nil
}

// writeSVGBubbleDirs writes directory circle elements, outermost (largest) first.
func writeSVGBubbleDirs(f *os.File, root bubbletree.BubbleNode) {
	dirs := collectBubblesByType(root, true)
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
	files := collectBubblesByType(root, false)

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

// writeSVGBubbleLabels writes text labels for nodes with ShowLabel set.
func writeSVGBubbleLabels(f *os.File, root bubbletree.BubbleNode) {
	writeSVGBubbleLabelRecursive(f, root)
}

func writeSVGBubbleLabelRecursive(f *os.File, node bubbletree.BubbleNode) {
	if node.ShowLabel && node.Label != "" {
		if node.IsDirectory {
			ly := node.Y - node.Radius + bubbleLabelInset

			writeSVGText(f,
				node.X, ly,
				colourToHex(bubbleLabelColour),
				"middle", "central",
				html.EscapeString(node.Label))
		} else {
			writeSVGText(f,
				node.X, node.Y,
				colourToHex(bubbleLabelColour),
				"middle", "central",
				html.EscapeString(node.Label))
		}
	}

	for _, child := range node.Children {
		writeSVGBubbleLabelRecursive(f, child)
	}
}
