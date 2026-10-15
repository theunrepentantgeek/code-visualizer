package canvas

import "github.com/theunrepentantgeek/code-visualizer/internal/geometry"

// Tooltip carries a hover target and its text.
type Tooltip struct {
	Bounds geometry.Rect
	Text   string
}

func (t *Tooltip) drawTo(b Backend) {
	b.DrawTooltip(t.Bounds, t.Text)
}
