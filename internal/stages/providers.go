package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RunProviders calculates Common().Requested metrics against Common().Root,
// wiring progress reporting based on Flags verbosity.
func RunProviders[S VizState](s S) error {
	c := s.Common()

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))

	if err := provider.Run(c.Root, c.Requested, metricProg); err != nil {
		stopMetricTicker()

		return eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	return nil
}

var _ pipeline.Stage[VizState] = RunProviders[VizState]
