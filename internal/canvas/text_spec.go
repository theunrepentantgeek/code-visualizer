package canvas

import (
	"github.com/bevan/code-visualizer/internal/canvas/model"
)

// LabelStyle controls how labels are rendered on shapes.
type LabelStyle int

const (
	// LabelCentered places text centered inside the shape.
	LabelCentered LabelStyle = iota
	// LabelArc curves text along a circle boundary (used by bubble tree directories).
	LabelArc
	// LabelRadial places text outside the shape, rotated outward (used by radial/spiral).
	LabelRadial
)

// TextAnchor is re-exported from model for backward compatibility.
type TextAnchor = model.TextAnchor

const (
	// AnchorStart aligns text to the left.
	AnchorStart = model.AnchorStart
	// AnchorMiddle centers text horizontally.
	AnchorMiddle = model.AnchorMiddle
	// AnchorEnd aligns text to the right.
	AnchorEnd = model.AnchorEnd
)

// TextSpec defines the visual template for standalone text.
// Font family is intentionally fixed (sans-serif for SVG, goregular for raster)
// and is not exposed as a configurable field.
type TextSpec struct {
	Ink      Ink
	FontSize float64
	Anchor   TextAnchor
	Rotation float64 // radians
}
