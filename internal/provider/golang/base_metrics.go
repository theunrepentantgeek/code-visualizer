package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

const (
	Types                metric.Name = "types"
	Interfaces           metric.Name = "interfaces"
	Structs              metric.Name = "structs"
	Functions            metric.Name = "functions"
	Methods              metric.Name = "methods"
	Constants            metric.Name = "constants"
	Variables            metric.Name = "variables"
	Imports              metric.Name = "imports"
	CyclomaticComplexity metric.Name = "cyclomatic-complexity"
	FunctionLength       metric.Name = "function-length"
)

// GoProvider is the provider descriptor for Go metrics.
var GoProvider = provider.ProviderDescriptor{
	Name: "go",
	Filters: map[metric.FilterName]string{
		"public":   "Exported declarations only",
		"private":  "Unexported declarations only",
		"stdlib":   "Standard library imports only",
		"external": "External (third-party) imports only",
		"internal": "Internal (same module) imports only",
	},
}

var (
	goDeclCountAggs   = []metric.AggregationName{metric.AggCount, metric.AggSum}
	goNumericAggs     = []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean}
	goSummaryAggs     = []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean}
	goVisibilityNames = []metric.FilterName{"public", "private"}
	goImportFilters   = []metric.FilterName{"stdlib", "external", "internal"}
	goBaseMetrics     = []provider.BaseMetricDescriptor{
		{
			Name:           Types,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of type declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Interfaces,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of interface type declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Structs,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of struct type declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Functions,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of function declarations (no receiver).",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Methods,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of method declarations (with receiver).",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Constants,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of constant declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Variables,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of variable declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           Imports,
			Kind:           metric.Quantity,
			Level:          metric.LevelFile,
			Description:    "Total import paths in Go files.",
			Filters:        goImportFilters,
			Aggregations:   goNumericAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           CyclomaticComplexity,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Cyclomatic complexity per function.",
			Aggregations:   goNumericAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           FunctionLength,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Function length in lines.",
			Aggregations:   goNumericAggs,
			DefaultPalette: palette.Neutral,
		},
		{
			Name:           CommentRatio,
			Kind:           metric.Measure,
			Level:          metric.LevelFile,
			Description:    "Ratio of comment lines to code lines in Go files.",
			Aggregations:   goSummaryAggs,
			DefaultPalette: palette.Neutral,
		},
	}
)

// RegisterBase adds Go base metric descriptors to the global base registry.
func RegisterBase() {
	for _, desc := range goBaseMetrics {
		provider.RegisterBaseWithProvider(desc, GoProvider)
	}
}
