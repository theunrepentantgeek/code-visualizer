package stages

import (
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

// WriteCanvas writes c.Canvas to c.Output.
func WriteCanvas(c *CommonState) error {
	// Only override the canvas drawing bounds when they have been explicitly set
	// via InitDrawingBounds. If the pipeline skipped that stage (e.g., in tests),
	// the canvas keeps its constructor default (full dimensions).
	if c.DrawingBounds.MaxY > 0 {
		c.Canvas.SetDrawingBounds(c.DrawingBounds.MinY, c.DrawingBounds.MaxY)
	}

	if err := c.Canvas.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	return nil
}

// ApplyFooter sets the footer on c.Canvas from RootConfig.Footer.
// If the Footer is hidden, the canvas footer is left unset.
// If the Footer is nil or has no explicit text, the built-in default text is used.
func ApplyFooter(c *CommonState) error {
	if c.Canvas == nil || c.RootConfig == nil {
		return nil
	}

	footer := c.RootConfig.Footer
	if !footer.ShowFooter() {
		return nil
	}

	now := time.Now()
	rep := strings.NewReplacer(
		"$date", now.Format(time.DateOnly),
		"$time", now.Format(time.TimeOnly),
	)

	text := rep.Replace(*footer.Text)
	c.Canvas.SetFooter(text)

	return nil
}

// EffectiveFooterHeight returns the number of pixels that the footer occupies
// when rendered. Returns 0 when cfg is nil or the footer is not shown.
func EffectiveFooterHeight(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}

	if !cfg.Footer.ShowFooter() {
		return 0
	}

	return int(canvas.FooterReservedHeight)
}

// ApplyTitle sets the title on c.Canvas from RootConfig.Title.
// If the Title is nil, hidden, or has no text, the canvas title is left unset.
func ApplyTitle(c *CommonState) error {
	if c.Canvas == nil || c.RootConfig == nil {
		return nil
	}

	title := c.RootConfig.Title
	if !title.ShowTitle() {
		return nil
	}

	c.Canvas.SetTitle(*title.Text)

	return nil
}

// EffectiveTitleHeight returns the number of pixels that the title occupies
// when rendered. Layout stages subtract this from the top of the available
// area so that visualisation content does not overlap the title.
// Returns 0 when cfg is nil or the title is not shown.
func EffectiveTitleHeight(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}

	if !cfg.Title.ShowTitle() {
		return 0
	}

	return int(canvas.TitleReservedHeight)
}

// InitDrawingBounds initializes c.DrawingBounds to the full canvas dimensions.
// Run this immediately after ResolveDimensions, before any Reserve* stages.
func InitDrawingBounds(c *CommonState) error {
	c.DrawingBounds = DrawingBounds{MaxX: c.Width, MaxY: c.Height}

	return nil
}

// ReserveTitleBounds shrinks DrawingBounds.MinY to reserve space for the title.
// Must run after InitDrawingBounds. No-op when no title is configured or shown.
func ReserveTitleBounds(c *CommonState) error {
	if c.RootConfig == nil || !c.RootConfig.Title.ShowTitle() {
		return nil
	}

	c.DrawingBounds.MinY = int(canvas.TitleReservedHeight)

	return nil
}

// ReserveFooterBounds shrinks DrawingBounds.MaxY to reserve space for the footer.
// Must run after InitDrawingBounds. No-op when no footer is configured or shown.
func ReserveFooterBounds(c *CommonState) error {
	if c.RootConfig == nil || !c.RootConfig.Footer.ShowFooter() {
		return nil
	}

	c.DrawingBounds.MaxY -= int(canvas.FooterReservedHeight)

	return nil
}
