package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all Go base metrics and loaders to the global registries.
func Register() {
	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{
			Imports,
			CommentRatio,
			stdlibImportsMetric,
			externalImportsMetric,
			internalImportsMetric,
		},
		Load: loadFileMetrics,
	})
}
