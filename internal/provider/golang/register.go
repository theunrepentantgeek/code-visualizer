package golang

import "github.com/theunrepentantgeek/code-visualizer/internal/provider"

// Register adds all Go metric providers to the global registry.
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}
}
