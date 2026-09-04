// Package model defines the shared types used by the canvas package
// and its backend implementations. It exists to break import cycles
// between the canvas package and backend packages.
package model

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// Backend is the rendering interface implemented by output format adapters.
// Methods receive resolved colours/fills and primitive geometry.
type Backend interface {
	DrawRectangle(pos geometry.Point, size geometry.Size, fill, border Fill, borderWidth float64)
	DrawDisc(center geometry.Point, radius float64, fill, border Fill, borderWidth float64)
	DrawPolygon(points []geometry.Point, fill, border Fill, borderWidth float64)
	DrawFilledPath(loops [][]geometry.Point, fill color.RGBA)
	DrawLine(from, to geometry.Point, stroke color.RGBA, strokeWidth float64)
	DrawPath(points []geometry.Point, stroke color.RGBA, strokeWidth float64)
	DrawText(pos geometry.Point, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
	DrawArcText(center geometry.Point, radius float64, text string, ink color.RGBA, fontSize float64)
	Finish(outputPath string) error
}

// TextAnchor controls horizontal text alignment.
type TextAnchor int

const (
	// AnchorStart aligns text to the left.
	AnchorStart TextAnchor = iota
	// AnchorMiddle centers text horizontally.
	AnchorMiddle
	// AnchorEnd aligns text to the right.
	AnchorEnd
)

// DefaultFontSize signals that the backend should use its built-in default
// font size. Callers can set FontSize to this value instead of a bare 0.
const DefaultFontSize float64 = 0

// ArcTextInset is the fixed inset applied by canvas backends when drawing text
// along a circle arc. Callers that need layout-aware arc geometry should use
// the same value so reserved label space matches rendered output.
const ArcTextInset float64 = 14.0
