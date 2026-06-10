package raster

import (
	"image/color"
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// BenchmarkDrawRadialGradientRect measures the cost of rendering a single
// radial-gradient rectangle — the hot path for treemap / bubbletree PNG output.
func BenchmarkDrawRadialGradientRect(b *testing.B) {
	grad := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 200, A: 255},
		Edge:   color.RGBA{R: 80, G: 80, B: 60, A: 255},
		Focus:  model.Point{X: 0.3, Y: 0.3},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	b.ResetTimer()

	for range b.N {
		be := New(200, 200)
		be.DrawRectangle(
			model.Position{X: 10, Y: 10},
			model.Size{Width: 180, Height: 180},
			grad, border, 1.0,
		)
	}
}

// BenchmarkDrawRadialGradientDisc measures the cost of rendering a single
// radial-gradient disc — the hot path for bubbletree / spiral PNG output.
func BenchmarkDrawRadialGradientDisc(b *testing.B) {
	grad := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 200, B: 100, A: 255},
		Edge:   color.RGBA{R: 80, G: 60, B: 30, A: 255},
		Focus:  model.Point{X: 0.4, Y: 0.4},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	b.ResetTimer()

	for range b.N {
		be := New(200, 200)
		be.DrawDisc(
			model.Position{X: 100, Y: 100},
			80,
			grad, border, 1.0,
		)
	}
}

// BenchmarkMaxCornerDist measures the cost of computing the maximum
// corner distance — called once per gradient rectangle.
func BenchmarkMaxCornerDist(b *testing.B) {
	for range b.N {
		_ = maxCornerDist(30, 30, 10, 10, 180, 180)
	}
}
