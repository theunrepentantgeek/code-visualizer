package golang

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/dave/dst/decorator"
)

func TestClassifyImports(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

import (
	"fmt"
	"net/http"
	"encoding/json"

	"github.com/example/external"
	"github.com/other/pkg"

	"github.com/myorg/mymod/internal/foo"
	"github.com/myorg/mymod/pkg/bar"
)
`
	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	classifyImports(dstFile, "github.com/myorg/mymod", stats)

	g.Expect(stats.imports).To(Equal(int64(7)))
	g.Expect(stats.stdlibImports).To(Equal(int64(3)))
	g.Expect(stats.externalImports).To(Equal(int64(2)))
	g.Expect(stats.internalImports).To(Equal(int64(2)))
}

func TestClassifyImportsNoModule(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

import (
	"fmt"
	"github.com/other/pkg"
)
`
	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	classifyImports(dstFile, "", stats)

	g.Expect(stats.imports).To(Equal(int64(2)))
	g.Expect(stats.stdlibImports).To(Equal(int64(1)))
	g.Expect(stats.externalImports).To(Equal(int64(1)))
	g.Expect(stats.internalImports).To(Equal(int64(0)))
}

func TestIsStdlib(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(isStdlib("fmt")).To(BeTrue())
	g.Expect(isStdlib("net/http")).To(BeTrue())
	g.Expect(isStdlib("encoding/json")).To(BeTrue())
	g.Expect(isStdlib("github.com/foo/bar")).To(BeFalse())
	g.Expect(isStdlib("golang.org/x/sync")).To(BeFalse())
}

func TestFindModulePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "deep")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/mod\n\ngo 1.26\n"), 0o600)

	mc := newModuleCache()

	path := mc.findModulePath(sub)
	g.Expect(path).To(Equal("github.com/test/mod"))

	// Cached: same result for parent dir
	path2 := mc.findModulePath(filepath.Join(dir, "sub"))
	g.Expect(path2).To(Equal("github.com/test/mod"))
}

func TestFindModulePathNoGoMod(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	mc := newModuleCache()

	path := mc.findModulePath(dir)
	g.Expect(path).To(Equal(""))
}
