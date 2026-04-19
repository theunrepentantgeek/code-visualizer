package bubbletree

import "image/color"

// LabelMode controls which node labels are shown in the diagram.
type LabelMode string

const (
	// LabelAll shows labels for all nodes.
	LabelAll LabelMode = "all"
	// LabelFoldersOnly shows labels for directory nodes only.
	LabelFoldersOnly LabelMode = "folders"
	// LabelNone hides all labels.
	LabelNone LabelMode = "none"
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
