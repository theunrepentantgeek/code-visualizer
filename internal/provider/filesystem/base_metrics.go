package filesystem

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// FilesystemProvider is the provider descriptor for filesystem metrics.
var FilesystemProvider = provider.ProviderDescriptor{
	Name:    "filesystem",
	Filters: nil,
}

// RegisterBase adds filesystem base metric descriptors to the global base registry.
func RegisterBase() {
	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileSize,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Size of each file in bytes.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, FilesystemProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileLines,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of lines in each text file.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, FilesystemProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileType,
		Kind:           metric.Classification,
		Level:          metric.LevelFile,
		Description:    "File extension category (e.g. go, md, png).",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggMode, metric.AggDistinct},
		DefaultPalette: palette.Categorization,
	}, FilesystemProvider)
}
