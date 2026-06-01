package stages

import (
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

// WriteCanvas writes Common().Canvas to Common().Output.
func WriteCanvas[S VizState](s S) error {
	c := s.Common()
	if err := c.Canvas.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	return nil
}

var _ pipeline.Stage[VizState] = WriteCanvas[VizState]

// ApplyFooter sets the footer on Common().Canvas from RootConfig.Footer.
// If the Footer is hidden, the canvas footer is left unset (no footer rendered).
// If the Footer is nil or has no explicit text, the built-in default text is used.
func ApplyFooter[S VizState](s S) error {
	c := s.Common()
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

var _ pipeline.Stage[VizState] = ApplyFooter[VizState]
