package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/classification"
)

// RunProviders calculates c.Requested metrics against c.Root.
func RunProviders(c *CommonState) error {
	// Register any user-defined selection metrics from the effective config.
	// This is a no-op when no selection-metrics are configured, and is
	// idempotent so it is safe to call on every pipeline run.
	classification.Register(c.RootConfig)

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))
	defer stopMetricTicker()

	return eris.Wrap(
		provider.RunLoaders(c.Root, c.Requested.BaseMetrics, metricProg),
		"failed to load metrics",
	)
}
