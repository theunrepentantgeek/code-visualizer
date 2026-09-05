package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Text carries position and content for standalone text.
type Text struct {
	Spec     *TextSpec
	Position geometry.Point
	Content  string
}

func (t *Text) drawTo(b Backend) {
	ink := t.Spec.Ink.Dip(inks.MetricValue{})

	b.DrawText(
		t.Position,
		t.Content, ink,
		t.Spec.FontSize,
		t.Spec.Anchor,
		t.Spec.Rotation,
	)
}
