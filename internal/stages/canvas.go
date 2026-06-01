package stages

import (
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
// A nil or hidden Footer leaves the canvas footer unset (no footer rendered).
func ApplyFooter[S VizState](s S) error {
	c := s.Common()
	if c.Canvas == nil || c.RootConfig == nil {
		return nil
	}

	footer := c.RootConfig.Footer
	if footer.IsHidden() {
		return nil
	}

	c.Canvas.SetFooter(footer.EffectiveText())

	return nil
}

var _ pipeline.Stage[VizState] = ApplyFooter[VizState]
