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
