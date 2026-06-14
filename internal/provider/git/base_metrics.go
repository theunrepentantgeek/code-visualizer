package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// GitProvider is the provider descriptor for git metrics.
var GitProvider = provider.ProviderDescriptor{
	Name:    "git",
	Filters: nil,
}

// RegisterBase adds git base metric descriptors to the global base registry.
func RegisterBase() {
	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileAge,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Time since first commit (days); older files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileFreshness,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Time since most recent commit (days); recently changed files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           AuthorCount,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of distinct commit authors; files touched by many people score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.GoodBad,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           CommitCount,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of commits that modified the file; frequently changed files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           TotalLinesAdded,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Lines added over all commits, excluding the initial commit; high-churn files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           TotalLinesRemoved,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Accumulated lines removed over all commits; high churn files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           CommitDensity,
		Kind:           metric.Measure,
		Level:          metric.LevelFile,
		Description:    "Commits per month of file lifetime; frequently changed files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax},
		DefaultPalette: palette.Temperature,
	}, GitProvider)
}
