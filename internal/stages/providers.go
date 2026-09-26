package stages

import (
	"context"
	"errors"
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// RunProviders calculates c.Requested metrics against c.Root.
//
//nolint:revive // Pipeline ApplyFuncXYZ fixes dependency order as state, context, sink.
func RunProviders(c *CommonState, ctx context.Context, sink progress.Sink) error {
	progressMetrics := metricsRemainingAfterPrewarm(c)
	total := provider.FileProgressTotal(progressMetrics, model.CountFiles(c.Root))
	metricProg := newMetricProgress(sink, total)

	err := loadRequestedMetrics(c, ctx, sink, filterMetricProgress(metricProg, progressMetrics))
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			return eris.Wrap(ctx.Err(), "metric loading cancelled")
		}

		return err
	}

	if err := metricProg.Err(); err != nil {
		return eris.Wrap(err, "report metric progress")
	}

	return nil
}

func metricsRemainingAfterPrewarm(c *CommonState) []metric.Name {
	if len(c.GitHistory) == 0 {
		return c.Requested.BaseMetrics
	}

	return withoutFileGitMetrics(c.Requested.BaseMetrics)
}

type metricProgressFilter struct {
	progress provider.MetricProgress
	selected map[metric.Name]struct{}
}

func filterMetricProgress(
	metricProgress provider.MetricProgress,
	selected []metric.Name,
) provider.MetricProgress {
	if metricProgress == nil {
		return nil
	}

	metrics := make(map[metric.Name]struct{}, len(selected))
	for _, name := range selected {
		metrics[name] = struct{}{}
	}

	return &metricProgressFilter{
		progress: metricProgress,
		selected: metrics,
	}
}

func (f *metricProgressFilter) OnMetricStarted(name metric.Name) {
	if _, ok := f.selected[name]; ok {
		f.progress.OnMetricStarted(name)
	}
}

func (f *metricProgressFilter) OnMetricFinished(name metric.Name) {
	if _, ok := f.selected[name]; ok {
		f.progress.OnMetricFinished(name)
	}
}

func (f *metricProgressFilter) OnFileProcessed(name metric.Name) {
	if _, ok := f.selected[name]; ok {
		f.progress.OnFileProcessed(name)
	}
}

//nolint:revive // Common state is the primary pipeline input.
func loadRequestedMetrics(
	c *CommonState,
	ctx context.Context,
	sink progress.Sink,
	metricProg provider.MetricProgress,
) error {
	c.Root.ReferenceTime = c.ReferenceNow

	requested := c.Requested.BaseMetrics
	if hasAuthorshipMetric(requested) {
		repoRoot, err := repoRootForState(c, "authorship metrics")
		if err != nil {
			return eris.Wrap(err, "failed to resolve git root")
		}

		total, err := git.CommitTotalInHistoryRange(ctx, repoRoot, c.Flags.HistoryRange)
		if err != nil {
			return eris.Wrap(err, "failed to count git commits")
		}

		historyProg := newHistoryProgress(sink, total)
		if err := git.LoadAuthorshipMetricsInHistoryRange(
			ctx,
			c.Root,
			authorshipParams(c.RootConfig),
			c.Flags.HistoryRange,
			c.ReferenceNow,
			historyProg.OnCommit,
		); err != nil {
			return eris.Wrap(err, "failed to load authorship metrics")
		}

		if err := historyProg.Err(); err != nil {
			return eris.Wrap(err, "report authorship progress")
		}

		requested = withoutAuthorshipMetrics(requested)
	}

	requested, err := loadFileGitMetrics(c, ctx, requested, metricProg)
	if err != nil {
		return err
	}

	return eris.Wrap(
		provider.RunLoaders(ctx, c.Root, requested, metricProg),
		"failed to load metrics",
	)
}

//nolint:revive // Common state is the primary pipeline input.
func loadFileGitMetrics(
	c *CommonState,
	ctx context.Context,
	requested []metric.Name,
	metricProg provider.MetricProgress,
) ([]metric.Name, error) {
	fileGitMetrics := onlyFileGitMetrics(requested)
	if len(fileGitMetrics) == 0 || len(c.GitHistory) > 0 {
		return requested, nil
	}

	onFile := func() {
		for _, name := range fileGitMetrics {
			metricProg.OnFileProcessed(name)
		}
	}

	if err := git.LoadFileMetricsInHistoryRange(
		ctx,
		c.Root,
		fileGitMetrics,
		c.Flags.HistoryRange,
		onFile,
		c.ReferenceNow,
	); err != nil {
		return nil, eris.Wrap(err, "failed to load git metrics")
	}

	return withoutFileGitMetrics(requested), nil
}

func onlyFileGitMetrics(names []metric.Name) []metric.Name {
	return slices.DeleteFunc(slices.Clone(names), func(name metric.Name) bool {
		return !git.IsGitMetric(name) || git.IsAuthorshipMetric(name)
	})
}

func withoutFileGitMetrics(names []metric.Name) []metric.Name {
	return slices.DeleteFunc(slices.Clone(names), func(name metric.Name) bool {
		return git.IsGitMetric(name) && !git.IsAuthorshipMetric(name)
	})
}

func hasAuthorshipMetric(names []metric.Name) bool {
	return slices.ContainsFunc(names, git.IsAuthorshipMetric)
}

func withoutAuthorshipMetrics(names []metric.Name) []metric.Name {
	result := make([]metric.Name, 0, len(names))
	for _, name := range names {
		if !git.IsAuthorshipMetric(name) {
			result = append(result, name)
		}
	}

	return result
}

func authorshipParams(cfg *config.Config) git.AuthorshipParams {
	params := git.DefaultAuthorshipParams()
	if cfg == nil || cfg.Authorship == nil {
		return params
	}

	authorship := cfg.Authorship

	if authorship.ActivityWindowDays != nil {
		params.ActivityWindowDays = *authorship.ActivityWindowDays
	}

	if authorship.RecentWindowDays != nil {
		params.RecentWindowDays = *authorship.RecentWindowDays
	}

	if authorship.EarlyWindowFraction != nil {
		params.EarlyWindowFraction = *authorship.EarlyWindowFraction
	}

	if authorship.SignificantShareThreshold != nil {
		params.SignificantShareThreshold = *authorship.SignificantShareThreshold
	}

	if authorship.BusFactorThreshold != nil {
		params.BusFactorThreshold = *authorship.BusFactorThreshold
	}

	if authorship.IdentityTopK != nil {
		params.IdentityTopK = *authorship.IdentityTopK
	}

	if authorship.HonorMailmap != nil {
		params.HonorMailmap = *authorship.HonorMailmap
	}

	return params
}
