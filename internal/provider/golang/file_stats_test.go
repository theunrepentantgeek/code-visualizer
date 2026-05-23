package golang

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/dave/dst/decorator"
)

func TestCountDeclarations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

type Exported struct{}
type unexported interface{}

func PublicFunc() {}
func privateFunc() {}

func (e *Exported) PublicMethod() {}
func (e *Exported) privateMethod() {}

const (
	PublicConst  = 1
	privateConst = 2
)

var (
	PublicVar  int
	privateVar string
)
`

	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	countDeclarations(dstFile, stats)

	g.Expect(stats.types.total).To(Equal(int64(2)))
	g.Expect(stats.types.public).To(Equal(int64(1)))
	g.Expect(stats.types.private).To(Equal(int64(1)))
	g.Expect(stats.structs.total).To(Equal(int64(1)))
	g.Expect(stats.structs.public).To(Equal(int64(1)))
	g.Expect(stats.structs.private).To(Equal(int64(0)))
	g.Expect(stats.interfaces.total).To(Equal(int64(1)))
	g.Expect(stats.interfaces.public).To(Equal(int64(0)))
	g.Expect(stats.interfaces.private).To(Equal(int64(1)))
	g.Expect(stats.functions.total).To(Equal(int64(2)))
	g.Expect(stats.functions.public).To(Equal(int64(1)))
	g.Expect(stats.functions.private).To(Equal(int64(1)))
	g.Expect(stats.methods.total).To(Equal(int64(2)))
	g.Expect(stats.methods.public).To(Equal(int64(1)))
	g.Expect(stats.methods.private).To(Equal(int64(1)))
	g.Expect(stats.constants.total).To(Equal(int64(2)))
	g.Expect(stats.constants.public).To(Equal(int64(1)))
	g.Expect(stats.constants.private).To(Equal(int64(1)))
	g.Expect(stats.variables.total).To(Equal(int64(2)))
	g.Expect(stats.variables.public).To(Equal(int64(1)))
	g.Expect(stats.variables.private).To(Equal(int64(1)))
}

func TestCountDeclarationsMultipleNames(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

var X, Y, z int
const A, b = 1, 2
`

	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	countDeclarations(dstFile, stats)

	g.Expect(stats.variables.total).To(Equal(int64(3)))
	g.Expect(stats.variables.public).To(Equal(int64(2)))
	g.Expect(stats.variables.private).To(Equal(int64(1)))
	g.Expect(stats.constants.total).To(Equal(int64(2)))
	g.Expect(stats.constants.public).To(Equal(int64(1)))
	g.Expect(stats.constants.private).To(Equal(int64(1)))
}

func TestComputeAggregates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stats := &fileStats{
		types:     visibilityCount{total: 2, public: 1, private: 1},
		functions: visibilityCount{total: 3, public: 2, private: 1},
		methods:   visibilityCount{total: 1, public: 1, private: 0},
		constants: visibilityCount{total: 4, public: 2, private: 2},
		variables: visibilityCount{total: 2, public: 1, private: 1},
	}

	stats.computeAggregates()

	g.Expect(stats.declarations.total).To(Equal(int64(12)))
	g.Expect(stats.declarations.public).To(Equal(int64(7)))
	g.Expect(stats.declarations.private).To(Equal(int64(5)))
}

func TestIsPublic(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(isPublic("Exported")).To(BeTrue())
	g.Expect(isPublic("unexported")).To(BeFalse())
	g.Expect(isPublic("X")).To(BeTrue())
	g.Expect(isPublic("x")).To(BeFalse())
	g.Expect(isPublic("_private")).To(BeFalse())
}

func TestAnalyzeFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/example\n\ngo 1.26\n"), 0o600)

	src := `package example

import (
	"fmt"

	"github.com/other/pkg"

	"github.com/test/example/internal/foo"
)

// Greet returns a greeting.
type Greeter interface {
	Greet() string
}

type greeterImpl struct {
	name string
}

func (g *greeterImpl) Greet() string {
	return fmt.Sprintf("Hello, %s", g.name)
}

func Public() {
	if true {
		for i := 0; i < 10; i++ {
			fmt.Println(pkg.Do(), foo.Bar())
		}
	}
}

const ExportedConst = 1

var unexportedVar int
`
	goFile := filepath.Join(dir, "example.go")
	_ = os.WriteFile(goFile, []byte(src), 0o600)

	stats, err := analyzeFile(goFile, "github.com/test/example")
	g.Expect(err).NotTo(HaveOccurred())

	if stats == nil {
		t.Fatal("analyzeFile returned nil stats without error")
	}

	// Types
	g.Expect(stats.types.total).To(Equal(int64(2)))
	g.Expect(stats.types.public).To(Equal(int64(1)))
	g.Expect(stats.interfaces.total).To(Equal(int64(1)))
	g.Expect(stats.structs.total).To(Equal(int64(1)))

	// Functions and methods
	g.Expect(stats.functions.total).To(Equal(int64(1)))
	g.Expect(stats.methods.total).To(Equal(int64(1)))

	// Constants and variables
	g.Expect(stats.constants.total).To(Equal(int64(1)))
	g.Expect(stats.variables.total).To(Equal(int64(1)))

	// Imports
	g.Expect(stats.imports).To(Equal(int64(3)))
	g.Expect(stats.stdlibImports).To(Equal(int64(1)))
	g.Expect(stats.externalImports).To(Equal(int64(1)))
	g.Expect(stats.internalImports).To(Equal(int64(1)))

	// Cyclomatic: Greet()=1, Public()=3 (if + for)
	g.Expect(stats.cyclomatic.sum).To(Equal(int64(4)))
	g.Expect(stats.cyclomatic.max).To(Equal(int64(3)))
	g.Expect(stats.cyclomatic.mean).To(BeNumerically("~", 2.0, 0.01))

	// Function length > 0
	g.Expect(stats.funcLength.sum).To(BeNumerically(">", 0))
	g.Expect(stats.funcLength.max).To(BeNumerically(">", 0))
	g.Expect(stats.funcLength.mean).To(BeNumerically(">", 0))

	// Comment ratio: 1 comment line / several code lines
	g.Expect(stats.commentRatio).To(BeNumerically(">", 0))
	g.Expect(stats.commentRatio).To(BeNumerically("<", 1.0))

	// Aggregates
	g.Expect(stats.declarations.total).To(Equal(int64(6)))
	g.Expect(stats.declarations.public).To(Equal(int64(4)))
	g.Expect(stats.declarations.private).To(Equal(int64(2)))
}

func TestAnalyzeFileNotGo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "bad.go"), []byte("not go code at all"), 0o600)

	_, err := analyzeFile(filepath.Join(dir, "bad.go"), "")
	g.Expect(err).To(HaveOccurred())
}
