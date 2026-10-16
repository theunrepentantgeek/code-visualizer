package canvas

import "github.com/theunrepentantgeek/code-visualizer/internal/geometry"

// Tooltip carries a hover target and its text.
type Tooltip struct {
	Bounds   geometry.Rect
	Text     string
	Fallback []Text
}

func (t *Tooltip) drawTo(b Backend) {
	b.DrawTooltip(t.Bounds, t.Text)
}

func (t *Tooltip) fallbackText() []Text {
	return t.Fallback
}

// PolygonTooltip carries a polygonal hover target, popup position, and text.
type PolygonTooltip struct {
	Points   []geometry.Point
	Position geometry.Point
	Text     string
	Fallback []Text
}

func (t *PolygonTooltip) drawTo(b Backend) {
	b.DrawPolygonTooltip(t.Points, t.Position, t.Text)
}

func (t *PolygonTooltip) fallbackText() []Text {
	return t.Fallback
}
