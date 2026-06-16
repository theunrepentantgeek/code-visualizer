// Package golang provides metric providers for Go-specific code metrics.
package golang

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

// IsGoMetric reports whether name is a Go-specific metric.
func IsGoMetric(name metric.Name) bool {
	_, ok := allMetrics[name]

	return ok
}

var allMetrics = map[metric.Name]struct{}{
	Types:                {},
	Interfaces:           {},
	Structs:              {},
	Functions:            {},
	Methods:              {},
	Constants:            {},
	Variables:            {},
	Imports:              {},
	CyclomaticComplexity: {},
	FunctionLength:       {},
	Declarations:         {},
	CommentRatio:         {},
}
