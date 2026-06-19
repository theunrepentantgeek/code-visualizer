package classification_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/classification"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{Name: "code-purpose"}
	p := classification.NewProvider(cfg)

	g.Expect(p.Name()).To(Equal(metric.Name("code-purpose")))
}

func TestProvider_Kind(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{Name: "code-purpose"}
	p := classification.NewProvider(cfg)

	g.Expect(p.Kind()).To(Equal(metric.Classification))
}

func TestProvider_Load_MatchesFirstRule(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{
		Name: "code-purpose",
		Rules: []config.SelectionMetricRule{
			{Category: "test", Filename: "*_test.go"},
			{Category: "source", Filename: "*"},
		},
	}
	p := classification.NewProvider(cfg)

	root := &model.Directory{Name: "root", Path: "/root"}
	testFile := &model.File{Path: "/root/foo_test.go"}
	srcFile := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{testFile, srcFile}

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	testCat, ok := testFile.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(testCat).To(Equal("test"))

	srcCat, ok := srcFile.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(srcCat).To(Equal("source"))
}

func TestProvider_Load_UnmatchedFileGetsNoValue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{
		Name: "code-purpose",
		Rules: []config.SelectionMetricRule{
			{Category: "test", Filename: "*_test.go"},
		},
	}
	p := classification.NewProvider(cfg)

	root := &model.Directory{Name: "root", Path: "/root"}
	file := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{file}

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	_, ok := file.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeFalse(), "file not matching any rule should have no metric value")
}

func TestProvider_Load_GeneratedFilePattern(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{
		Name: "code-source",
		Rules: []config.SelectionMetricRule{
			{Category: "gen", Filename: "*_gen.go"},
			{Category: "gen", Filename: "*_gen_test.go"},
			{Category: "authored", Filename: "*"},
		},
	}
	p := classification.NewProvider(cfg)

	root := &model.Directory{Name: "root", Path: "/root"}
	genFile := &model.File{Path: "/root/schema_gen.go"}
	genTestFile := &model.File{Path: "/root/schema_gen_test.go"}
	authoredFile := &model.File{Path: "/root/service.go"}
	root.Files = []*model.File{genFile, genTestFile, authoredFile}

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	cat, _ := genFile.Classification(metric.Name("code-source"))
	g.Expect(cat).To(Equal("gen"))

	cat, _ = genTestFile.Classification(metric.Name("code-source"))
	g.Expect(cat).To(Equal("gen"))

	cat, _ = authoredFile.Classification(metric.Name("code-source"))
	g.Expect(cat).To(Equal("authored"))
}

func TestProvider_Load_MatchesRelativePath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{
		Name: "file-role",
		Rules: []config.SelectionMetricRule{
			{Category: "testdata", Filename: "testdata/**"},
			{Category: "customized", Filename: "*customizations*/*.go"},
			{Category: "other", Filename: "*"},
		},
	}
	p := classification.NewProvider(cfg)

	root := &model.Directory{
		Name: "project",
		Path: "/project",
		Dirs: []*model.Directory{
			{
				Name: "testdata",
				Path: "/project/testdata",
				Files: []*model.File{
					{Path: "/project/testdata/fixture.json"},
				},
			},
			{
				Name: "customizations",
				Path: "/project/customizations",
				Files: []*model.File{
					{Path: "/project/customizations/theme.go"},
				},
			},
		},
		Files: []*model.File{
			{Path: "/project/main.go"},
		},
	}

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	cat, ok := root.Dirs[0].Files[0].Classification(metric.Name("file-role"))
	g.Expect(ok).To(BeTrue())
	g.Expect(cat).To(Equal("testdata"))

	cat, ok = root.Dirs[1].Files[0].Classification(metric.Name("file-role"))
	g.Expect(ok).To(BeTrue())
	g.Expect(cat).To(Equal("customized"))

	cat, ok = root.Files[0].Classification(metric.Name("file-role"))
	g.Expect(ok).To(BeTrue())
	g.Expect(cat).To(Equal("other"))
}

func TestProvider_Load_EmptyRules(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cfg := config.SelectionMetric{Name: "code-purpose", Rules: nil}
	p := classification.NewProvider(cfg)

	root := &model.Directory{Name: "root", Path: "/root"}
	file := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{file}

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())
}
