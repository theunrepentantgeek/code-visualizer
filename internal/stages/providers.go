package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RunProviders calculates c.Requested metrics against c.Root.
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, c.Root.AllFileCount)
	defer stopMetricTicker()

	return eris.Wrap(
		provider.RunLoaders(c.Root, c.Requested.BaseMetrics, metricProg),
		"failed to load metrics",
	)
}
