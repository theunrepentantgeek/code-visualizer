package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
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
	X, Y        float64 // centre position in pixels
	Radius      float64 // circle radius in pixels
	Path        string  // model path — stable identifier for colour mapping
	Label       string  // display name
	ShowLabel   bool    // whether to render the label for this node
	IsDirectory bool    // true for directory nodes, false for file nodes
	Children    []BubbleNode
}

// Index builds path-based lookup maps for all descendant BubbleNodes,
// separating directories from files. The root node itself is not included.
func (n *BubbleNode) Index() (dirs map[string]*BubbleNode, files map[string]*BubbleNode) {
	dirs = make(map[string]*BubbleNode)
	files = make(map[string]*BubbleNode)
	n.walkIndex(dirs, files)

	return dirs, files
}

func (n *BubbleNode) walkIndex(dirs, files map[string]*BubbleNode) {
	for i := range n.Children {
		child := &n.Children[i]
		if child.IsDirectory {
			dirs[child.Path] = child
			child.walkIndex(dirs, files)
		} else {
			files[child.Path] = child
		}
	}
}
