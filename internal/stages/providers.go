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

	// New loader system for base metrics needed by expressions
	if err := provider.RunLoaders(c.Root, c.Requested.BaseMetrics, metricProg); err != nil {
		return eris.Wrap(err, "failed to load base metrics")
	}

	// Legacy system for backward-compat metrics
	if len(c.Requested.Legacy) > 0 {
		if err := provider.Run(c.Root, c.Requested.Legacy, metric.File, metricProg); err != nil {
			return eris.Wrap(err, "failed to load metrics")
		}
	}

	return nil
}
