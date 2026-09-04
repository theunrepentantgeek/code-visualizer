package canvas

import (
	"image/color"
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// BenchmarkFitBlockLabel measures the cost of fitting a multi-line label into
// a rectangular area. The optimisation — replacing a 14-step binary search
// (one truetype.NewFace call per iteration) with a single reference
// measurement at 12pt followed by proportional scaling — reduces font-face
// allocations per label from 14 × nLines to 1 × nLines.
func BenchmarkFitBlockLabel(b *testing.B) {
	lines := []string{"internal/canvas", "block_label.go", "847"}

	b.ResetTimer()

	for range b.N {
		c := NewCanvas(200, 120)
		c.AddBlockLabel(LayerOverlay, BlockLabel{
			Bounds: geometry.Rect{Min: geometry.NewPoint(10, 10), Max: geometry.NewPoint(190, 110)},
			Lines:  lines,
			Ink:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}, FormatPNG)
	}
}
