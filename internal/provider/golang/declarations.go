package golang

import (
	"go/token"
	"log/slog"
	"os"
	"sync"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/singleflight"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

type declarationInfo struct {
	name               string
	kind               string
	visibility         string
	cyclomatic         int64
	functionLength     int64
	hasFunctionMetrics bool
}

type declarationAnalysis struct {
	declarations []declarationInfo
}

type declCache struct {
	mu       sync.Mutex
	group    singleflight.Group
	decls    map[string]*declarationAnalysis
	parseErr map[string]error
}

var globalDeclCache = &declCache{
	decls:    make(map[string]*declarationAnalysis),
	parseErr: make(map[string]error),
}

// PopulateDeclarations parses a Go file and attaches declaration nodes to it.
func PopulateDeclarations(f *model.File) {
	if f.Extension != "go" {
		return
	}

	analysis, err := getOrAnalyzeDeclarations(f.Path)
	if err != nil {
		slog.Warn("could not analyze Go declarations", "path", f.Path, "error", err)
		f.Declarations = nil

		return
	}

	f.Declarations = append(f.Declarations, buildDeclarations(analysis.declarations)...)
}

func getOrAnalyzeDeclarations(path string) (*declarationAnalysis, error) {
	globalDeclCache.mu.Lock()
	if declarations, ok := globalDeclCache.decls[path]; ok {
		globalDeclCache.mu.Unlock()

		return declarations, nil
	}

	if err, ok := globalDeclCache.parseErr[path]; ok {
		globalDeclCache.mu.Unlock()

		return nil, err
	}
	globalDeclCache.mu.Unlock()

	result, err, _ := globalDeclCache.group.Do(path, func() (any, error) {
		analysis, analyzeErr := analyzeDeclarations(path)
		if analyzeErr != nil {
			globalDeclCache.mu.Lock()
			globalDeclCache.parseErr[path] = analyzeErr
			globalDeclCache.mu.Unlock()

			return nil, analyzeErr
		}

		globalDeclCache.mu.Lock()
		globalDeclCache.decls[path] = analysis
		globalDeclCache.mu.Unlock()

		return analysis, nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "analyzing Go declarations")
	}

	analysis, ok := result.(*declarationAnalysis)
	if !ok {
		return nil, eris.New("unexpected type from declaration singleflight result")
	}

	return analysis, nil
}

func analyzeDeclarations(path string) (*declarationAnalysis, error) {
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

	return &declarationAnalysis{
		declarations: collectDeclarations(dstFile, dec, fset),
	}, nil
}

func collectDeclarations(
	dstFile *dst.File,
	dec *decorator.Decorator,
	fset *token.FileSet,
) []declarationInfo {
	declarations := make([]declarationInfo, 0, len(dstFile.Decls))

	for _, decl := range dstFile.Decls {
		switch typedDecl := decl.(type) {
		case *dst.GenDecl:
			declarations = appendGenDeclDeclarations(declarations, typedDecl)
		case *dst.FuncDecl:
			declarations = append(declarations, newFunctionDeclaration(typedDecl, dec, fset))
		default:
			// Ignore unsupported declaration nodes.
		}
	}

	return declarations
}

func appendGenDeclDeclarations(
	declarations []declarationInfo,
	decl *dst.GenDecl,
) []declarationInfo {
	for _, spec := range decl.Specs {
		switch typedSpec := spec.(type) {
		case *dst.TypeSpec:
			declarations = append(declarations, declarationInfo{
				name:       typedSpec.Name.Name,
				kind:       declarationKindForType(typedSpec.Type),
				visibility: visibilityForName(typedSpec.Name.Name),
			})
		case *dst.ValueSpec:
			kind := declarationKindForToken(decl.Tok)
			if kind == "" {
				continue
			}

			for _, name := range typedSpec.Names {
				declarations = append(declarations, declarationInfo{
					name:       name.Name,
					kind:       kind,
					visibility: visibilityForName(name.Name),
				})
			}
		default:
			// Ignore unsupported spec nodes.
		}
	}

	return declarations
}

func newFunctionDeclaration(
	decl *dst.FuncDecl,
	dec *decorator.Decorator,
	fset *token.FileSet,
) declarationInfo {
	kind := model.DeclKindFunction
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		kind = model.DeclKindMethod
	}

	return declarationInfo{
		name:               decl.Name.Name,
		kind:               kind,
		visibility:         visibilityForName(decl.Name.Name),
		cyclomatic:         cyclomaticComplexity(decl.Body),
		functionLength:     functionLength(decl, dec, fset),
		hasFunctionMetrics: true,
	}
}

func declarationKindForType(expr dst.Expr) string {
	switch expr.(type) {
	case *dst.InterfaceType:
		return model.DeclKindInterface
	case *dst.StructType:
		return model.DeclKindStruct
	default:
		return model.DeclKindType
	}
}

func declarationKindForToken(tok token.Token) string {
	switch tok {
	case token.CONST:
		return model.DeclKindConstant
	case token.VAR:
		return model.DeclKindVariable
	default:
		return ""
	}
}

func visibilityForName(name string) string {
	if token.IsExported(name) {
		return string(filterPublic)
	}

	return string(filterPrivate)
}

func functionLength(
	decl *dst.FuncDecl,
	dec *decorator.Decorator,
	fset *token.FileSet,
) int64 {
	astNode := dec.Ast.Nodes[decl]
	if astNode == nil {
		return 0
	}

	startLine := fset.Position(astNode.Pos()).Line
	endLine := fset.Position(astNode.End()).Line

	return int64(endLine - startLine + 1)
}

func buildDeclarations(declarations []declarationInfo) []*model.Declaration {
	result := make([]*model.Declaration, 0, len(declarations))

	for _, declaration := range declarations {
		modelDeclaration := &model.Declaration{
			Name:       declaration.name,
			Kind:       declaration.kind,
			Visibility: declaration.visibility,
		}

		if declaration.hasFunctionMetrics {
			modelDeclaration.SetQuantity(CyclomaticComplexity, declaration.cyclomatic)
			modelDeclaration.SetQuantity(FunctionLength, declaration.functionLength)
		}

		result = append(result, modelDeclaration)
	}

	return result
}

// ResetDeclCacheForTesting clears the declaration cache. Test use only.
func ResetDeclCacheForTesting() {
	globalDeclCache = &declCache{
		decls:    make(map[string]*declarationAnalysis),
		parseErr: make(map[string]error),
	}
}
