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

	metricProg, stopMetricTicker := BuildMetricProgress(
		c.Flags,
		provider.FileProgressTotal(c.Requested.BaseMetrics, model.CountFiles(c.Root)),
	)

	metricsLoaded := false
	defer func() { stopMetricTicker(metricsLoaded) }()

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

	if err := provider.RunLoaders(c.Root, requested, metricProg); err != nil {
		return eris.Wrap(err, "failed to load metrics")
	}

	metricsLoaded = true

	return nil
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
