package canvas

import (
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/raster"
	svgbackend "github.com/theunrepentantgeek/code-visualizer/internal/canvas/svg"
)

// shapeKind tags the type of shape stored in a layered entry.
type shapeKind int

const (
	shapeRectangle shapeKind = iota
	shapeDisc
	shapeText
	shapeLine
	shapePath
	shapeArcText
)

// layeredShape holds a shape with its assigned layer and insertion order.
type layeredShape struct {
	layer   Layer
	order   int
	kind    shapeKind
	rect    *Rectangle
	disc    *Disc
	text    *Text
	line    *Line
	path    *Path
	arcText *ArcText
}

// Canvas is a retained-then-render drawing surface.
// Shapes are added with layer assignments, then rendered in batch.
type Canvas struct {
	width  int
	height int
	shapes []layeredShape
	legend *LegendConfig
}

// NewCanvas creates a canvas for the given dimensions.
func NewCanvas(width, height int) *Canvas {
	return &Canvas{
		width:  width,
		height: height,
	}
}

// AddRectangle records a rectangle on the given layer.
func (c *Canvas) AddRectangle(layer Layer, r Rectangle) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeRectangle,
		rect:  &r,
	})
}

// AddDisc records a disc on the given layer.
func (c *Canvas) AddDisc(layer Layer, d Disc) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeDisc,
		disc:  &d,
	})
}

// AddText records text on the given layer.
func (c *Canvas) AddText(layer Layer, t Text) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeText,
		text:  &t,
	})
}

// AddLine records a line on the given layer.
func (c *Canvas) AddLine(layer Layer, l Line) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeLine,
		line:  &l,
	})
}

// AddPath records a path on the given layer.
func (c *Canvas) AddPath(layer Layer, p Path) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapePath,
		path:  &p,
	})
}

// AddArcText records arc text on the given layer.
func (c *Canvas) AddArcText(layer Layer, a ArcText) {
	c.shapes = append(c.shapes, layeredShape{
		layer:   layer,
		order:   len(c.shapes),
		kind:    shapeArcText,
		arcText: &a,
	})
}

// SetLegend configures the legend overlay for this canvas.
func (c *Canvas) SetLegend(config LegendConfig) {
	c.legend = &config
}

// Render resolves all inks, sorts shapes by layer, selects the backend
// from the file extension, and writes the output.
func (c *Canvas) Render(outputPath string) error {
	format, err := FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	backend, err := c.createBackend(format)
	if err != nil {
		return err
	}

	if err := c.RenderTo(backend); err != nil {
		return err
	}

	if err := backend.Finish(outputPath); err != nil {
		return eris.Wrap(err, "backend finish failed")
	}

	return nil
}

// RenderTo dispatches all shapes to the given backend, sorted by layer.
// This method is the primary test seam — tests inject a mock backend.
func (c *Canvas) RenderTo(backend Backend) error {
	sorted := make([]layeredShape, len(c.shapes))
	copy(sorted, c.shapes)

	slices.SortStableFunc(sorted, func(a, b layeredShape) int {
		if a.layer != b.layer {
			return int(a.layer - b.layer)
		}

		return a.order - b.order
	})

	for _, s := range sorted {
		c.dispatchShape(backend, s)
	}

	if c.legend != nil {
		data := c.legend.toLegendData()
		if data != nil {
			backend.DrawLegend(*data, c.width, c.height)
		}
	}

	return nil
}

func (c *Canvas) dispatchShape(backend Backend, s layeredShape) {
	switch s.kind {
	case shapeRectangle:
		c.drawRectangle(backend, s.rect)
	case shapeDisc:
		c.drawDisc(backend, s.disc)
	case shapeText:
		c.drawText(backend, s.text)
	case shapeLine:
		c.drawLine(backend, s.line)
	case shapePath:
		c.drawPath(backend, s.path)
	case shapeArcText:
		c.drawArcText(backend, s.arcText)
	default:
		// No default case needed - shapeKind is exhaustively defined
	}
}

func (*Canvas) drawRectangle(b Backend, r *Rectangle) {
	fill := r.Spec.Fill.Dip(r.Fill)
	border := r.Spec.Border.Dip(r.Border)

	b.DrawRectangle(
		Position{X: r.X, Y: r.Y},
		Size{Width: r.W, Height: r.H},
		fill, border,
		r.Spec.BorderWidth,
	)
}

func (*Canvas) drawDisc(b Backend, d *Disc) {
	fill := d.Spec.Fill.Dip(d.Fill)
	border := d.Spec.Border.Dip(d.Border)

	b.DrawDisc(
		Position{X: d.X, Y: d.Y},
		d.Radius,
		fill, border,
		d.Spec.BorderWidth,
	)
}

func (*Canvas) drawText(b Backend, t *Text) {
	ink := t.Spec.Ink.Dip(MetricValue{})

	b.DrawText(
		Position{X: t.X, Y: t.Y},
		t.Content, ink,
		t.Spec.FontSize,
		t.Spec.Anchor,
		t.Spec.Rotation,
	)
}

func (*Canvas) drawLine(b Backend, l *Line) {
	stroke := l.Spec.Stroke.Dip(MetricValue{})

	b.DrawLine(
		Position{X: l.X1, Y: l.Y1},
		Position{X: l.X2, Y: l.Y2},
		stroke,
		l.Spec.StrokeWidth,
	)
}

func (*Canvas) drawPath(b Backend, p *Path) {
	stroke := p.Spec.Stroke.Dip(MetricValue{})

	b.DrawPath(p.Points, stroke, p.Spec.StrokeWidth)
}

func (*Canvas) drawArcText(b Backend, a *ArcText) {
	ink := a.Spec.Ink.Dip(MetricValue{})
	b.DrawArcText(
		Position{X: a.X, Y: a.Y},
		a.Radius,
		a.Text,
		ink,
		a.Spec.FontSize,
	)
}

// createBackend creates the appropriate backend for the given format.
// Backend subpackages are imported and instantiated here.
func (c *Canvas) createBackend(format ImageFormat) (Backend, error) {
	switch format {
	case FormatPNG, FormatJPG:
		return raster.New(c.width, c.height), nil
	case FormatSVG:
		return svgbackend.New(c.width, c.height), nil
	default:
		return nil, eris.Errorf("unsupported format: %d", format)
	}
}
