package canvas

import (
	"image/color"
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/raster"
	svgbackend "github.com/theunrepentantgeek/code-visualizer/internal/canvas/svg"
)

const (
	footerFontSize = 13.0
	footerMarginY  = 14.0

	// FooterReservedHeight is the vertical space (in pixels) that the footer
	// occupies when rendered. Layout stages subtract this from the available
	// height when the footer is enabled, preventing content from being drawn
	// underneath it.
	FooterReservedHeight = footerFontSize + footerMarginY

	titleFontSize = 18.0
	titleMarginY  = 20.0

	// TitleReservedHeight is the vertical space (in pixels) that the title
	// occupies when rendered. Layout stages subtract this from the available
	// height (offset from the top) when the title is enabled.
	TitleReservedHeight = titleFontSize + titleMarginY
)

var (
	footerColor = color.RGBA{R: 128, G: 128, B: 128, A: 200}
	titleColor  = color.RGBA{R: 40, G: 40, B: 40, A: 255}
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
	width       int
	height      int
	shapes      []layeredShape
	legend      *LegendConfig
	title       *string
	footer      *string
	drawingMinY int // top of content area (pixels below title)
	drawingMaxY int // bottom of content area (pixels above footer)
}

// NewCanvas creates a canvas for the given dimensions.
func NewCanvas(width, height int) *Canvas {
	return &Canvas{
		width:       width,
		height:      height,
		drawingMaxY: height, // default: full canvas
	}
}

// SetDrawingBounds stores the vertical drawing bounds for legend placement.
// topY is the first pixel available for content (0 unless there's a title).
// bottomY is the last+1 pixel available (height unless there's a footer).
// Call this before Render so top-center / bottom-center legends are placed
// below the title and above the footer respectively.
func (c *Canvas) SetDrawingBounds(topY, bottomY int) {
	c.drawingMinY = topY
	c.drawingMaxY = bottomY
}

// DrawingMinY returns the topmost Y pixel available for non-title content.
func (c *Canvas) DrawingMinY() int { return c.drawingMinY }

// DrawingMaxY returns the bottommost Y pixel (exclusive) available for
// non-footer content.
func (c *Canvas) DrawingMaxY() int { return c.drawingMaxY }

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

// FooterText returns the current footer text, or an empty string if no footer
// has been set. Primarily useful for testing.
func (c *Canvas) FooterText() string {
	if c.footer == nil {
		return ""
	}

	return *c.footer
}

// SetFooter configures the attribution footer text for this canvas.
// An empty string clears a previously set footer.
func (c *Canvas) SetFooter(text string) {
	if text == "" {
		c.footer = nil

		return
	}

	c.footer = &text
}

// TitleText returns the current title text, or an empty string if no title
// has been set. Primarily useful for testing.
func (c *Canvas) TitleText() string {
	if c.title == nil {
		return ""
	}

	return *c.title
}

// SetTitle configures the title text for this canvas.
// An empty string clears a previously set title.
func (c *Canvas) SetTitle(text string) {
	if text == "" {
		c.title = nil

		return
	}

	c.title = &text
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

	if c.title != nil {
		pos := model.Position{
			X: float64(c.width) / 2,
			Y: titleMarginY,
		}
		backend.DrawText(pos, *c.title, titleColor, titleFontSize, model.AnchorMiddle, 0)
	}

	if c.footer != nil {
		pos := model.Position{
			X: float64(c.width) / 2,
			Y: float64(c.height) - footerMarginY,
		}
		backend.DrawText(pos, *c.footer, footerColor, footerFontSize, model.AnchorMiddle, 0)
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
