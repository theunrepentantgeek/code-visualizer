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

// PolygonTooltip carries a polygonal hover target, popup position, and text.
type PolygonTooltip struct {
	Points   []geometry.Point
	Position geometry.Point
	Text     string
}

func (t *PolygonTooltip) drawTo(b Backend) {
	b.DrawPolygonTooltip(t.Points, t.Position, t.Text)
}
