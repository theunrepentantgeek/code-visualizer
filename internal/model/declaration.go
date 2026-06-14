package model

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

// Declaration kind constants.
const (
	DeclKindType      = "type"
	DeclKindStruct    = "struct"
	DeclKindInterface = "interface"
	DeclKindFunction  = "function"
	DeclKindMethod    = "method"
	DeclKindConstant  = "constant"
	DeclKindVariable  = "variable"
)

// Declaration represents a single named declaration within a source file.
type Declaration struct {
	MetricContainer
	Name       string // e.g., "HandleRequest", "UserService"
	Kind       string // e.g., "function", "method", "interface", "struct", "constant", "variable"
	Visibility string // "public" or "private"
}

// MatchesFilter reports whether this declaration passes the named filter.
func (d *Declaration) MatchesFilter(filter metric.FilterName) bool {
	switch filter {
	case "public":
		return d.Visibility == "public"
	case "private":
		return d.Visibility == "private"
	default:
		return false
	}
}
