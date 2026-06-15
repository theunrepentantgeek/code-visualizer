package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all Go metric providers and loaders to the global registry.
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}

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
