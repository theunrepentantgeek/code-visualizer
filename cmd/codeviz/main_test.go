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

	cmd := []string{"render", "treemap", ".", "-o", "out.png", "-s", "file-size"}

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

	_, err = parser.Parse([]string{"render", "treemap", ".", "-o", "out.png", "-s", "file-size", "--flat"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.Render.Treemap.Flat).To(BeTrue())
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
			args:    []string{"render", "bubbletree", ".", "-o", "out.png", "--legend", "sideways"},
			wantErr: "--legend must be one of",
		},
		{
			name:    "legend-orientation",
			args:    []string{"render", "bubbletree", ".", "-o", "out.png", "--legend-orientation", "diagonal"},
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
	g.Expect(filesystem.FileLines).NotTo(Equal("file-age"))
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
		"render", "treemap", ".",
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
		"render", "treemap", ".",
		"-o", "out.png",
		"-s", "file-size",
		"--exclude", ".*",
		"--include", ".github/**",
		"--exclude", "**/*.log",
	})
	g.Expect(err).NotTo(HaveOccurred())
	expectRuleSliceField(g, cli.Render.Treemap, "Include", []filter.Rule{
		{Pattern: ".github/**", Mode: filter.Include},
	})
	expectRuleSliceField(g, cli.Render.Treemap, "Exclude", []filter.Rule{
		{Pattern: ".*", Mode: filter.Exclude},
		{Pattern: "**/*.log", Mode: filter.Exclude},
	})
	expectRuleSlice(g, cli.Render.Treemap.Filters(), []filter.Rule{
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

	got, ok := field.Interface().([]filter.Rule)
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

func TestRadialCmd_Validate_EmptyDiscSize_Passes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &RadialCmd{
		TargetPath: ".",
		Output:     "out.png",
		DiscSize:   "", // will be supplied by config file later in Run()
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
	g.Expect(os.WriteFile(cfgPath, []byte("treemap:\n  size: file-size\n"), 0o600)).To(Succeed())

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
	g.Expect(os.WriteFile(cfgPath, []byte("treemap:\n  size: file-size\n"), 0o600)).To(Succeed())

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

	cfg := config.New() // default config does not set treemap.size

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

func TestTreemapCmd_ValidateConfig_ConfigSuppliesFillAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(
		cfgPath,
		[]byte("treemap:\n  size: file-size\n  fill: file-lines,temperature\n"),
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

	errText := err.Error()
	g.Expect(errText).To(ContainSubstring(`unknown size metric "not-a-real-metric"`))
	g.Expect(errText).To(ContainSubstring("available metrics:"))
	g.Expect(errText).To(ContainSubstring("file-size"))
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
		"render", "scatter", ".",
		"-o", "out.png",
		"--x-axis", "file-type",
		"--y-axis", "file-lines",
		"-s", "file-size",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.Render.Scatter.XAxis).To(Equal(metric.Name("file-type")))
	g.Expect(cli.Render.Scatter.YAxis).To(Equal(metric.Name("file-lines")))
	g.Expect(cli.Render.Scatter.Size).To(Equal(metric.Name("file-size")))
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
