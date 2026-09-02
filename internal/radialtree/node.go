package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// LabelMode is an alias for [viz.LabelMode].
type LabelMode = viz.LabelMode

const (
	LabelAll         = viz.LabelAll
	LabelFoldersOnly = viz.LabelFoldersOnly
	LabelNone        = viz.LabelNone
)

// Grain is an alias for [viz.Grain].
type Grain = viz.Grain

const (
	GrainFile      = viz.GrainFile
	GrainDirectory = viz.GrainDirectory
)

// RadialNode is a positioned visual element in the rendered radial tree.
// Position is a pixel offset vector from the canvas centre (canvas centre = origin).
type RadialNode struct {
	Position    geometry.Vector
	DiscRadius  float64 // radius of the node disc in pixels
	Angle       float64 // angle in radians (0 = right/east, π/2 = down, in screen coordinates)
	Label       string  // display name
	ShowLabel   bool    // whether to render the label for this node
	IsDirectory bool    // true for directory nodes, false for file nodes
	Children    []RadialNode
}
