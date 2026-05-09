package canvas

import (
	"image/color"
)

// Backend is the rendering interface implemented by output format adapters.
// Methods receive resolved RGBA colours and primitive geometry —
// no Inks, Specs, or MetricValues.
type Backend interface {
	DrawRectangle(pos Position, size Size, fill, border color.RGBA, borderWidth float64)
	DrawDisc(center Position, radius float64, fill, border color.RGBA, borderWidth float64)
	DrawLine(from, to Position, stroke color.RGBA, strokeWidth float64)
	DrawPath(points []Position, stroke color.RGBA, strokeWidth float64)
	DrawText(pos Position, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
	DrawArcText(center Position, radius float64, text string, ink color.RGBA, fontSize float64)
	Finish(outputPath string) error
}
