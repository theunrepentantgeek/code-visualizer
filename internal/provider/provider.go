// Package provider defines the metric provider interface, registry, and scheduler.
package provider

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Interface is the contract every metric provider implements.
type Interface interface {
	Name() metric.Name
	Kind() metric.Kind
	Description() string
	Dependencies() []metric.Name
	DefaultPalette() palette.PaletteName
	Load(root *model.Directory) error
}
