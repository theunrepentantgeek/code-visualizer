package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
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
	Declarations         metric.Name = "declarations"
)

const (
	filterPublic   metric.FilterName = "public"
	filterPrivate  metric.FilterName = "private"
	filterStdlib   metric.FilterName = "stdlib"
	filterExternal metric.FilterName = "external"
	filterInternal metric.FilterName = "internal"
)

// GoProvider is the provider descriptor for Go metrics.
var GoProvider = provider.ProviderDescriptor{
	Name: "go",
	Filters: map[metric.FilterName]string{
		filterPublic:   "Exported declarations only",
		filterPrivate:  "Unexported declarations only",
		filterStdlib:   "Standard library imports only",
		filterExternal: "External (third-party) imports only",
		filterInternal: "Internal (same module) imports only",
	},
}

var (
	goDeclCountAggs   = []metric.AggregationName{metric.AggCount, metric.AggSum}
	goNumericAggs     = []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean}
	goSummaryAggs     = []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean}
	goVisibilityNames = []metric.FilterName{filterPublic, filterPrivate}
	goImportFilters   = []metric.FilterName{filterStdlib, filterExternal, filterInternal}

	// goDeclarationFilter evaluates visibility filters against a Declaration node.
	goDeclarationFilter = func(filter metric.FilterName, node any) bool {
		d, ok := node.(*model.Declaration)
		if !ok {
			return false
		}

		return d.MatchesFilter(filter)
	}

	goBaseMetrics = []provider.BaseMetricDescriptor{
		{
			Name:           Types,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of type declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Interfaces,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of interface type declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Structs,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of struct type declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Functions,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of function declarations (no receiver).",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Methods,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of method declarations (with receiver).",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Constants,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of constant declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Variables,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of variable declarations.",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           Declarations,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Count of all declarations (types, functions, methods, constants, variables).",
			Filters:        goVisibilityNames,
			Aggregations:   goDeclCountAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
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
			Filters:        goVisibilityNames,
			Aggregations:   goNumericAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
		},
		{
			Name:           FunctionLength,
			Kind:           metric.Quantity,
			Level:          metric.LevelDeclaration,
			Description:    "Function length in lines.",
			Filters:        goVisibilityNames,
			Aggregations:   goNumericAggs,
			DefaultPalette: palette.Neutral,
			FilterFunc:     goDeclarationFilter,
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
