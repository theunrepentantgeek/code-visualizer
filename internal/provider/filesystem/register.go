package filesystem

import "github.com/bevan/code-visualizer/internal/provider"

// Register adds all filesystem metric providers to the global registry.
func Register() {
	provider.Register(FileSizeProvider{})
	provider.Register(&FileLinesProvider{})
	provider.Register(FileTypeProvider{})
}
