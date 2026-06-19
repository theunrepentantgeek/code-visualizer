package classification_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"go.yaml.in/yaml/v3"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/classification"
)

func resetRegistry(t *testing.T) {
	t.Helper()
	t.Cleanup(provider.ResetBaseRegistryForTesting)
	provider.ResetBaseRegistryForTesting()
}

// buildCfg constructs a Config with a single selection metric defined via YAML.
func buildCfg(t *testing.T, name string, rules ...string) *config.Config {
	t.Helper()

	var sb strings.Builder
	fmt.Fprint(&sb, "selectionMetrics:\n")
	fmt.Fprintf(&sb, "  %s:\n", name)

	for i := 0; i+1 < len(rules); i += 2 {
		fmt.Fprintf(&sb, "    - category: %s\n      filename: %q\n", rules[i], rules[i+1])
	}

	cfg := config.New()

	err := yaml.Unmarshal([]byte(sb.String()), cfg)
	if err != nil {
		t.Fatalf("buildCfg: yaml.Unmarshal: %v", err)
	}

	return cfg
}

func TestRegister_RegistersMetricDescriptor(t *testing.T) { //nolint:paralleltest // Uses global base registry.
	// Not parallel: mutates the global base registry.
	resetRegistry(t)

	g := NewWithT(t)

	cfg := buildCfg(t, "code-purpose", "test", "*_test.go", "source", "*")

	classification.Register(cfg)

	desc, ok := provider.GetBase(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(desc.Kind).To(Equal(metric.Classification))
	g.Expect(desc.Level).To(Equal(metric.LevelFile))
}

func TestRegister_Idempotent(t *testing.T) { //nolint:paralleltest // Uses global base registry.
	// Not parallel: mutates the global base registry.
	resetRegistry(t)

	g := NewWithT(t)

	cfg := buildCfg(t, "code-purpose", "test", "*_test.go")

	// Should not panic on second call.
	g.Expect(func() {
		classification.Register(cfg)
		classification.Register(cfg)
	}).NotTo(Panic())
}

func TestRegister_Load_MatchesFirstRule(t *testing.T) { //nolint:paralleltest // Uses global base registry.
	// Not parallel: mutates the global base registry.
	resetRegistry(t)

	g := NewWithT(t)

	cfg := buildCfg(t, "code-purpose", "test", "*_test.go", "source", "*")

	classification.Register(cfg)

	root := &model.Directory{Name: "root", Path: "/root"}
	testFile := &model.File{Path: "/root/foo_test.go"}
	srcFile := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{testFile, srcFile}

	loaders := provider.LoadersFor([]metric.Name{"code-purpose"})
	g.Expect(loaders).To(HaveLen(1))

	err := loaders[0].Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	testCat, ok := testFile.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(testCat).To(Equal("test"))

	srcCat, ok := srcFile.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeTrue())
	g.Expect(srcCat).To(Equal("source"))
}

func TestRegister_Load_UnmatchedFileGetsNoValue(t *testing.T) { //nolint:paralleltest // Uses global base registry.
	// Not parallel: mutates the global base registry.
	resetRegistry(t)

	g := NewWithT(t)

	cfg := buildCfg(t, "code-purpose", "test", "*_test.go")

	classification.Register(cfg)

	root := &model.Directory{Name: "root", Path: "/root"}
	file := &model.File{Path: "/root/bar.go"}
	root.Files = []*model.File{file}

	loaders := provider.LoadersFor([]metric.Name{"code-purpose"})
	g.Expect(loaders).To(HaveLen(1))

	err := loaders[0].Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	_, ok := file.Classification(metric.Name("code-purpose"))
	g.Expect(ok).To(BeFalse(), "file not matching any rule should have no metric value")
}

func TestRegister_Load_MatchesRelativePath(t *testing.T) { //nolint:paralleltest // Uses global base registry.
	// Not parallel: mutates the global base registry.
	resetRegistry(t)

	g := NewWithT(t)

	cfg := buildCfg(t, "file-role", "testdata", "testdata/**", "other", "*")

	classification.Register(cfg)

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
		},
		Files: []*model.File{
			{Path: "/project/main.go"},
		},
	}

	loaders := provider.LoadersFor([]metric.Name{"file-role"})
	g.Expect(loaders).To(HaveLen(1))

	err := loaders[0].Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	cat, ok := root.Dirs[0].Files[0].Classification(metric.Name("file-role"))
	g.Expect(ok).To(BeTrue())
	g.Expect(cat).To(Equal("testdata"))

	cat, ok = root.Files[0].Classification(metric.Name("file-role"))
	g.Expect(ok).To(BeTrue())
	g.Expect(cat).To(Equal("other"))
}

func TestRegister_Load_EmptyRules(t *testing.T) { //nolint:paralleltest // Uses global base registry.
	// Not parallel: uses the global base registry.
	resetRegistry(t)

	g := NewWithT(t)

	cfg := config.New() // no selection-metrics configured

	classification.Register(cfg)

	// No loaders should be registered.
	loaders := provider.LoadersFor([]metric.Name{"code-purpose"})
	g.Expect(loaders).To(BeEmpty())
}
