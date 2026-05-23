package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// providerDefs is the authoritative map of all Go metric providers.
// Adding a new Go metric requires only a new entry here.
var providerDefs = map[metric.Name]providerDef{
	TypeCount: {
		kind:           metric.Quantity,
		description:    "Total type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.types }),
	},
	PublicTypeCount: {
		kind:           metric.Quantity,
		description:    "Exported type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicTypes }),
	},
	PrivateTypeCount: {
		kind:           metric.Quantity,
		description:    "Unexported type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateTypes }),
	},
	InterfaceCount: {
		kind:           metric.Quantity,
		description:    "Interface type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.interfaces }),
	},
	PublicInterfaceCount: {
		kind:           metric.Quantity,
		description:    "Exported interface type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicInterfaces }),
	},
	PrivateInterfaceCount: {
		kind:           metric.Quantity,
		description:    "Unexported interface type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateInterfaces }),
	},
	StructCount: {
		kind:           metric.Quantity,
		description:    "Struct type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.structs }),
	},
	PublicStructCount: {
		kind:           metric.Quantity,
		description:    "Exported struct type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicStructs }),
	},
	PrivateStructCount: {
		kind:           metric.Quantity,
		description:    "Unexported struct type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateStructs }),
	},
	FunctionCount: {
		kind:           metric.Quantity,
		description:    "Function declarations (no receiver) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.functions }),
	},
	PublicFunctionCount: {
		kind:           metric.Quantity,
		description:    "Exported function declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicFunctions }),
	},
	PrivateFunctionCount: {
		kind:           metric.Quantity,
		description:    "Unexported function declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateFunctions }),
	},
	MethodCount: {
		kind:           metric.Quantity,
		description:    "Method declarations (with receiver) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.methods }),
	},
	PublicMethodCount: {
		kind:           metric.Quantity,
		description:    "Exported method declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicMethods }),
	},
	PrivateMethodCount: {
		kind:           metric.Quantity,
		description:    "Unexported method declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateMethods }),
	},
	ConstantCount: {
		kind:           metric.Quantity,
		description:    "Constant declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.constants }),
	},
	PublicConstantCount: {
		kind:           metric.Quantity,
		description:    "Exported constant declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicConstants }),
	},
	PrivateConstantCount: {
		kind:           metric.Quantity,
		description:    "Unexported constant declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateConstants }),
	},
	VariableCount: {
		kind:           metric.Quantity,
		description:    "Variable declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.variables }),
	},
	PublicVariableCount: {
		kind:           metric.Quantity,
		description:    "Exported variable declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicVariables }),
	},
	PrivateVariableCount: {
		kind:           metric.Quantity,
		description:    "Unexported variable declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateVariables }),
	},
	ImportCount: {
		kind:           metric.Quantity,
		description:    "Total import paths in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.imports }),
	},
	StdlibImportCount: {
		kind:           metric.Quantity,
		description:    "Standard library import count in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.stdlibImports }),
	},
	ExternalImportCount: {
		kind:           metric.Quantity,
		description:    "External (third-party) import count in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.externalImports }),
	},
	InternalImportCount: {
		kind:           metric.Quantity,
		description:    "Internal import count (same module) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.internalImports }),
	},
	DeclarationCount: {
		kind:           metric.Quantity,
		description:    "Total declarations (types + functions + methods + constants + variables) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.declarations }),
	},
	PublicDeclarationCount: {
		kind:           metric.Quantity,
		description:    "Total exported declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicDeclarations }),
	},
	PrivateDeclarationCount: {
		kind:           metric.Quantity,
		description:    "Total unexported declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateDeclarations }),
	},
	CyclomaticComplexitySum: {
		kind:           metric.Quantity,
		description:    "Sum of cyclomatic complexity across all functions in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.cyclomaticSum }),
	},
	CyclomaticComplexityMax: {
		kind:           metric.Quantity,
		description:    "Maximum cyclomatic complexity of any single function in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.cyclomaticMax }),
	},
	CyclomaticComplexityMean: {
		kind:           metric.Measure,
		description:    "Mean cyclomatic complexity per function in Go files.",
		defaultPalette: palette.Neutral,
		extract:        measureField(func(s *fileStats) float64 { return s.cyclomaticMean }),
	},
	FunctionLengthSum: {
		kind:           metric.Quantity,
		description:    "Sum of function lengths (lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.funcLengthSum }),
	},
	FunctionLengthMax: {
		kind:           metric.Quantity,
		description:    "Length of longest function (lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.funcLengthMax }),
	},
	FunctionLengthMean: {
		kind:           metric.Measure,
		description:    "Mean function length (lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        measureField(func(s *fileStats) float64 { return s.funcLengthMean }),
	},
	CommentRatio: {
		kind:           metric.Measure,
		description:    "Ratio of comment lines to code lines (ignoring blank lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        measureField(func(s *fileStats) float64 { return s.commentRatio }),
	},
}
