package filesystem

import (
	"context"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all filesystem base metrics and loaders to the global registries.
func Register() {
	RegisterBase()

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileSize},
		Load: func(ctx context.Context, root *model.Directory, _ []metric.Name) error {
			return FileSizeProvider{}.Load(ctx, root)
		},
	})

	fileLinesProvider := &FileLinesProvider{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileLines},
		Load: func(ctx context.Context, root *model.Directory, _ []metric.Name) error {
			return fileLinesProvider.Load(ctx, root)
		},
		Reporter: fileLinesProvider,
	})
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{FileType},
		Load: func(ctx context.Context, root *model.Directory, _ []metric.Name) error {
			return FileTypeProvider{}.Load(ctx, root)
		},
	})
}
