package filesystem

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all filesystem base metrics and loaders to the global registries.
func Register() {
	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileSize},
		Load:    FileSizeProvider{}.Load,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileLines},
		Load:    (&FileLinesProvider{}).Load,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileType},
		Load:    FileTypeProvider{}.Load,
	})
}
