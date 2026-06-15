package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all git metric providers and loaders to the global registry.
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}

	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{
			FileAge,
			FileFreshness,
			AuthorCount,
			CommitCount,
			TotalLinesAdded,
			TotalLinesRemoved,
			CommitDensity,
		},
		Load: loadAllFileMetrics,
	})
}
