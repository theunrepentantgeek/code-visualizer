package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all git base metrics and loaders to the global registries.
//
// Authorship metrics are NOT registered as a provider loader here. They are
// dispatched exclusively through stages.RunProviders → git.LoadAuthorshipMetrics
// so that configuration parameters (thresholds, window sizes) can be passed
// without embedding config in global mutable state.
func Register() {
	RegisterBase()

	// Single-pass file-metadata loader for the seven file metrics.
	loader := &metricsLoader{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  fileMetricNames,
		Load:     loader.Load,
		Reporter: loader,
	})
}
