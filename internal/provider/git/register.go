package git

import "github.com/bevan/code-visualizer/internal/provider"

// Register adds all git metric providers to the global registry.
func Register() {
	for i := range providerDefs {
		provider.Register(newProvider(providerDefs[i].name))
	}
}
