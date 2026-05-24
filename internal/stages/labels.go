package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

// CanvasLabelledState extends VizState with interior labels to add after the
// visualization canvas has been populated with its base shapes.
type CanvasLabelledState interface {
	VizState
	CanvasLabels() []canvas.BlockLabel
}

// ApplyCanvasBlockLabels fits and adds the state's block labels to Common().Canvas.
func ApplyCanvasBlockLabels[S CanvasLabelledState](s S) error {
	c := s.Common()
	if c.Canvas == nil {
		return nil
	}

	format, err := canvas.FormatFromPath(c.Output)
	if err != nil {
		return eris.Wrap(err, "resolve canvas label format")
	}

	for _, label := range s.CanvasLabels() {
		c.Canvas.AddBlockLabel(canvas.LayerOverlay, label, format)
	}

	return nil
}

var _ pipeline.Stage[CanvasLabelledState] = ApplyCanvasBlockLabels[CanvasLabelledState]
