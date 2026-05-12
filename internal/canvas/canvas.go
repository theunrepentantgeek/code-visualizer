package canvas

import (
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/raster"
	svgbackend "github.com/theunrepentantgeek/code-visualizer/internal/canvas/svg"
)

// drawnShape is implemented by every concrete shape that can be rendered.
type drawnShape interface {
	drawTo(backend Backend)
}

// layeredShape holds a shape with its assigned layer and insertion order.
type layeredShape struct {
	layer Layer
	order int
	shape drawnShape
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
		shape: &r,
	})
}

// AddDisc records a disc on the given layer.
func (c *Canvas) AddDisc(layer Layer, d Disc) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		shape: &d,
	})
}

// AddText records text on the given layer.
func (c *Canvas) AddText(layer Layer, t Text) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		shape: &t,
	})
}

// AddLine records a line on the given layer.
func (c *Canvas) AddLine(layer Layer, l Line) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		shape: &l,
	})
}

// AddPath records a path on the given layer.
func (c *Canvas) AddPath(layer Layer, p Path) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		shape: &p,
	})
}

// AddArcText records arc text on the given layer.
func (c *Canvas) AddArcText(layer Layer, a ArcText) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		shape: &a,
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
// Legend shapes are decomposed into primitives and merged into the shape
// list before dispatch.
// This method is the primary test seam — tests inject a mock backend.
func (c *Canvas) RenderTo(backend Backend) error {
	allShapes := make([]layeredShape, 0, len(c.shapes))
	allShapes = append(allShapes, c.shapes...)

	if c.legend != nil {
		allShapes = append(allShapes, c.decomposeLegend()...)
	}

	slices.SortStableFunc(allShapes, func(a, b layeredShape) int {
		if a.layer != b.layer {
			return int(a.layer - b.layer)
		}

		return a.order - b.order
	})

	for _, s := range allShapes {
		s.shape.drawTo(backend)
	}

	return nil
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
