package classification

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestRegister_CreatesDescriptorAndLoader(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	t.Cleanup(provider.ResetBaseRegistryForTesting)

	cfg := config.SelectionMetric{Name: "test-metric"}
	Register(cfg)

	desc, ok := provider.GetBase("test-metric")
	g.Expect(ok).To(BeTrue())
	g.Expect(desc.Kind).To(Equal(metric.Classification))
	g.Expect(desc.Level).To(Equal(metric.LevelFile))

	loaders := provider.LoadersFor([]metric.Name{"test-metric"})
	g.Expect(loaders).To(HaveLen(1))
}

func TestLoad_MatchesFirstRule(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	l := &loader{
		name: "code-purpose",
		rules: []config.SelectionMetricRule{
			{Category: "test", Filename: "*_test.go"},
			{Category: "source", Filename: "*"},
		},
	}

	root := &model.Directory{Name: "root", Path: "/root"}
	testFile := &model.File{Path: "/root/foo_test.go"}
	srcFile := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{testFile, srcFile}

	err := l.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	testCat, ok := testFile.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(testCat).To(Equal("test"))

	srcCat, ok := srcFile.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(srcCat).To(Equal("source"))
}

func TestLoad_UnmatchedFileGetsNoValue(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	l := &loader{
		name: "code-purpose",
		rules: []config.SelectionMetricRule{
			{Category: "test", Filename: "*_test.go"},
		},
	}

	root := &model.Directory{Name: "root", Path: "/root"}
	file := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{file}

	err := l.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	_, ok := file.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeFalse(), "file not matching any rule should have no metric value")
}

func TestLoad_GeneratedFilePattern(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	l := &loader{
		name: "code-source",
		rules: []config.SelectionMetricRule{
			{Category: "gen", Filename: "*_gen.go"},
			{Category: "gen", Filename: "*_gen_test.go"},
			{Category: "authored", Filename: "*"},
		},
	}

	root := &model.Directory{Name: "root", Path: "/root"}
	genFile := &model.File{Path: "/root/schema_gen.go"}
	genTestFile := &model.File{Path: "/root/schema_gen_test.go"}
	authoredFile := &model.File{Path: "/root/service.go"}
	root.Files = []*model.File{genFile, genTestFile, authoredFile}

	err := l.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	cat, _ := genFile.Classification(metric.Name("code-source"))
	g.Expect(cat).To(Equal("gen"))

	cat, _ = genTestFile.Classification(metric.Name("code-source"))
	g.Expect(cat).To(Equal("gen"))

	cat, _ = authoredFile.Classification(metric.Name("code-source"))
	g.Expect(cat).To(Equal("authored"))
}

func TestLoad_MatchesRelativePath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	l := &loader{
		name: "file-role",
		rules: []config.SelectionMetricRule{
			{Category: "testdata", Filename: "testdata/**"},
			{Category: "customized", Filename: "*customizations*/*.go"},
			{Category: "other", Filename: "*"},
		},
	}

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

	err := l.Load(root)
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

func TestLoad_EmptyRules(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	l := &loader{name: "code-purpose"}

	root := &model.Directory{Name: "root", Path: "/root"}
	file := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{file}

	err := l.Load(root)
	g.Expect(err).NotTo(HaveOccurred())
}
