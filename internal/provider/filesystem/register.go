package filesystem

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all filesystem metric providers and loaders to the global registry.
func Register() {
	// Legacy providers (kept temporarily for backward compat)
	provider.Register(FileSizeProvider{})
	provider.Register(&FileLinesProvider{})
	provider.Register(FileTypeProvider{})

	RegisterBase()

	// New loader registrations
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
