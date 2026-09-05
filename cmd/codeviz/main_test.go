package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	git.Register()
	golang.Register()
	m.Run()
}

func TestCLI_MutuallyExclusiveFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := []string{"tree-map", ".", "-o", "out.png", "-s", "file-size"}

	cases := []struct {
		args      []string
		expectErr bool
	}{
		{args: append([]string{"--quiet", "--verbose"}, cmd...), expectErr: true},
		{args: append([]string{"--quiet", "--debug"}, cmd...), expectErr: true},
		{args: append([]string{"--verbose", "--debug"}, cmd...), expectErr: true},
		{args: append([]string{"--quiet"}, cmd...), expectErr: false},
		{args: append([]string{"--verbose"}, cmd...), expectErr: false},
		{args: append([]string{"--debug"}, cmd...), expectErr: false},
	}

	for _, tc := range cases {
		cli := CLI{}

		parser, err := kong.New(
			&cli,
			kong.Name("codeviz"),
			filterMapperOption(),
			kong.Exit(func(int) {}),
		)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = parser.Parse(tc.args)

		if tc.expectErr {
			g.Expect(err).To(HaveOccurred(),
				"expected error for args %v", tc.args)
		} else {
			g.Expect(err).NotTo(HaveOccurred(),
				"expected no error for args %v", tc.args)
		}
	}
}

func TestCLI_ParsesTreemapFlatFlag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{"tree-map", ".", "-o", "out.png", "-s", "file-size", "--flat"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.TreeMap.Flat).To(BeTrue())
}

func TestCLI_ParsesUnifiedHistoryReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"tree-map", ".", "-o", "out.png",
		"--from", "sha:abc1234",
		"--until", "tag:v2.0",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.TreeMap.From).To(Equal("sha:abc1234"))
	g.Expect(cli.TreeMap.Until).To(Equal("tag:v2.0"))
}

func TestCLI_RejectsRemovedTagRangeFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"tree-map", ".", "-o", "out.png",
		"--from-tag", "v1.0",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestCLI_ParsesRadialFileAndDirectoryMetricFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"radial-tree", ".", "-o", "out.png",
		"--file-disc-size", "file-size",
		"--file-fill", "file-type,categorization",
		"--file-border", "file-freshness,good-bad",
		"--directory-disc-size", "file-size.sum",
		"--directory-fill", "file-type.mode,categorization",
		"--directory-border", "file-freshness.mean,good-bad",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.RadialTree.FileDiscSize).To(Equal(metric.Name("file-size")))
	g.Expect(cli.RadialTree.FileFill).To(Equal(config.MetricSpec{Metric: "file-type", Palette: "categorization"}))
	g.Expect(cli.RadialTree.FileBorder).To(Equal(config.MetricSpec{Metric: "file-freshness", Palette: "good-bad"}))
	g.Expect(cli.RadialTree.DirectoryDiscSize).To(Equal(metric.Name("file-size.sum")))
	g.Expect(cli.RadialTree.DirectoryFill).To(Equal(config.MetricSpec{
		Metric: "file-type.mode", Palette: "categorization",
	}))
	g.Expect(cli.RadialTree.DirectoryBorder).To(Equal(config.MetricSpec{
		Metric: "file-freshness.mean", Palette: "good-bad",
	}))
}

func TestCLI_ParsesDonutTreeMetricFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"donut-tree", ".", "-o", "out.png",
		"--size", "file-size",
		"--fill", "file-type,categorization",
		"--border", "file-freshness,good-bad",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.DonutTree.Size).To(Equal(metric.Name("file-size")))
	g.Expect(cli.DonutTree.Fill).To(Equal(config.MetricSpec{Metric: "file-type", Palette: "categorization"}))
	g.Expect(cli.DonutTree.Border).To(Equal(config.MetricSpec{Metric: "file-freshness", Palette: "good-bad"}))
}

func TestCLI_ParsesMaxLayersFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cases := []struct {
		name string
		args []string
		want int
	}{
		{name: "radial", args: []string{"radial-tree", ".", "-o", "out.png", "--max-layers", "2"}, want: 2},
		{name: "donut", args: []string{"donut-tree", ".", "-o", "out.png", "--max-layers", "3"}, want: 3},
	}

	for _, tc := range cases {
		cli := CLI{}
		parser, err := kong.New(
			&cli,
			kong.Name("codeviz"),
			filterMapperOption(),
			kong.Exit(func(int) {}),
		)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = parser.Parse(tc.args)
		g.Expect(err).NotTo(HaveOccurred(), tc.name)

		if tc.name == "radial" {
			g.Expect(cli.RadialTree.MaxLayers).To(Equal(tc.want), tc.name)
		} else {
			g.Expect(cli.DonutTree.MaxLayers).To(Equal(tc.want), tc.name)
		}
	}
}

func TestCLI_SpiralLabelsFlagIsUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{"spiral", ".", "-o", "out.png", "--labels", "all"})
	g.Expect(err).To(MatchError(ContainSubstring("unknown flag --labels")))
}

func TestCLI_BubbletreeLegendFlags_UseKongEnumValidation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "legend",
			args:    []string{"bubble-tree", ".", "-o", "out.png", "--legend", "sideways"},
			wantErr: "--legend must be one of",
		},
		{
			name:    "legend-orientation",
			args:    []string{"bubble-tree", ".", "-o", "out.png", "--legend-orientation", "diagonal"},
			wantErr: "--legend-orientation must be one of",
		},
	}

	for _, tc := range cases {
		cli := CLI{}
		parser, err := kong.New(
			&cli,
			kong.Name("codeviz"),
			filterMapperOption(),
			kong.Exit(func(int) {}),
		)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = parser.Parse(tc.args)
		g.Expect(err).To(MatchError(ContainSubstring(tc.wantErr)), tc.name)
	}
}

func TestClassifyNoFilesAfterFilterError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := &stages.NoFilesAfterFilterError{Msg: "no files available for visualization after excluding binary files"}
	code := classifyError(err)
	g.Expect(code).To(Equal(6))
}

func TestClassifyErrorPreservesExistingCodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(classifyError(&stages.TargetPathError{Msg: "bad path"})).To(Equal(2))
	g.Expect(classifyError(&stages.GitRequiredError{})).To(Equal(3))
	g.Expect(classifyError(&stages.OutputPathError{Msg: "bad output"})).To(Equal(4))
	g.Expect(classifyError(&stages.NoFilesAfterFilterError{Msg: "no files"})).To(Equal(6))
}

func TestFilterNotCalledForFileSizeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(filesystem.FileSize).NotTo(Equal(filesystem.FileLines))

	f := &model.File{Path: "/project/image.png", Name: "image.png", IsBinary: true}
	f.SetQuantity(filesystem.FileSize, 1024)
	root := &model.Directory{
		Path: "/project", Name: "project",
		Files: []*model.File{f},
	}
	g.Expect(countFilesInTree(root)).To(Equal(1))
}

func TestFilterNotCalledForFileAgeMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Verify that file-age metric does not trigger filtering
	g.Expect(filesystem.FileLines).NotTo(BeEquivalentTo("file-age"))
}

func TestFilterAppliedRegardlessOfFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fGo := &model.File{Path: "/project/main.go", Name: "main.go", IsBinary: false}
	fGo.SetQuantity(filesystem.FileLines, 50)
	fGo.SetClassification(filesystem.FileType, "go")

	fPng := &model.File{Path: "/project/image.png", Name: "image.png", IsBinary: true}
	fPng.SetQuantity(filesystem.FileSize, 1024)
	fPng.SetClassification(filesystem.FileType, "png")

	root := &model.Directory{
		Path: "/project", Name: "project",
		Files: []*model.File{fGo, fPng},
	}

	filtered := scan.FilterBinaryFiles(root)
	g.Expect(filtered.Files).To(HaveLen(1))
	g.Expect(filtered.Files[0].Name).To(Equal("main.go"))
}

func TestNoFilterWhenFileSizeWithFileTypeFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fGo := &model.File{Path: "/project/main.go", Name: "main.go", IsBinary: false}
	fGo.SetQuantity(filesystem.FileSize, 100)
	fGo.SetClassification(filesystem.FileType, "go")

	fPng := &model.File{Path: "/project/image.png", Name: "image.png", IsBinary: true}
	fPng.SetQuantity(filesystem.FileSize, 1024)
	fPng.SetClassification(filesystem.FileType, "png")

	root := &model.Directory{
		Path: "/project", Name: "project",
		Files: []*model.File{fGo, fPng},
	}

	// Without filtering, both files remain
	g.Expect(countFilesInTree(root)).To(Equal(2))
}

// countFilesInTree is a test helper that counts all files in a tree.
func countFilesInTree(node *model.Directory) int {
	count := len(node.Files)
	for _, d := range node.Dirs {
		count += countFilesInTree(d)
	}

	return count
}

func TestTreemapCmd_Validate_InvalidFilterGlob(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"tree-map", ".",
		"-o", "out.png",
		"-s", "file-size",
		"--exclude", "[invalid",
	})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("invalid exclude")))
}

func TestTreemapCmd_Validate_ValidFilters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-size",
		Include:    []filter.Rule{{Pattern: "*.go", Mode: filter.Include}},
		Exclude:    []filter.Rule{{Pattern: ".*", Mode: filter.Exclude}, {Pattern: "**/*.log", Mode: filter.Exclude}},
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestCLI_ParsesIncludeExcludeFiltersInArgumentOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"tree-map", ".",
		"-o", "out.png",
		"-s", "file-size",
		"--exclude", ".*",
		"--include", ".github/**",
		"--exclude", "**/*.log",
	})
	g.Expect(err).NotTo(HaveOccurred())
	expectRuleSliceField(g, cli.TreeMap, "Include", []filter.Rule{
		{Pattern: ".github/**", Mode: filter.Include},
	})
	expectRuleSliceField(g, cli.TreeMap, "Exclude", []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
		{Pattern: "**/*.log", Mode: filter.Exclude},
	})
	expectRuleSlice(g, cli.TreeMap.Filters(), []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
		{Pattern: ".github/**", Mode: filter.Include},
		{Pattern: "**/*.log", Mode: filter.Exclude},
	})
}

func expectRuleSliceField(g *WithT, cmd any, fieldName string, want []filter.Rule) {
	value := reflect.ValueOf(cmd)
	field := value.FieldByName(fieldName)
	g.Expect(field.IsValid()).To(BeTrue())
	g.Expect(field.Type()).To(Equal(reflect.TypeFor[[]filter.Rule]()))

	got, ok := reflect.TypeAssert[[]filter.Rule](field)
	g.Expect(ok).To(BeTrue())
	expectRuleSlice(g, got, want)
}

func expectRuleSlice(g *WithT, got, want []filter.Rule) {
	g.Expect(got).To(HaveLen(len(want)))

	for i := range want {
		g.Expect(got[i].Pattern).To(Equal(want[i].Pattern))
		g.Expect(got[i].Mode).To(Equal(want[i].Mode))
	}
}

// Issue #99 — config-supplied parameters bypass early validation.
// After the fix, Validate() no longer checks size/disc-size metrics;
// that validation moves to validateConfig() which validates the merged
// config (the single source of truth) rather than CLI struct fields.

func TestTreemapCmd_Validate_EmptySize_Passes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "", // will be supplied by config file later in Run()
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRadialCmd_Validate_EmptyFileDiscSize_Passes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &RadialCmd{
		TargetPath:   ".",
		Output:       "out.png",
		FileDiscSize: "", // will be supplied by config file later in Run()
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestBubbletreeCmd_Validate_EmptySize_Passes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &BubbletreeCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "", // will be supplied by config file later in Run()
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestTreemapCmd_ConfigSuppliesSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(cfgPath, []byte("tree-map:\n  size: file-size\n"), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "", // not supplied on CLI
	}

	cmd.applyOverrides(cfg)

	// Config supplies the size — it stays on the config, not the CLI struct.
	g.Expect(cfg.Treemap).NotTo(BeNil())
	g.Expect(cfg.Treemap.Size).NotTo(BeNil())
	g.Expect(*cfg.Treemap.Size).To(Equal("file-size"))
}

func TestTreemapCmd_CLISizeOverridesConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(cfgPath, []byte("tree-map:\n  size: file-size\n"), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-lines", // explicit CLI flag
	}

	cmd.applyOverrides(cfg)

	g.Expect(cfg.Treemap).NotTo(BeNil())
	g.Expect(cfg.Treemap.Size).NotTo(BeNil())
	g.Expect(*cfg.Treemap.Size).To(Equal("file-lines"))
}

func TestTreemapCmd_MissingSizeEverywhere_NilAfterMerge(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New() // default config does not set tree-map.size

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "", // not supplied on CLI
	}

	cmd.applyOverrides(cfg)

	// After merge with no size from either source, effective size is nil.
	// validateConfig (called from Run) should surface a clear error.
	g.Expect(cfg.Treemap.Size).To(BeNil())
}

func TestTreemapCmd_Run_WritesFileLabelsIntoSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "alpha.go")
	g.Expect(os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0o600)).To(Succeed())

	out := filepath.Join(dir, "treemap.svg")
	cmd := &TreemapCmd{
		TargetPath: dir,
		Output:     out,
		Size:       filesystem.FileLines,
		Width:      320,
		Height:     240,
	}

	flags := &Flags{Config: config.New()}
	g.Expect(cmd.Run(flags)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("alpha.go"))
}

// validateConfig validates the merged config (single source of truth).

func TestDonutTreeCmd_ConfigSuppliesSizeAndCLIOverridesIt(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.DonutTree.Size = new("file-size")

	cmd := &DonutTreeCmd{Size: "file-lines"}
	cmd.applyOverrides(cfg)

	g.Expect(*cfg.DonutTree.Size).To(Equal("file-lines"))
}

func TestDonutTreeCmd_OmittedDimensionsPreserveConfigValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{"donut-tree", ".", "-o", "out.png", "-s", "file-lines"})
	g.Expect(err).NotTo(HaveOccurred())

	cfg := config.New()
	*cfg.ImageSize.Width = 800
	*cfg.ImageSize.Height = 600

	cli.DonutTree.applyOverrides(cfg)

	g.Expect(*cfg.ImageSize.Width).To(Equal(800))
	g.Expect(*cfg.ImageSize.Height).To(Equal(600))
}

func TestDonutTreeCmd_ExplicitDimensionsOverrideConfigValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	*cfg.ImageSize.Width = 800
	*cfg.ImageSize.Height = 600

	(&DonutTreeCmd{Width: 2560, Height: 1440}).applyOverrides(cfg)

	g.Expect(*cfg.ImageSize.Width).To(Equal(2560))
	g.Expect(*cfg.ImageSize.Height).To(Equal(1440))
}

func TestDonutTreeCmd_ValidateConfig_MissingSizeFails(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := (&DonutTreeCmd{}).validateConfig(config.New().DonutTree)

	g.Expect(err).To(MatchError(ContainSubstring(`unknown size metric ""`)))
}

func TestDonutTreeCmd_ValidateConfig_ClassificationSizeFails(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.DonutTree.Size = new("file-type")

	err := (&DonutTreeCmd{}).validateConfig(cfg.DonutTree)

	g.Expect(err).To(MatchError(ContainSubstring("size metric must be numeric")))
}

func TestDonutTreeCmd_ValidateConfig_OmittedFillAndBorderPass(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.DonutTree.Size = new("file-size")

	err := (&DonutTreeCmd{}).validateConfig(cfg.DonutTree)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestDonutTreeCmd_ValidateConfig_InvalidFillFailsWithContext(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.DonutTree.Size = new("file-size")
	cfg.DonutTree.Fill = &config.MetricSpec{Metric: "not-a-real-metric"}

	err := (&DonutTreeCmd{}).validateConfig(cfg.DonutTree)

	g.Expect(err).To(MatchError(ContainSubstring("invalid fill spec: invalid fill metric")))
}

func TestDonutTreeCmd_ValidateConfig_InvalidBorderFailsWithContext(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.DonutTree.Size = new("file-size")
	cfg.DonutTree.Border = &config.MetricSpec{Metric: "not-a-real-metric"}

	err := (&DonutTreeCmd{}).validateConfig(cfg.DonutTree)

	g.Expect(err).To(MatchError(ContainSubstring("invalid border spec: invalid border metric")))
}

func TestTreemapCmd_ValidateConfig_ConfigSuppliesFillAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(
		cfgPath,
		[]byte("tree-map:\n  size: file-size\n  fill: file-lines,temperature\n"),
		0o600,
	)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &TreemapCmd{Output: "out.png"}
	cmd.applyOverrides(cfg)

	// Validation passes with values from config only — no CLI fill/palette needed.
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestTreemapCmd_ValidateConfig_InvalidFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Treemap.Size = new("file-size")
	cfg.Treemap.Fill = &config.MetricSpec{Metric: "not-a-real-metric"}

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).To(MatchError(ContainSubstring("invalid fill metric")))
}

func TestTreemapCmd_ValidateConfig_InvalidFillPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Treemap.Size = new("file-size")
	cfg.Treemap.Fill = &config.MetricSpec{Metric: "file-lines", Palette: "not-a-real-palette"}

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).To(MatchError(ContainSubstring("invalid fill palette")))
}

func TestTreemapCmd_ValidateConfig_InvalidSizeMetricListsAvailableMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Treemap.Size = new("not-a-real-metric")

	cmd := &TreemapCmd{}

	err := cmd.validateConfig(cfg.Treemap)
	if err == nil {
		t.Fatal("expected error")
	}

	g.Expect(err).To(MatchError(ContainSubstring(`unknown size metric "not-a-real-metric"`)))
	g.Expect(err).To(MatchError(ContainSubstring("available metrics:")))
	g.Expect(err).To(MatchError(ContainSubstring("file-size")))
}

func TestTreemapCmd_ValidateConfig_MeasureMetricAccepted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Treemap.Size = new("commit-density")

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestSpiralCmd_Validate_EmptySize_Passes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &SpiralCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "", // will be supplied by config file later in Run()
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestSpiralCmd_ConfigSuppliesSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(cfgPath, []byte("spiral:\n  size: file-size\n"), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &SpiralCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "", // not supplied on CLI
	}

	cmd.applyOverrides(cfg)

	// Config supplies the size — it stays on the config, not the CLI struct.
	g.Expect(cfg.Spiral).NotTo(BeNil())
	g.Expect(cfg.Spiral.Size).NotTo(BeNil())
	g.Expect(*cfg.Spiral.Size).To(Equal("file-size"))
}

func TestSpiralCmd_CLISizeOverridesConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(cfgPath, []byte("spiral:\n  size: file-size\n"), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &SpiralCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-lines", // explicit CLI flag
	}

	cmd.applyOverrides(cfg)

	g.Expect(cfg.Spiral).NotTo(BeNil())
	g.Expect(cfg.Spiral.Size).NotTo(BeNil())
	g.Expect(*cfg.Spiral.Size).To(Equal("file-lines"))
}

func TestSpiralCmd_ValidateConfig_InvalidFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.Size = new("file-size")
	cfg.Spiral.Fill = &config.MetricSpec{Metric: "not-a-real-metric"}

	cmd := &SpiralCmd{}
	err := cmd.validateConfig(cfg.Spiral)
	g.Expect(err).To(MatchError(ContainSubstring("invalid fill metric")))
}

func TestSpiralCmd_ValidateConfig_InvalidFillPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.Size = new("file-size")
	cfg.Spiral.Fill = &config.MetricSpec{Metric: "file-lines", Palette: "not-a-real-palette"}

	cmd := &SpiralCmd{}
	err := cmd.validateConfig(cfg.Spiral)
	g.Expect(err).To(MatchError(ContainSubstring("invalid fill palette")))
}

func TestCLI_ParsesSpiralSurfaceFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"spiral", ".", "-o", "out.png",
		"--surface",
		"--surface-metric", "file-lines,foliage",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.Spiral.Surface).To(BeTrue())
	g.Expect(cli.Spiral.SurfaceMetric).To(Equal(config.MetricSpec{
		Metric: metric.Name("file-lines"), Palette: "foliage",
	}))
}

func TestSpiralCmd_ValidateConfig_SurfaceDisabledLeavesFillOptional(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := (&SpiralCmd{}).validateConfig(config.New().Spiral)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestSpiralCmd_ValidateConfig_SurfaceNeedsMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.Surface = new(true)

	err := (&SpiralCmd{}).validateConfig(cfg.Spiral)

	g.Expect(err).To(MatchError(ContainSubstring(
		"surface requires a numeric fill metric or surface metric",
	)))
}

func TestSpiralCmd_ValidateConfig_SurfaceMetricMustBeNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.SurfaceMetric = &config.MetricSpec{Metric: "file-type"}

	err := (&SpiralCmd{}).validateConfig(cfg.Spiral)

	g.Expect(err).To(MatchError(ContainSubstring("surface metric must be numeric")))
}

func TestSpiralCmd_ValidateConfig_SurfaceMetricMustNotUseAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.SurfaceMetric = &config.MetricSpec{Metric: "file-size.sum"}

	err := (&SpiralCmd{}).validateConfig(cfg.Spiral)

	g.Expect(err).To(MatchError(ContainSubstring("surface metric must not use aggregation")))
}

func TestSpiralCmd_ValidateConfig_SurfaceFallbackFillMustBeNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.Surface = new(true)
	cfg.Spiral.Fill = &config.MetricSpec{Metric: "file-type"}

	err := (&SpiralCmd{}).validateConfig(cfg.Spiral)

	g.Expect(err).To(MatchError(ContainSubstring("surface metric must be numeric")))
}

func TestSpiralCmd_ValidateConfig_NumericSurfaceMetricWithoutFillPasses(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Spiral.SurfaceMetric = &config.MetricSpec{Metric: "file-lines"}

	err := (&SpiralCmd{}).validateConfig(cfg.Spiral)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestCLI_ParsesScatterAxisFlags(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"scatter", ".",
		"-o", "out.png",
		"--x-axis", "file-type",
		"--y-axis", "file-lines",
		"-s", "file-size",
		"--grain", "directory",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.Scatter.XAxis).To(Equal(metric.Name("file-type")))
	g.Expect(cli.Scatter.YAxis).To(Equal(metric.Name("file-lines")))
	g.Expect(cli.Scatter.Size).To(Equal(metric.Name("file-size")))
	g.Expect(cli.Scatter.Grain).To(Equal("directory"))
}

func TestScatterCmd_Validate_EmptyAxesPass(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &ScatterCmd{
		TargetPath: ".",
		Output:     "out.png",
		XAxis:      "",
		YAxis:      "",
		Size:       "",
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_CategoricalAxesAreAccepted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("file-type")
	cfg.Scatter.YAxis = new("file-lines")
	cfg.Scatter.Size = new("file-size")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_XAxisIsRequired(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.YAxis = new("file-lines")
	cfg.Scatter.Size = new("file-size")

	err := (&ScatterCmd{}).validateConfig(cfg.Scatter)

	g.Expect(err).To(MatchError("x-axis metric is required"))
}

func TestScatterCmd_ValidateConfig_YAxisIsRequired(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.XAxis = new("file-type")
	cfg.Scatter.Size = new("file-size")

	err := (&ScatterCmd{}).validateConfig(cfg.Scatter)

	g.Expect(err).To(MatchError("y-axis metric is required"))
}

func TestScatterCmd_ValidateConfig_SizeIsRequired(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.XAxis = new("file-type")
	cfg.Scatter.YAxis = new("file-lines")

	err := (&ScatterCmd{}).validateConfig(cfg.Scatter)

	g.Expect(err).To(MatchError("size metric is required"))
}

func TestScatterCmd_ValidateConfig_SizeMustBeNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Scatter.XAxis = new("file-type")
	cfg.Scatter.YAxis = new("file-lines")
	cfg.Scatter.Size = new("file-type")

	cmd := &ScatterCmd{}
	err := cmd.validateConfig(cfg.Scatter)
	g.Expect(err).To(MatchError(ContainSubstring("size metric must be numeric")))
}

func TestScatterCmd_ValidateConfig_DirectoryGrainAcceptsAggregatedMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.Grain = new("directory")
	cfg.Scatter.XAxis = new("file-lines.sum")
	cfg.Scatter.YAxis = new("file-size.sum")
	cfg.Scatter.Size = new("file-size.sum")

	err := (&ScatterCmd{}).validateConfig(cfg.Scatter)

	g.Expect(err).NotTo(HaveOccurred())
}

func TestScatterCmd_ValidateConfig_DirectoryGrainRejectsBareFileMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.Grain = new("directory")
	cfg.Scatter.XAxis = new("file-lines")
	cfg.Scatter.YAxis = new("file-size.sum")
	cfg.Scatter.Size = new("file-size.sum")

	err := (&ScatterCmd{}).validateConfig(cfg.Scatter)

	g.Expect(err).To(MatchError(ContainSubstring("requires aggregation at directory level")))
}

func TestScatterCmd_ValidateConfig_RejectsUnknownGrain(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.Grain = new("package")
	cfg.Scatter.XAxis = new("file-lines")
	cfg.Scatter.YAxis = new("file-size")
	cfg.Scatter.Size = new("file-size")

	err := (&ScatterCmd{}).validateConfig(cfg.Scatter)

	g.Expect(err).To(MatchError(ContainSubstring(`unknown grain "package"`)))
}

func TestScatterCmd_ConfigSuppliesAxesAndSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	configText := "scatter:\n  xAxis: file-type\n  yAxis: file-lines\n  size: file-size\n"
	g.Expect(os.WriteFile(cfgPath, []byte(configText), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &ScatterCmd{TargetPath: ".", Output: "out.png"}
	cmd.applyOverrides(cfg)

	g.Expect(cfg.Scatter).NotTo(BeNil())
	g.Expect(cfg.Scatter.XAxis).NotTo(BeNil())
	g.Expect(*cfg.Scatter.XAxis).To(Equal("file-type"))
	g.Expect(cfg.Scatter.YAxis).NotTo(BeNil())
	g.Expect(*cfg.Scatter.YAxis).To(Equal("file-lines"))
	g.Expect(cfg.Scatter.Size).NotTo(BeNil())
	g.Expect(*cfg.Scatter.Size).To(Equal("file-size"))
}

func TestScatterCmd_CLIAxesOverrideConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	configText := "scatter:\n  xAxis: file-lines\n  yAxis: file-size\n  size: file-size\n"
	g.Expect(os.WriteFile(cfgPath, []byte(configText), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &ScatterCmd{
		TargetPath: ".",
		Output:     "out.png",
		XAxis:      "file-type",
		YAxis:      "file-lines",
		Size:       "file-size",
	}
	cmd.applyOverrides(cfg)

	g.Expect(cfg.Scatter).NotTo(BeNil())
	g.Expect(*cfg.Scatter.XAxis).To(Equal("file-type"))
	g.Expect(*cfg.Scatter.YAxis).To(Equal("file-lines"))
	g.Expect(*cfg.Scatter.Size).To(Equal("file-size"))
}

func TestScatterCmd_CLIGrainOverridesConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := config.New()
	cfg.Scatter.Grain = new("file")

	(&ScatterCmd{Grain: "directory"}).applyOverrides(cfg)

	g.Expect(*cfg.Scatter.Grain).To(Equal("directory"))
}

func TestScatterCmd_MergeConfigAndValidate_LoadsScatterConfig(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	configText := "scatter:\n  xAxis: file-type\n  yAxis: file-lines\n  size: file-size\n"
	g.Expect(os.WriteFile(cfgPath, []byte(configText), 0o600)).To(Succeed())

	cfg := config.New()
	g.Expect(cfg.Load(cfgPath)).To(Succeed())

	cmd := &ScatterCmd{TargetPath: ".", Output: filepath.Join(dir, "out.png")}
	flags := &Flags{Config: cfg, configPath: cfgPath}

	err := cmd.mergeConfigAndValidate(flags)
	g.Expect(err).NotTo(HaveOccurred())
}
