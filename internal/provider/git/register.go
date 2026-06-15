package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all git base metrics and loaders to the global registries.
func Register() {
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
