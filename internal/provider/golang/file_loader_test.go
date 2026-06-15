package golang

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

//nolint:paralleltest // mutates package globals via ResetCacheForTesting
func TestLoadFileMetrics_PopulatesGoFileMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/proj\n\ngo 1.26\n"), 0o600)

	src := `package proj

import (
	"fmt"
	"github.com/example/lib"
	"github.com/test/proj/internal/sub"
)

// Hello documents the exported function.
func Hello() string {
	// Keep one inline comment so the ratio is non-zero.
	return fmt.Sprint(sub.Name())
}
`
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme\n"), 0o600)

	goFile := &model.File{Path: filepath.Join(dir, "main.go"), Name: "main.go", Extension: "go"}
	otherFile := &model.File{Path: filepath.Join(dir, "README.md"), Name: "README.md", Extension: "md"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{goFile, otherFile},
	}

	ResetCacheForTesting()

	err := loadFileMetrics(root)
	g.Expect(err).NotTo(HaveOccurred())

	imports, ok := goFile.Quantity(Imports)
	g.Expect(ok).To(BeTrue())
	g.Expect(imports).To(Equal(int64(3)))

	stdlibImports, ok := goFile.Quantity(metric.Name("stdlib.imports"))
	g.Expect(ok).To(BeTrue())
	g.Expect(stdlibImports).To(Equal(int64(1)))

	externalImports, ok := goFile.Quantity(metric.Name("external.imports"))
	g.Expect(ok).To(BeTrue())
	g.Expect(externalImports).To(Equal(int64(1)))

	internalImports, ok := goFile.Quantity(metric.Name("internal.imports"))
	g.Expect(ok).To(BeTrue())
	g.Expect(internalImports).To(Equal(int64(1)))

	commentRatio, ok := goFile.Measure(CommentRatio)
	g.Expect(ok).To(BeTrue())
	g.Expect(commentRatio).To(BeNumerically(">", 0))

	_, ok = otherFile.Quantity(Imports)
	g.Expect(ok).To(BeFalse())
}

//nolint:paralleltest // mutates package globals via ResetCacheForTesting
func TestLoadFileMetrics_SetsZeroCommentRatio(t *testing.T) {
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/proj\n\ngo 1.26\n"), 0o600)

	src := `package proj

func Hello() string {
	return "hi"
}
`
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600)

	goFile := &model.File{Path: filepath.Join(dir, "main.go"), Name: "main.go", Extension: "go"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{goFile},
	}

	ResetCacheForTesting()

	err := loadFileMetrics(root)
	g.Expect(err).NotTo(HaveOccurred())

	commentRatio, ok := goFile.Measure(CommentRatio)
	g.Expect(ok).To(BeTrue())
	g.Expect(commentRatio).To(Equal(float64(0)))
}

//nolint:paralleltest // mutates global provider and base registries
func TestRegister_RegistersGoFileMetricsLoader(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetRegistryForTesting()
	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetRegistryForTesting)
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	Register()

	loaders := provider.LoadersFor([]metric.Name{stdlibImportsMetric})
	g.Expect(loaders).To(HaveLen(1))
	g.Expect(loaders[0].Metrics).To(ConsistOf(
		Imports,
		CommentRatio,
		stdlibImportsMetric,
		externalImportsMetric,
		internalImportsMetric,
	))
	g.Expect(loaders[0].Load).ToNot(BeNil())
}
