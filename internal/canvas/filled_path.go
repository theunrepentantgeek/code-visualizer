package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// FilledPath carries closed loops to be filled as one borderless shape.
type FilledPath struct {
	Loops [][]geometry.Point
	Fill  color.RGBA
}

func (p *FilledPath) drawTo(b Backend) {
	b.DrawFilledPath(p.Loops, p.Fill)
}
