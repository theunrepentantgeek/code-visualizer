package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/classification"
)

// RegisterSelectionMetrics registers a classification provider for each
// user-defined selection metric declared in the config. It must be called
// before any stage that resolves metric names (e.g. ResolveMetrics), so that
// user-defined metric names are present in the global registry.
//
// Already-registered metrics are silently skipped to allow multiple
// pipeline runs within a single process (e.g. in tests).
func RegisterSelectionMetrics(c *CommonState) error {
	if c.Flags == nil || c.Flags.Config == nil {
		return nil
	}

	for _, m := range c.Flags.Config.SelectionMetricsList() {
		if _, already := provider.GetBase(metric.Name(m.Name)); already {
			continue
		}

		classification.Register(m)
	}

	return nil
}
