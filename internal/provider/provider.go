// Package provider defines the metric provider interface, registry, and scheduler.
package provider

import (
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// Scope describes what a metric applies to.
type Scope string

const (
	ScopeFile   Scope = "File"
	ScopeFolder Scope = "Folder"
)

// Interface is the contract every metric provider implements.
type Interface interface {
	Name() metric.Name
	Kind() metric.Kind
	Scope() Scope
	Description() string
	Dependencies() []metric.Name
	DefaultPalette() palette.PaletteName
	Load(root *model.Directory) error
}
