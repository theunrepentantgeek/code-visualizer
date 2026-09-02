package donuttree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// DonutNode is a directory's annular sector in a donut tree.
type DonutNode struct {
	Directory   *model.Directory
	Depth       int
	StartAngle  float64
	SweepAngle  float64
	InnerRadius float64
	OuterRadius float64
	Children    []DonutNode
}

// EndAngle returns the angle at the end of the node's sector.
func (n DonutNode) EndAngle() float64 {
	return n.StartAngle + n.SweepAngle
}

// LayoutResult contains the central root anchor and its directory sectors.
type LayoutResult struct {
	RootName     string
	Center       geometry.Point
	AnchorRadius float64
	Children     []DonutNode
}
