package golang

import (
	"go/token"
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
