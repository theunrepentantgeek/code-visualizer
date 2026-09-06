package filesystem

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all filesystem base metrics and loaders to the global registries.
func Register() {
	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileSize},
		Load: func(root *model.Directory, _ []metric.Name) error {
			return FileSizeProvider{}.Load(root)
		},
	})

	fileLinesProvider := &FileLinesProvider{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileLines},
		Load: func(root *model.Directory, _ []metric.Name) error {
			return fileLinesProvider.Load(root)
		},
		Reporter: fileLinesProvider,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileType},
		Load: func(root *model.Directory, _ []metric.Name) error {
			return FileTypeProvider{}.Load(root)
		},
	})
}
