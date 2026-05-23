package golang

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

//nolint:paralleltest // deliberately sequential: ResetCacheForTesting mutates package globals
func TestGoProviderIntegration(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/proj\n\ngo 1.26\n"), 0o600)

	src := `package proj

import (
	"fmt"
	"github.com/test/proj/internal/sub"
)

// Hello is exported.
type Hello struct{}

func (h *Hello) Greet() string {
	return fmt.Sprint(sub.Name())
}

func helper() {}
`
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme\n"), 0o600)

	f1 := &model.File{Path: filepath.Join(dir, "main.go"), Name: "main.go", Extension: "go"}
	f2 := &model.File{Path: filepath.Join(dir, "readme.md"), Name: "readme.md", Extension: "md"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	ResetCacheForTesting()

	// Run type-count provider
	p := newProvider(TypeCount)
	g.Expect(p.Name()).To(Equal(TypeCount))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.DefaultPalette()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := f1.Quantity(TypeCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(1)))

	// Non-.go file should have no Go metrics
	_, ok = f2.Quantity(TypeCount)
	g.Expect(ok).To(BeFalse())
}

//nolint:paralleltest // deliberately sequential: ResetCacheForTesting mutates package globals
func TestGoProviderCacheConsistency(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/proj\n\ngo 1.26\n"), 0o600)

	src := `package proj

type Foo struct{}
type bar interface{}

func Public() {}
func private() {}
`
	_ = os.WriteFile(filepath.Join(dir, "code.go"), []byte(src), 0o600)

	f := &model.File{Path: filepath.Join(dir, "code.go"), Name: "code.go", Extension: "go"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{f},
	}

	ResetCacheForTesting()

	// Run multiple providers on the same file
	typeP := newProvider(TypeCount)
	err := typeP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	funcP := newProvider(FunctionCount)
	err = funcP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	structP := newProvider(StructCount)
	err = structP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	interfaceP := newProvider(InterfaceCount)
	err = interfaceP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	commentP := newProvider(CommentRatio)
	err = commentP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// All metrics should be set from the same cached parse
	typeVal, ok := f.Quantity(TypeCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(typeVal).To(Equal(int64(2)))

	funcVal, ok := f.Quantity(FunctionCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(funcVal).To(Equal(int64(2)))

	structVal, ok := f.Quantity(StructCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(structVal).To(Equal(int64(1)))

	interfaceVal, ok := f.Quantity(InterfaceCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(interfaceVal).To(Equal(int64(1)))

	ratioVal, ok := f.Measure(CommentRatio)
	g.Expect(ok).To(BeTrue())
	g.Expect(ratioVal).To(BeNumerically(">=", 0))
}

func TestGoProviderAllMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for name := range providerDefs {
		p := newProvider(name)
		g.Expect(p.Name()).To(Equal(name), "name mismatch for "+string(name))
		g.Expect(p.Description()).NotTo(BeEmpty(), "empty description for "+string(name))
		g.Expect(p.DefaultPalette()).NotTo(BeEmpty(), "empty palette for "+string(name))
		g.Expect(p.Dependencies()).To(BeNil(), "unexpected dependencies for "+string(name))
	}
}
