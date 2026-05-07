package git

import (
	"github.com/bevan/code-visualizer/internal/provider"
)

// Register adds all git metric providers to the global registry.
func Register() {
	for name, def := range providerDefs {
		provider.Register(provider.MetricDescriptor{
			Name:           name,
			Kind:           def.kind,
			Description:    def.description,
			DefaultPalette: def.defaultPalette,
		}, newProvider(name))
	}
}
