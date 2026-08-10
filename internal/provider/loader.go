package provider

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// BaseMetricLoader describes a unit of metric loading work.
// A single loader may populate multiple base metrics in one pass.
type BaseMetricLoader struct {
	// Metrics lists the base metric names this loader populates.
	Metrics []metric.Name
	// Dependencies lists base metrics callers must schedule before this loader runs.
	Dependencies []metric.Name
	// Load populates the requested metric values in the directory tree.
	Load LoadFunc
	// Reporter optionally receives per-file progress callbacks during loading.
	Reporter FileProgressReporter
}

// LoadFunc loads requested metrics into root. requested contains only metrics
// declared by the loader, in the order they appear in its Metrics field.
type LoadFunc func(root *model.Directory, requested []metric.Name) error
