package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all git base metrics and loaders to the global registries.
func Register() {
	RegisterBase()

	// Existing single-pass file-metadata loader.
	loader := &metricsLoader{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  fileMetricNames,
		Load:     loader.Load,
		Reporter: loader,
	})

	// Authorship loader: separate BulkAuthorHistory walk for the nine
	// authorship metrics.
	al := newAuthorshipLoader()
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: authorshipMetricNames,
		Load:    al.Load,
	})
}
