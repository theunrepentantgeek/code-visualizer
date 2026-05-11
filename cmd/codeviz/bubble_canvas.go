package main

import (
	"cmp"
	"image/color"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var (
	bubbleDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bubbleDefaultDirFill  = color.RGBA{R: 0x66, G: 0x99, B: 0xCC, A: 0xFF}
	bubbleDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	bubbleLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bubbleBgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	bubbleDirOpacity      = float64(0x30) / 255.0
	bubbleBorderWidth     = 0.5
	bubbleArcLabelInset   = 14.0
	bubbleMinArcFontSize  = 7.0
	bubbleDefaultFontSize = 14.0
	bubbleMaxArcFraction  = math.Pi / 2.0
)

// bubbleInks holds the Ink instances for a bubbletree render pass.
type bubbleInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildBubbleInks creates fill and border inks from metric configuration.
func buildBubbleInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) bubbleInks {
	inks := bubbleInks{
		fill:   canvas.FixedInk(bubbleDefaultFileFill),
		border: canvas.FixedInk(bubbleDefaultBorder),
	}

	inks.fill = buildMetricInk(root, fillMetric, fillPaletteName, bubbleDefaultFileFill)
	if borderMetric != "" {
		inks.border = buildMetricInk(root, borderMetric, borderPaletteName, bubbleDefaultBorder)
	}

	return inks
}

// renderBubbleToCanvas walks the BubbleNode tree and model tree using
// path-based lookup, adding shapes to the canvas.
func renderBubbleToCanvas(
	nodes *bubbletree.BubbleNode,
	root *model.Directory,
	width, height int,
	inks bubbleInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addBubbleBackground(cv, width, height)

	dirs, files := indexBubbleNodes(nodes)
	addBubbleDirDiscs(cv, dirs, root)
	addBubbleFileDiscs(cv, files, root, inks)
	addBubbleLabels(cv, *nodes)

	return cv
}

// addBubbleBackground adds a white background rectangle.
func addBubbleBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(bubbleBgColour),
			Border:      canvas.FixedInk(bubbleBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    float64(width), H: float64(height),
	})
}

// indexBubbleNodes recursively indexes all BubbleNodes by their Path,
// separating directories and files. Returns two maps.
func indexBubbleNodes(
	node *bubbletree.BubbleNode,
) (dirs map[string]*bubbletree.BubbleNode, files map[string]*bubbletree.BubbleNode) {
	dirs = make(map[string]*bubbletree.BubbleNode)
	files = make(map[string]*bubbletree.BubbleNode)

	indexBubbleNodesWalk(node, dirs, files)

	return dirs, files
}

func indexBubbleNodesWalk(
	node *bubbletree.BubbleNode,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory {
			dirs[child.Path] = child
			indexBubbleNodesWalk(child, dirs, files)
		} else {
			files[child.Path] = child
		}
	}
}

// bubbleDirEntry holds a directory node for sorted drawing.
type bubbleDirEntry struct {
	node *bubbletree.BubbleNode
}

// addBubbleDirDiscs collects directory nodes from the model tree (via path lookup),
// sorts them largest-first, and adds semi-transparent discs to the canvas.
func addBubbleDirDiscs(
	cv *canvas.Canvas,
	dirIndex map[string]*bubbletree.BubbleNode,
	root *model.Directory,
) {
	entries := collectBubbleDirEntries(dirIndex, root)

	slices.SortFunc(entries, func(a, b bubbleDirEntry) int {
		return cmp.Compare(b.node.Radius, a.node.Radius)
	})

	dirFill := canvas.FixedInk(bubbleDefaultDirFill, canvas.WithOpacity(bubbleDirOpacity))
	dirBorder := canvas.FixedInk(bubbleDefaultBorder)

	dirSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        dirFill,
			Border:      dirBorder,
			BorderWidth: bubbleBorderWidth,
		},
	}

	for _, e := range entries {
		cv.AddDisc(canvas.LayerStructure, canvas.Disc{
			Spec:   dirSpec,
			X:      e.node.X,
			Y:      e.node.Y,
			Radius: e.node.Radius,
		})
	}
}

// collectBubbleDirEntries recursively walks model.Directory to find
// all directories that have a corresponding BubbleNode.
func collectBubbleDirEntries(
	dirIndex map[string]*bubbletree.BubbleNode,
	dir *model.Directory,
) []bubbleDirEntry {
	var entries []bubbleDirEntry

	for _, d := range dir.Dirs {
		if bn, ok := dirIndex[d.Path]; ok && bn.Radius > 0 {
			entries = append(entries, bubbleDirEntry{node: bn})
			entries = append(entries, collectBubbleDirEntries(dirIndex, d)...)
		}
	}

	return entries
}

// addBubbleFileDiscs walks the model tree via path lookup
// and adds file discs to the canvas.
func addBubbleFileDiscs(
	cv *canvas.Canvas,
	fileIndex map[string]*bubbletree.BubbleNode,
	root *model.Directory,
	inks bubbleInks,
) {
	addBubbleFileDiscsWalk(cv, fileIndex, root, inks)
}

func addBubbleFileDiscsWalk(
	cv *canvas.Canvas,
	fileIndex map[string]*bubbletree.BubbleNode,
	dir *model.Directory,
	inks bubbleInks,
) {
	fileSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.fill,
			Border:      inks.border,
			BorderWidth: bubbleBorderWidth,
		},
	}

	for _, f := range dir.Files {
		bn, ok := fileIndex[f.Path]
		if !ok || bn.Radius <= 0 {
			continue
		}

		fillMV := metricValueForFile(f, inks.fill)
		borderMV := metricValueForFile(f, inks.border)

		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   fileSpec,
			X:      bn.X,
			Y:      bn.Y,
			Radius: bn.Radius,
			Fill:   fillMV,
			Border: borderMV,
		})
	}

	for _, d := range dir.Dirs {
		addBubbleFileDiscsWalk(cv, fileIndex, d, inks)
	}
}

// addBubbleLabels recursively adds labels for all nodes with ShowLabel set.
// Directory labels use arc text; file labels use centred text.
func addBubbleLabels(cv *canvas.Canvas, node bubbletree.BubbleNode) {
	if node.ShowLabel && node.Label != "" {
		if node.IsDirectory {
			addBubbleDirLabel(cv, node)
		} else {
			addBubbleFileLabel(cv, node)
		}
	}

	for _, child := range node.Children {
		addBubbleLabels(cv, child)
	}
}

// addBubbleDirLabel adds an arc text label curved along the top of a directory circle.
func addBubbleDirLabel(cv *canvas.Canvas, node bubbletree.BubbleNode) {
	fontSize := bubbleArcFontSize(node.Label, node.Radius)
	if fontSize == 0 {
		return
	}

	arcSpec := &canvas.ArcTextSpec{
		Ink:      canvas.FixedInk(bubbleLabelColour),
		FontSize: fontSize,
	}

	cv.AddArcText(canvas.LayerOverlay, canvas.ArcText{
		Spec:   arcSpec,
		X:      node.X,
		Y:      node.Y,
		Radius: node.Radius,
		Text:   node.Label,
	})
}

// addBubbleFileLabel adds a centred text label on a file circle.
func addBubbleFileLabel(cv *canvas.Canvas, node bubbletree.BubbleNode) {
	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(bubbleLabelColour),
		FontSize: 0,
		Anchor:   canvas.AnchorMiddle,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       node.X,
		Y:       node.Y,
		Content: node.Label,
	})
}

// bubbleArcFontSize computes the font size for a label to fit within
// bubbleMaxArcFraction of the circle arc. Returns 0 if the label cannot fit
// at the minimum readable font size.
func bubbleArcFontSize(label string, radius float64) float64 {
	charCount := float64(utf8.RuneCountInString(label))
	if charCount == 0 {
		return 0
	}

	arcR := radius - bubbleArcLabelInset
	if arcR <= 0 {
		return 0
	}

	maxArcLen := arcR * bubbleMaxArcFraction
	// Each character is approximately 0.6 × fontSize wide.
	maxSize := maxArcLen / (charCount * 0.6)
	fontSize := min(bubbleDefaultFontSize, maxSize)

	if fontSize < bubbleMinArcFontSize {
		return 0
	}

	return fontSize
}
