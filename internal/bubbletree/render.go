package bubbletree

import (
	"cmp"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	bubbleDirOpacity        = float64(0x30) / 255.0
	bubbleBorderWidth       = 0.5
	bubbleMetricBorderWidth = 2.0
	bubbleArcLabelInset     = canvasmodel.ArcTextInset
	bubbleMinArcFontSize    = 7.0
	bubbleDefaultFontSize   = 14.0
	bubbleMaxArcFraction    = math.Pi / 2.0
)

// RenderToCanvas walks the BubbleNode tree and model tree using
// path-based lookup, adding shapes to the canvas.
func RenderToCanvas(
	nodes *BubbleNode,
	root *model.Directory,
	width, height int,
	is Inks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addBubbleBackground(cv, width, height)

	dirs, files := indexBubbleNodes(nodes)
	addBubbleDirDiscs(cv, dirs, root)
	addBubbleFileDiscs(cv, files, root, is)
	addBubbleLabels(cv, *nodes)

	return cv
}

// addBubbleBackground adds a white background rectangle.
func addBubbleBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(bubbleBgColour),
			Border:      inks.FixedInk(bubbleBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec:  bgSpec,
		W:     float64(width),
		H:     float64(height),
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}

// indexBubbleNodes recursively indexes all BubbleNodes by their Path,
// separating directories and files. Returns two maps.
func indexBubbleNodes(
	node *BubbleNode,
) (dirs map[string]*BubbleNode, files map[string]*BubbleNode) {
	dirs = make(map[string]*BubbleNode)
	files = make(map[string]*BubbleNode)

	indexBubbleNodesWalk(node, dirs, files)

	return dirs, files
}

func indexBubbleNodesWalk(
	node *BubbleNode,
	dirs map[string]*BubbleNode,
	files map[string]*BubbleNode,
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
	node *BubbleNode
}

// addBubbleDirDiscs collects directory nodes from the model tree (via path lookup),
// sorts them largest-first, and adds semi-transparent discs to the canvas.
func addBubbleDirDiscs(
	cv *canvas.Canvas,
	dirIndex map[string]*BubbleNode,
	root *model.Directory,
) {
	entries := collectBubbleDirEntries(dirIndex, root)

	slices.SortFunc(entries, func(a, b bubbleDirEntry) int {
		return cmp.Compare(b.node.Radius, a.node.Radius)
	})

	dirFill := inks.FixedInk(bubbleDefaultDirFill, inks.WithOpacity(bubbleDirOpacity))
	dirBorder := inks.FixedInk(bubbleDefaultBorder)

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
			Radius: bubbleDirDiscRadius(*e.node),
		})
	}
}

// collectBubbleDirEntries recursively walks model.Directory to find
// all directories that have a corresponding BubbleNode.
func collectBubbleDirEntries(
	dirIndex map[string]*BubbleNode,
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
	fileIndex map[string]*BubbleNode,
	root *model.Directory,
	is Inks,
) {
	addBubbleFileDiscsWalk(cv, fileIndex, root, is)
}

func addBubbleFileDiscsWalk(
	cv *canvas.Canvas,
	fileIndex map[string]*BubbleNode,
	dir *model.Directory,
	is Inks,
) {
	borderWidth := bubbleBorderWidth
	if is.HasBorderMetric {
		borderWidth = bubbleMetricBorderWidth
	}

	fileSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        is.Fill,
			Border:      is.Border,
			BorderWidth: borderWidth,
		},
	}

	for _, f := range dir.Files {
		bn, ok := fileIndex[f.Path]
		if !ok || bn.Radius <= 0 {
			continue
		}

		fillMV := inks.MetricValueForFile(f, is.Fill)
		borderMV := inks.MetricValueForFile(f, is.Border)

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
		addBubbleFileDiscsWalk(cv, fileIndex, d, is)
	}
}

// addBubbleLabels recursively adds labels for all nodes with ShowLabel set.
// Directory labels use arc text; file labels use centred text.
// Pre-allocates a shared labelInk and fileTextSpec so they are not re-created
// for every labelled node in the tree.
func addBubbleLabels(cv *canvas.Canvas, node BubbleNode) {
	labelInk := inks.FixedInk(bubbleLabelColour)
	fileTextSpec := &canvas.TextSpec{
		Ink:      labelInk,
		FontSize: 0,
		Anchor:   canvas.AnchorMiddle,
	}

	addBubbleLabelsInner(cv, node, labelInk, fileTextSpec)
}

// addBubbleLabelsInner is the recursive worker for addBubbleLabels.
// It accepts pre-allocated specs to avoid repeated allocations per node.
func addBubbleLabelsInner(cv *canvas.Canvas, node BubbleNode, labelInk inks.Ink, fileTextSpec *canvas.TextSpec) {
	if node.ShowLabel && node.Label != "" {
		if node.IsDirectory {
			addBubbleDirLabel(cv, node, labelInk)
		} else {
			addBubbleFileLabel(cv, node, fileTextSpec)
		}
	}

	for _, child := range node.Children {
		addBubbleLabelsInner(cv, child, labelInk, fileTextSpec)
	}
}

func bubbleDirDiscRadius(node BubbleNode) float64 {
	if node.IsDirectory && node.ShowLabel {
		return max(0.0, node.Radius-LabelReservation)
	}

	return node.Radius
}

func bubbleDirLabelRadius(node BubbleNode, fontSize float64) float64 {
	return bubbleDirDiscRadius(node) + fontSize/2 + bubbleArcLabelInset
}

func bubbleDirLabelFontSize(node BubbleNode) float64 {
	fontSize := bubbleArcFontSize(node.Label, bubbleDirLabelRadius(node, bubbleDefaultFontSize))
	if fontSize == 0 {
		return 0
	}

	return bubbleArcFontSize(node.Label, bubbleDirLabelRadius(node, fontSize))
}

// addBubbleDirLabel adds an arc text label curved just above the top of a directory circle.
func addBubbleDirLabel(cv *canvas.Canvas, node BubbleNode, labelInk inks.Ink) {
	fontSize := bubbleDirLabelFontSize(node)
	if fontSize == 0 {
		return
	}

	arcSpec := &canvas.ArcTextSpec{
		Ink:      labelInk,
		FontSize: fontSize,
	}

	cv.AddArcText(canvas.LayerOverlay, canvas.ArcText{
		Spec:   arcSpec,
		X:      node.X,
		Y:      node.Y,
		Radius: bubbleDirLabelRadius(node, fontSize),
		Text:   node.Label,
	})
}

// addBubbleFileLabel adds a centred text label on a file circle.
func addBubbleFileLabel(cv *canvas.Canvas, node BubbleNode, spec *canvas.TextSpec) {
	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    spec,
		X:       node.X,
		Y:       node.Y,
		Content: node.Label,
	})
}

// bubbleArcFontSize computes the font size for a label to fit within
// bubbleMaxArcFraction of the target arc. The radius includes the canvas
// backend's fixed arc inset. Returns 0 if the label cannot fit at the
// minimum readable font size.
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
