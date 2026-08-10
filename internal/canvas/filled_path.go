package canvas

import "image/color"

// FilledPath carries closed loops to be filled as one borderless shape.
type FilledPath struct {
	Loops [][]Position
	Fill  color.RGBA
}

func (p *FilledPath) drawTo(b Backend) {
	b.DrawFilledPath(p.Loops, p.Fill)
}
