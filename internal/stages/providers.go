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
	defer stopMetricTicker()

	requested := c.Requested.BaseMetrics
	if hasAuthorshipMetric(requested) {
		if err := git.LoadAuthorshipMetrics(c.Root, authorshipParams(c.RootConfig)); err != nil {
			return eris.Wrap(err, "failed to load authorship metrics")
		}

		requested = withoutAuthorshipMetrics(requested)
	}

	return eris.Wrap(
		provider.RunLoaders(c.Root, requested, metricProg),
		"failed to load metrics",
	)
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
