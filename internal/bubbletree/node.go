package bubbletree

import (
	"image/color"

	"github.com/bevan/code-visualizer/internal/viz"
)

// LabelMode is an alias for [viz.LabelMode].
type LabelMode = viz.LabelMode

const (
	LabelAll         = viz.LabelAll
	LabelFoldersOnly = viz.LabelFoldersOnly
	LabelNone        = viz.LabelNone
)

// BubbleNode is a positioned visual element in the rendered bubble tree.
// X and Y are absolute pixel coordinates after layout; Radius is the circle radius in pixels.
type BubbleNode struct {
	X, Y         float64     // centre position in pixels
	Radius       float64     // circle radius in pixels
	Path         string      // model path — stable identifier for colour mapping
	Label        string      // display name
	ShowLabel    bool        // whether to render the label for this node
	IsDirectory  bool        // true for directory nodes, false for file nodes
	FillColour   color.RGBA  // fill colour (zero value means use default)
	BorderColour *color.RGBA // border colour (nil means use default)
	Children     []BubbleNode
}
