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
