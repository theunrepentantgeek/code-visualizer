package filesystem

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider"
)

// Register adds all filesystem metric providers to the global registry.
func Register() {
	provider.Register(provider.MetricDescriptor{
		Name:           FileSize,
		Kind:           metric.Quantity,
		Description:    "Size of each file in bytes.",
		DefaultPalette: palette.Neutral,
	}, FileSizeProvider{})

	provider.Register(provider.MetricDescriptor{
		Name:           FileLines,
		Kind:           metric.Quantity,
		Description:    "Number of lines in each text file.",
		DefaultPalette: palette.Neutral,
	}, &FileLinesProvider{})

	provider.Register(provider.MetricDescriptor{
		Name:           FileType,
		Kind:           metric.Classification,
		Description:    "File extension category (e.g. go, md, png).",
		DefaultPalette: palette.Categorization,
	}, FileTypeProvider{})
}
