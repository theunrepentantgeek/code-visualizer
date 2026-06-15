package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RunProviders calculates c.Requested metrics against c.Root.
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))
	defer stopMetricTicker()

	return eris.Wrap(provider.Run(c.Root, c.Requested.LegacyNames(), metric.File, metricProg), "failed to load metrics")
}
