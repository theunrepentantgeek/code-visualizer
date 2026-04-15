package folder

import "github.com/bevan/code-visualizer/internal/provider"

// Register adds all folder metric providers to the global registry.
func Register() {
	provider.Register(&FolderAuthorCountProvider{})
	provider.Register(&FolderAgeProvider{})
	provider.Register(&FolderFreshnessProvider{})
	provider.Register(&TotalFolderLinesProvider{})
	provider.Register(&TotalFolderSizeProvider{})
	provider.Register(&MeanFileAgeProvider{})
	provider.Register(&MeanFileFreshnessProvider{})
	provider.Register(&MeanFileLinesProvider{})
	provider.Register(&MeanFileSizeProvider{})
}
