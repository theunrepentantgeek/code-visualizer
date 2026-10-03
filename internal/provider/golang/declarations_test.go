package golang

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func TestPopulateDeclarations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path, err := filepath.Abs(filepath.Join("testdata", "sample.go"))
	g.Expect(err).NotTo(HaveOccurred())

	f := &model.File{
		Path:       path,
		SourcePath: "sample.go",
		Name:       "sample.go",
		Extension:  "go",
		Source:     os.DirFS("testdata"),
	}

	PopulateDeclarations(f)

	g.Expect(f.Declarations).To(HaveLen(11))

	declarations := declarationMap(f.Declarations)

	expectDeclaration(t, declarations, "SampleInterface", "interface", "public")
	expectDeclaration(t, declarations, "SampleStruct", "struct", "public")
	expectDeclaration(t, declarations, "SampleType", "type", "public")
	expectDeclaration(t, declarations, "PublicConst", "constant", "public")
	expectDeclaration(t, declarations, "privateConst", "constant", "private")
	expectDeclaration(t, declarations, "PublicVar", "variable", "public")
	expectDeclaration(t, declarations, "privateVar", "variable", "private")
	expectDeclaration(t, declarations, "PublicFunc", "function", "public")
	expectDeclaration(t, declarations, "privateFunc", "function", "private")
	expectDeclaration(t, declarations, "PublicMethod", "method", "public")
	expectDeclaration(t, declarations, "privateMethod", "method", "private")

	expectFunctionMetrics(t, declarations["PublicFunc"], 2)
	expectFunctionMetrics(t, declarations["privateFunc"], 1)
	expectFunctionMetrics(t, declarations["PublicMethod"], 1)
	expectFunctionMetrics(t, declarations["privateMethod"], 3)
}

func TestPopulateDeclarationsAppendsToExistingDeclarations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path, err := filepath.Abs(filepath.Join("testdata", "sample.go"))
	g.Expect(err).NotTo(HaveOccurred())

	existing := &model.Declaration{
		Name:       "ExistingDecl",
		Kind:       "type",
		Visibility: "public",
	}

	f := &model.File{
		Path:         path,
		SourcePath:   "sample.go",
		Name:         "sample.go",
		Extension:    "go",
		Source:       os.DirFS("testdata"),
		Declarations: []*model.Declaration{existing},
	}

	PopulateDeclarations(f)

	g.Expect(f.Declarations).To(HaveLen(12))
	g.Expect(f.Declarations[0]).To(BeIdenticalTo(existing))
	g.Expect(declarationMap(f.Declarations)).To(HaveKey("PublicFunc"))
}

func TestPopulateDeclarationsSkipsNonGoFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{
		Path:      filepath.Join("testdata", "sample.go"),
		Name:      "sample.txt",
		Extension: "txt",
	}

	PopulateDeclarations(f)

	g.Expect(f.Declarations).To(BeEmpty())
}

func TestAnalyzeModelDeclarationsRejectsMissingSourceEvenWhenPathExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "readable.go")
	g.Expect(os.WriteFile(filePath, []byte("package readable\n"), 0o600)).To(Succeed())

	_, err := analyzeModelDeclarations(&model.File{Path: filePath, Extension: "go"})

	g.Expect(err).To(MatchError(ContainSubstring("file has no content source")))
}

func declarationMap(declarations []*model.Declaration) map[string]*model.Declaration {
	result := make(map[string]*model.Declaration, len(declarations))

	for _, declaration := range declarations {
		result[declaration.Name] = declaration
	}

	return result
}

func expectDeclaration(
	t *testing.T,
	declarations map[string]*model.Declaration,
	name string,
	kind string,
	visibility string,
) {
	t.Helper()
	g := NewGomegaWithT(t)

	declaration, ok := declarations[name]
	g.Expect(ok).To(BeTrue(), "expected declaration %q", name)
	g.Expect(declaration).NotTo(BeNil())

	if declaration == nil {
		return
	}

	g.Expect(declaration.Kind).To(Equal(kind))
	g.Expect(declaration.Visibility).To(Equal(visibility))
}

func expectFunctionMetrics(
	t *testing.T,
	declaration *model.Declaration,
	wantComplexity int64,
) {
	t.Helper()
	g := NewGomegaWithT(t)

	g.Expect(declaration).NotTo(BeNil())

	if declaration == nil {
		return
	}

	complexity, ok := declaration.Quantity(CyclomaticComplexity)
	g.Expect(ok).To(BeTrue())
	g.Expect(complexity).To(Equal(wantComplexity))

	length, ok := declaration.Quantity(FunctionLength)
	g.Expect(ok).To(BeTrue())
	g.Expect(length).To(BeNumerically(">", 0))
}
