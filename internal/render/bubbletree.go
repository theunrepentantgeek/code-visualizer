package render

import (
	"image/color"
	"sort"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/bubbletree"
)

var (
	bubbleDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bubbleDefaultDirFill  = color.RGBA{R: 0x66, G: 0x99, B: 0xCC, A: 0xFF}
	bubbleDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	bubbleLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
)

const (
	// bubbleDirAlpha is the fill opacity for directory circles (~18%),
	// low enough that nested layers remain visible through parents.
	bubbleDirAlpha = 0x30

	// bubbleLabelInset is the distance in pixels from the top edge of a
	// directory circle to the label baseline.
	bubbleLabelInset = 14.0
)

// RenderBubble renders the bubble tree layout to an image file at the given path.
// The output format is determined by the file extension (png, jpg/jpeg, svg).
// Drawing uses three passes — directory circles, file circles, then labels —
// to ensure correct z-ordering.
func RenderBubble(root *bubbletree.BubbleNode, width, height int, outputPath string) error {
	if root == nil {
		return eris.New("nil root node")
	}

	format, err := FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	if format == FormatSVG {
		return renderBubbleSVG(root, width, height, outputPath)
	}

	dc := renderBubbleImage(root, width, height)

	switch format {
	case FormatPNG:
		return saveContextPNG(dc, outputPath)
	case FormatJPG:
		return saveContextJPG(dc, outputPath)
	default:
		return eris.Errorf("unexpected format: %d", format)
	}
}

// renderBubbleImage draws the bubble tree to a gg context using three passes.
func renderBubbleImage(root *bubbletree.BubbleNode, width, height int) *gg.Context {
	dc := gg.NewContext(width, height)

	dc.SetColor(color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	dc.Clear()

	drawBubbleDirs(dc, *root)
	drawBubbleFiles(dc, *root)
	drawBubbleLabels(dc, *root)

	return dc
}

// collectBubbleDirs recursively gathers all directory nodes with positive radius.
func collectBubbleDirs(node bubbletree.BubbleNode) []bubbletree.BubbleNode {
	var result []bubbletree.BubbleNode

	if node.IsDirectory && node.Radius > 0 {
		result = append(result, node)
	}

	for _, child := range node.Children {
		result = append(result, collectBubbleDirs(child)...)
	}

	return result
}

// collectBubbleFiles recursively gathers all file nodes with positive radius.
func collectBubbleFiles(node bubbletree.BubbleNode) []bubbletree.BubbleNode {
	var result []bubbletree.BubbleNode

	if !node.IsDirectory && node.Radius > 0 {
		result = append(result, node)
	}

	for _, child := range node.Children {
		result = append(result, collectBubbleFiles(child)...)
	}

	return result
}

// resolveDirFill returns the fill colour for a directory circle, applying a
// low alpha so nested circles remain visible through their parents.
func resolveDirFill(node bubbletree.BubbleNode) color.RGBA {
	fill := bubbleDefaultDirFill
	if node.FillColour.A > 0 {
		fill = node.FillColour
	}

	fill.A = bubbleDirAlpha

	return fill
}

// resolveFileFill returns the solid fill colour for a file circle.
func resolveFileFill(node bubbletree.BubbleNode) color.RGBA {
	if node.FillColour.A > 0 {
		return node.FillColour
	}

	return bubbleDefaultFileFill
}

// resolveBorder returns the border colour for a bubble node.
func resolveBorder(node bubbletree.BubbleNode) color.RGBA {
	if node.BorderColour != nil {
		return *node.BorderColour
	}

	return bubbleDefaultBorder
}

// drawBubbleDirs draws all directory circles sorted by radius descending
// (outermost first) so inner circles are never obscured.
func drawBubbleDirs(dc *gg.Context, root bubbletree.BubbleNode) {
	dirs := collectBubbleDirs(root)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Radius > dirs[j].Radius
	})

	for _, n := range dirs {
		dc.SetColor(resolveDirFill(n))
		dc.DrawCircle(n.X, n.Y, n.Radius)
		dc.Fill()

		dc.SetColor(resolveBorder(n))
		dc.SetLineWidth(0.5)
		dc.DrawCircle(n.X, n.Y, n.Radius)
		dc.Stroke()
	}
}

// drawBubbleFiles draws all file circles with solid fills.
func drawBubbleFiles(dc *gg.Context, root bubbletree.BubbleNode) {
	files := collectBubbleFiles(root)

	for _, n := range files {
		dc.SetColor(resolveFileFill(n))
		dc.DrawCircle(n.X, n.Y, n.Radius)
		dc.Fill()

		dc.SetColor(resolveBorder(n))
		dc.SetLineWidth(0.5)
		dc.DrawCircle(n.X, n.Y, n.Radius)
		dc.Stroke()
	}
}

// drawBubbleLabels draws labels for all nodes with ShowLabel set.
// Directory labels sit inside the circle near the top edge;
// file labels are centred on the circle.
func drawBubbleLabels(dc *gg.Context, root bubbletree.BubbleNode) {
	drawBubbleLabelRecursive(dc, root)
}

func drawBubbleLabelRecursive(dc *gg.Context, node bubbletree.BubbleNode) {
	if node.ShowLabel && node.Label != "" {
		dc.SetColor(bubbleLabelColour)

		if node.IsDirectory {
			ly := node.Y - node.Radius + bubbleLabelInset
			dc.DrawStringAnchored(node.Label, node.X, ly, 0.5, 0.5)
		} else {
			dc.DrawStringAnchored(node.Label, node.X, node.Y, 0.5, 0.5)
		}
	}

	for _, child := range node.Children {
		drawBubbleLabelRecursive(dc, child)
	}
}
