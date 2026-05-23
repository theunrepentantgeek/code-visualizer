package golang

import (
	"go/ast"
	"go/token"
	"os"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/rotisserie/eris"
)

// visibilityCount groups total/public/private counts for a declaration kind.
type visibilityCount struct {
	total   int64
	public  int64
	private int64
}

// aggregate groups sum/max/mean for a per-function metric.
type aggregate struct {
	sum  int64
	max  int64
	mean float64
}

type fileStats struct {
	types           visibilityCount
	interfaces      visibilityCount
	structs         visibilityCount
	functions       visibilityCount
	methods         visibilityCount
	constants       visibilityCount
	variables       visibilityCount
	declarations    visibilityCount
	imports         int64
	stdlibImports   int64
	externalImports int64
	internalImports int64
	cyclomatic      aggregate
	funcLength      aggregate
	commentRatio    float64
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
	stats.types.add(s.Name.Name)

	switch s.Type.(type) {
	case *dst.InterfaceType:
		stats.interfaces.add(s.Name.Name)
	case *dst.StructType:
		stats.structs.add(s.Name.Name)
	default:
		// other type forms (maps, slices, funcs, etc.) are counted as plain types only
	}
}

func countValueSpec(s *dst.ValueSpec, tok token.Token, stats *fileStats) {
	for _, name := range s.Names {
		switch tok {
		case token.CONST:
			stats.constants.add(name.Name)
		case token.VAR:
			stats.variables.add(name.Name)
		default:
			// only CONST and VAR value specs contribute to counts
		}
	}
}

func countFuncDecl(d *dst.FuncDecl, stats *fileStats) {
	if d.Recv != nil && len(d.Recv.List) > 0 {
		stats.methods.add(d.Name.Name)
	} else {
		stats.functions.add(d.Name.Name)
	}
}

func (s *fileStats) computeAggregates() {
	s.declarations.total = s.types.total + s.functions.total + s.methods.total +
		s.constants.total + s.variables.total
	s.declarations.public = s.types.public + s.functions.public + s.methods.public +
		s.constants.public + s.variables.public
	s.declarations.private = s.types.private + s.functions.private + s.methods.private +
		s.constants.private + s.variables.private
}

func (vc *visibilityCount) add(name string) {
	vc.total++

	if token.IsExported(name) {
		vc.public++
	} else {
		vc.private++
	}
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

	computeAggregate(complexities, &stats.cyclomatic)
	computeAggregate(lengths, &stats.funcLength)
}

func computeAggregate(values []int64, agg *aggregate) {
	if len(values) == 0 {
		return
	}

	for _, v := range values {
		agg.sum += v

		if v > agg.max {
			agg.max = v
		}
	}

	agg.mean = float64(agg.sum) / float64(len(values))
}
