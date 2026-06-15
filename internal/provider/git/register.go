package git

import "github.com/theunrepentantgeek/code-visualizer/internal/provider"

// Register adds all git metric providers to the global registry.
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}

	RegisterBase()
}
