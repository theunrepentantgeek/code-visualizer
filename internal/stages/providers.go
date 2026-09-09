package stages

import (
	"log/slog"
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// RunProviders calculates c.Requested metrics against c.Root.
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")

	progressMetrics := metricsRemainingAfterPrewarm(c)
	total := provider.FileProgressTotal(progressMetrics, model.CountFiles(c.Root))
	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, total)

	err := loadRequestedMetrics(c, filterMetricProgress(metricProg, progressMetrics))

	stopMetricTicker()

	if err != nil {
		return err
	}

	if metricProg != nil {
		logMetricCompletion(total)
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
	progress provider.MetricProgress,
	selected []metric.Name,
) provider.MetricProgress {
	if progress == nil {
		return nil
	}

	metrics := make(map[metric.Name]struct{}, len(selected))
	for _, name := range selected {
		metrics[name] = struct{}{}
	}

	return &metricProgressFilter{
		progress: progress,
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

func loadRequestedMetrics(c *CommonState, metricProg provider.MetricProgress) error {
	requested := c.Requested.BaseMetrics
	if hasAuthorshipMetric(requested) {
		if err := git.LoadAuthorshipMetricsInHistoryRange(
			c.Root,
			authorshipParams(c.RootConfig),
			c.Flags.HistoryRange,
		); err != nil {
			return eris.Wrap(err, "failed to load authorship metrics")
		}

		requested = withoutAuthorshipMetrics(requested)
	}

	requested, err := loadFileGitMetrics(c, requested, metricProg)
	if err != nil {
		return err
	}

	return eris.Wrap(
		provider.RunLoaders(c.Root, requested, metricProg),
		"failed to load metrics",
	)
}

func loadFileGitMetrics(
	c *CommonState,
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
