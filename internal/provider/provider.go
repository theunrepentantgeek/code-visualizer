// Package provider defines the metric provider interface, registry, and scheduler.
package provider

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// MetricDescriptor holds the static metadata for a metric provider.
// This is the narrowest type for consumers that only need provider metadata
// (e.g., help commands, UI lists, legend defaults).
type MetricDescriptor struct {
	Name           metric.Name
	Kind           metric.Kind
	Description    string
	Dependencies   []metric.Name
	DefaultPalette palette.PaletteName
}

// Loader is the execution interface for metric providers.
// Providers implement this to populate metrics on a directory tree.
type Loader interface {
	Load(root *model.Directory) error
}

// Interface is the combined contract every metric provider implements.
// It embeds both metadata and execution concerns.
//
// Deprecated: prefer accepting MetricDescriptor or Loader where possible.
type Interface interface {
	Name() metric.Name
	Kind() metric.Kind
	Description() string
	Dependencies() []metric.Name
	DefaultPalette() palette.PaletteName
	Loader
}

// Descriptor extracts the MetricDescriptor from a provider Interface.
// This allows registry code to continue storing Interface while consumers
// can receive only the metadata they need.
func Descriptor(p Interface) MetricDescriptor {
	return MetricDescriptor{
		Name:           p.Name(),
		Kind:           p.Kind(),
		Description:    p.Description(),
		Dependencies:   p.Dependencies(),
		DefaultPalette: p.DefaultPalette(),
	}
}
