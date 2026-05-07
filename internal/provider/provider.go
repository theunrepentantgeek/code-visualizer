// Package provider defines the metric provider interface, registry, and scheduler.
package provider

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// MetricDescriptor holds the static metadata for a metric provider.
// Metadata-only consumers (e.g. help display, legend building) work
// exclusively with MetricDescriptor without depending on execution logic.
type MetricDescriptor struct {
	Name           metric.Name
	Kind           metric.Kind
	Description    string
	Dependencies   []metric.Name
	DefaultPalette palette.PaletteName
}

// Loader executes a metric computation over a directory tree.
type Loader interface {
	Load(root *model.Directory) error
}
