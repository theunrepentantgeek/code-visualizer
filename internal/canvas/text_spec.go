package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
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
	Ink      inks.Ink
	FontSize float64
	Anchor   TextAnchor
	Rotation float64 // radians
}

// ArcTextSpec defines the visual template for text curved along a circle arc.
type ArcTextSpec struct {
	Ink      inks.Ink
	FontSize float64
}

// ArcText carries position and content for text curved along a circle arc.
type ArcText struct {
	Spec   *ArcTextSpec
	X, Y   float64 // circle centre
	Radius float64 // reference arc radius; backends apply their fixed inset from this value
	Text   string
}

func (a *ArcText) drawTo(b Backend) {
	ink := a.Spec.Ink.Dip(inks.MetricValue{})

	b.DrawArcText(
		Position{X: a.X, Y: a.Y},
		a.Radius,
		a.Text,
		ink,
		a.Spec.FontSize,
	)
}
