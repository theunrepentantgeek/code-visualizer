package treemap

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ApplyCanvasBlockLabels fits and adds the treemap's block labels to c.Canvas.
// No-op when c.Canvas is nil.
func ApplyCanvasBlockLabels(c *stages.CommonState, t *State) error {
	if c.Canvas == nil {
		return nil
	}

	format, err := canvas.FormatFromPath(c.Output)
	if err != nil {
		return eris.Wrap(err, "resolve canvas label format")
	}

	for _, label := range t.BlockLabels {
		c.Canvas.AddBlockLabel(canvas.LayerOverlay, label, format)
	}

	return nil
}
