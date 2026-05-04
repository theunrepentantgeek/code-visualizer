package radialtree

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

// RadialNode is a positioned visual element in the rendered radial tree.
// X and Y are pixel offsets from the canvas centre (canvas centre = origin).
type RadialNode struct {
	X, Y         float64     // pixel position relative to canvas centre
	DiscRadius   float64     // radius of the node disc in pixels
	Angle        float64     // angle in radians (0 = right/east, π/2 = down, in screen coordinates)
	Label        string      // display name
	ShowLabel    bool        // whether to render the label for this node
	IsDirectory  bool        // true for directory nodes, false for file nodes
	FillColour   color.RGBA  // fill colour (zero value means use default)
	BorderColour *color.RGBA // border colour (nil means use default)
	Children     []RadialNode
}
