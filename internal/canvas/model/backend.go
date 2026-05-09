// Package types defines the shared types used by the canvas package
// and its backend implementations. It exists to break import cycles
// between the canvas package and backend packages.
package types

import (
"image/color"
)

// Backend is the rendering interface implemented by output format adapters.
// Methods receive resolved RGBA colours and primitive geometry.
type Backend interface {
DrawRectangle(pos Position, size Size, fill, border color.RGBA, borderWidth float64)
DrawDisc(center Position, radius float64, fill, border color.RGBA, borderWidth float64)
DrawLine(from, to Position, stroke color.RGBA, strokeWidth float64)
DrawPath(points []Position, stroke color.RGBA, strokeWidth float64)
DrawText(pos Position, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
DrawArcText(center Position, radius float64, text string, ink color.RGBA, fontSize float64)
Finish(outputPath string) error
}

// Position represents a 2D coordinate.
type Position struct {
X, Y float64
}

// Size represents a width and height.
type Size struct {
Width, Height float64
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
