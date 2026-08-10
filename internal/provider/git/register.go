package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// Register adds all git base metrics and loaders to the global registries.
func Register() {
	RegisterBase()

	loader := &metricsLoader{}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  fileMetricNames,
		Load:     loader.Load,
		Reporter: loader,
	})
}
