// Package golang provides metric providers for Go-specific code metrics.
package golang

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

const (
	TypeCount               metric.Name = "type-count"
	PublicTypeCount         metric.Name = "public-type-count"
	PrivateTypeCount        metric.Name = "private-type-count"
	InterfaceCount          metric.Name = "interface-count"
	PublicInterfaceCount    metric.Name = "public-interface-count"
	PrivateInterfaceCount   metric.Name = "private-interface-count"
	StructCount             metric.Name = "struct-count"
	PublicStructCount       metric.Name = "public-struct-count"
	PrivateStructCount      metric.Name = "private-struct-count"
	FunctionCount           metric.Name = "function-count"
	PublicFunctionCount     metric.Name = "public-function-count"
	PrivateFunctionCount    metric.Name = "private-function-count"
	MethodCount             metric.Name = "method-count"
	PublicMethodCount       metric.Name = "public-method-count"
	PrivateMethodCount      metric.Name = "private-method-count"
	ConstantCount           metric.Name = "constant-count"
	PublicConstantCount     metric.Name = "public-constant-count"
	PrivateConstantCount    metric.Name = "private-constant-count"
	VariableCount           metric.Name = "variable-count"
	PublicVariableCount     metric.Name = "public-variable-count"
	PrivateVariableCount    metric.Name = "private-variable-count"
	ImportCount             metric.Name = "import-count"
	StdlibImportCount       metric.Name = "stdlib-import-count"
	ExternalImportCount     metric.Name = "external-import-count"
	InternalImportCount     metric.Name = "internal-import-count"
	DeclarationCount        metric.Name = "declaration-count"
	PublicDeclarationCount  metric.Name = "public-declaration-count"
	PrivateDeclarationCount metric.Name = "private-declaration-count"
	CyclomaticComplexitySum  metric.Name = "cyclomatic-complexity-sum"
	CyclomaticComplexityMax  metric.Name = "cyclomatic-complexity-max"
	CyclomaticComplexityMean metric.Name = "cyclomatic-complexity-mean"
	FunctionLengthSum       metric.Name = "function-length-sum"
	FunctionLengthMax       metric.Name = "function-length-max"
	FunctionLengthMean      metric.Name = "function-length-mean"
	CommentRatio            metric.Name = "comment-ratio"
)

// IsGoMetric reports whether name is a Go-specific metric.
func IsGoMetric(name metric.Name) bool {
	_, ok := allMetrics[name]
	return ok
}

var allMetrics = map[metric.Name]struct{}{
	TypeCount:                {},
	PublicTypeCount:          {},
	PrivateTypeCount:         {},
	InterfaceCount:           {},
	PublicInterfaceCount:     {},
	PrivateInterfaceCount:    {},
	StructCount:              {},
	PublicStructCount:        {},
	PrivateStructCount:       {},
	FunctionCount:            {},
	PublicFunctionCount:      {},
	PrivateFunctionCount:     {},
	MethodCount:              {},
	PublicMethodCount:        {},
	PrivateMethodCount:       {},
	ConstantCount:            {},
	PublicConstantCount:      {},
	PrivateConstantCount:     {},
	VariableCount:            {},
	PublicVariableCount:      {},
	PrivateVariableCount:     {},
	ImportCount:              {},
	StdlibImportCount:        {},
	ExternalImportCount:      {},
	InternalImportCount:      {},
	DeclarationCount:         {},
	PublicDeclarationCount:   {},
	PrivateDeclarationCount:  {},
	CyclomaticComplexitySum:  {},
	CyclomaticComplexityMax:  {},
	CyclomaticComplexityMean: {},
	FunctionLengthSum:        {},
	FunctionLengthMax:        {},
	FunctionLengthMean:       {},
	CommentRatio:             {},
}
