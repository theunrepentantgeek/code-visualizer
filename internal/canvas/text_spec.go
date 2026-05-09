package canvas

import (
	"github.com/bevan/code-visualizer/internal/canvas/types"
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

// TextAnchor is re-exported from types for backward compatibility.
type TextAnchor = types.TextAnchor

const (
	// AnchorStart aligns text to the left.
	AnchorStart = types.AnchorStart
	// AnchorMiddle centers text horizontally.
	AnchorMiddle = types.AnchorMiddle
	// AnchorEnd aligns text to the right.
	AnchorEnd = types.AnchorEnd
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

