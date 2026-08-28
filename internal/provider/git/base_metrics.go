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
	registerFileMetrics()
	registerAuthorshipMetrics()
	registerCommitMetrics()
}

func registerFileMetrics() {
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
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)
}

func registerCommitMetrics() {
	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           LinesAdded,
		Kind:           metric.Quantity,
		Level:          metric.LevelCommit,
		Description:    "Lines added in a single commit.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           LinesRemoved,
		Kind:           metric.Quantity,
		Level:          metric.LevelCommit,
		Description:    "Lines removed in a single commit.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           LinesChanged,
		Kind:           metric.Quantity,
		Level:          metric.LevelCommit,
		Description:    "Lines changed (added + removed) in a single commit.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)
}

//nolint:funlen // Nine metrics, each with a brief descriptor; splitting would obscure the structure.
func registerAuthorshipMetrics() {
	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:  InitialDeveloperMetric,
		Kind:  metric.Classification,
		Level: metric.LevelFile,
		Description: "Greatest-weight author within the early window " +
			"(first earlyWindowFraction of the node's lifetime). Who started it.",
		Aggregations:   []metric.AggregationName{metric.AggMode},
		DefaultPalette: palette.Categorization,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:  CurrentMaintainerMetric,
		Kind:  metric.Classification,
		Level: metric.LevelFile,
		Description: "Greatest-weight author within the recent window " +
			"(recentWindowDays before HEAD); «unmaintained» if none. Who tends it now.",
		Aggregations:   []metric.AggregationName{metric.AggMode},
		DefaultPalette: palette.Categorization,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           CodeOwnerMetric,
		Kind:           metric.Classification,
		Level:          metric.LevelFile,
		Description:    "Greatest lifetime-weight author. Who has done the most overall.",
		Aggregations:   []metric.AggregationName{metric.AggMode},
		DefaultPalette: palette.Categorization,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           SignificantContributorCountMetric,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of authors with share Sₐ ≥ significantShareThreshold.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.GoodBad,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:  BusFactorMetric,
		Kind:  metric.Quantity,
		Level: metric.LevelFile,
		Description: "Smallest number of top authors whose combined share reaches " +
			"busFactorThreshold; 1 = single point of knowledge.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.GoodBad,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           OwnershipDominanceMetric,
		Kind:           metric.Measure,
		Level:          metric.LevelFile,
		Description:    "Maximum per-author share max(Sₐ); 1.0 = one owner.",
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           ContributorEntropyMetric,
		Kind:           metric.Measure,
		Level:          metric.LevelFile,
		Description:    "Normalised Shannon entropy of per-author shares; 0 = one owner, →1 = evenly shared.",
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.GoodBad,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           OrphanRiskMetric,
		Kind:           metric.Measure,
		Level:          metric.LevelFile,
		Description:    "Summed share of authors not active repo-wide within activityWindowDays of HEAD.",
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:  KnowledgeHandoffMetric,
		Kind:  metric.Measure,
		Level: metric.LevelFile,
		Description: "Share of recent-window contribution from authors absent " +
			"in the early window; 0 for young nodes.",
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)
}
