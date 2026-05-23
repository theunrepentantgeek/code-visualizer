package golang

import (
	"go/ast"
	"go/token"
	"os"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/rotisserie/eris"
)

type fileStats struct {
	types               int64
	publicTypes         int64
	privateTypes        int64
	interfaces          int64
	publicInterfaces    int64
	privateInterfaces   int64
	structs             int64
	publicStructs       int64
	privateStructs      int64
	functions           int64
	publicFunctions     int64
	privateFunctions    int64
	methods             int64
	publicMethods       int64
	privateMethods      int64
	constants           int64
	publicConstants     int64
	privateConstants    int64
	variables           int64
	publicVariables     int64
	privateVariables    int64
	imports             int64
	stdlibImports       int64
	externalImports     int64
	internalImports     int64
	declarations        int64
	publicDeclarations  int64
	privateDeclarations int64
	cyclomaticSum       int64
	cyclomaticMax       int64
	cyclomaticMean      float64
	funcLengthSum       int64
	funcLengthMax       int64
	funcLengthMean      float64
	commentRatio        float64
}

//nolint:nilaway,nolintlint // caller guarantees non-nil after successful parse
func countDeclarations(dstFile *dst.File, stats *fileStats) {
	for _, decl := range dstFile.Decls {
		switch d := decl.(type) {
		case *dst.GenDecl:
			countGenDecl(d, stats)
		case *dst.FuncDecl:
			countFuncDecl(d, stats)
		default:
			// ignore unknown declaration types
		}
	}
}

func countGenDecl(d *dst.GenDecl, stats *fileStats) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *dst.TypeSpec:
			countTypeSpec(s, stats)
		case *dst.ValueSpec:
			countValueSpec(s, d.Tok, stats)
		default:
			// ignore import specs and other spec types
		}
	}
}

func countTypeSpec(s *dst.TypeSpec, stats *fileStats) {
	pub := isPublic(s.Name.Name)

	stats.types++
	if pub {
		stats.publicTypes++
	} else {
		stats.privateTypes++
	}

	switch s.Type.(type) {
	case *dst.InterfaceType:
		stats.interfaces++
		if pub {
			stats.publicInterfaces++
		} else {
			stats.privateInterfaces++
		}
	case *dst.StructType:
		stats.structs++
		if pub {
			stats.publicStructs++
		} else {
			stats.privateStructs++
		}
	default:
		// other type forms (maps, slices, funcs, etc.) are counted as plain types only
	}
}

func countValueSpec(s *dst.ValueSpec, tok token.Token, stats *fileStats) {
	for _, name := range s.Names {
		pub := isPublic(name.Name)

		switch tok {
		case token.CONST:
			stats.constants++
			if pub {
				stats.publicConstants++
			} else {
				stats.privateConstants++
			}
		case token.VAR:
			stats.variables++
			if pub {
				stats.publicVariables++
			} else {
				stats.privateVariables++
			}
		default:
			// only CONST and VAR value specs contribute to counts
		}
	}
}

func countFuncDecl(d *dst.FuncDecl, stats *fileStats) {
	pub := isPublic(d.Name.Name)

	if d.Recv != nil && len(d.Recv.List) > 0 {
		stats.methods++
		if pub {
			stats.publicMethods++
		} else {
			stats.privateMethods++
		}
	} else {
		stats.functions++
		if pub {
			stats.publicFunctions++
		} else {
			stats.privateFunctions++
		}
	}
}

func (s *fileStats) computeAggregates() {
	s.declarations = s.types + s.functions + s.methods + s.constants + s.variables
	s.publicDeclarations = s.publicTypes + s.publicFunctions + s.publicMethods +
		s.publicConstants + s.publicVariables
	s.privateDeclarations = s.privateTypes + s.privateFunctions + s.privateMethods +
		s.privateConstants + s.privateVariables
}

func isPublic(name string) bool {
	return token.IsExported(name)
}

// analyzeFile parses a .go file with dst and extracts all metrics in a single
// pass. The modulePath is used for internal import classification.
func analyzeFile(path string, modulePath string) (*fileStats, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, eris.Wrapf(err, "reading Go file %s", path)
	}

	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	dstFile, err := dec.ParseFile(path, src, 0)
	if err != nil {
		return nil, eris.Wrapf(err, "parsing Go file %s", path)
	}

	stats := &fileStats{}

	countDeclarations(dstFile, stats)
	computeFunctionMetrics(dstFile, dec, fset, stats)
	classifyImports(dstFile, modulePath, stats)

	astFile, ok := dec.Ast.Nodes[dstFile].(*ast.File)
	if ok {
		stats.commentRatio = computeCommentRatio(src, astFile.Comments, fset)
	}

	stats.computeAggregates()

	return stats, nil
}

// computeFunctionMetrics computes cyclomatic complexity and function length
// for all functions/methods, then aggregates to sum/max/mean.
func computeFunctionMetrics(
	dstFile *dst.File,
	dec *decorator.Decorator,
	fset *token.FileSet,
	stats *fileStats,
) {
	complexities := make([]int64, 0, len(dstFile.Decls))
	lengths := make([]int64, 0, len(dstFile.Decls))

	for _, decl := range dstFile.Decls {
		funcDecl, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}

		cc := cyclomaticComplexity(funcDecl.Body)
		complexities = append(complexities, cc)

		astNode := dec.Ast.Nodes[funcDecl]
		if astNode != nil {
			startLine := fset.Position(astNode.Pos()).Line
			endLine := fset.Position(astNode.End()).Line
			length := int64(endLine - startLine + 1)
			lengths = append(lengths, length)
		}
	}

	aggregateInt64s(complexities, &stats.cyclomaticSum, &stats.cyclomaticMax, &stats.cyclomaticMean)
	aggregateInt64s(lengths, &stats.funcLengthSum, &stats.funcLengthMax, &stats.funcLengthMean)
}

func aggregateInt64s(values []int64, sum *int64, maxVal *int64, mean *float64) {
	if len(values) == 0 {
		return
	}

	for _, v := range values {
		*sum += v

		if v > *maxVal {
			*maxVal = v
		}
	}

	*mean = float64(*sum) / float64(len(values))
}
