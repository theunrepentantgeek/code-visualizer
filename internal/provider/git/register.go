package git

import "github.com/bevan/code-visualizer/internal/provider"

// Register adds all git metric providers to the global registry.
func Register() {
	provider.Register(&FileAgeProvider{})
	provider.Register(&FileFreshnessProvider{})
	provider.Register(&AuthorCountProvider{})
	provider.Register(&CommitCountProvider{})
	provider.Register(&TotalLinesAddedProvider{})
	provider.Register(&TotalLinesRemovedProvider{})
	provider.Register(&CommitDensityProvider{})
}
