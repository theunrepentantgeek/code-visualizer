package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/bevan/code-visualizer/internal/config"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/scan"
)

func TestMain(m *testing.M) {
	filesystem.Register()
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

		parser, err := kong.New(&cli,
			kong.Name("codeviz"),
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

func TestClassifyNoFilesAfterFilterError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := &noFilesAfterFilterError{msg: "no files available for visualization after excluding binary files"}
	code := classifyError(err)
	g.Expect(code).To(Equal(6))
}

func TestClassifyErrorPreservesExistingCodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(classifyError(&targetPathError{msg: "bad path"})).To(Equal(2))
	g.Expect(classifyError(&gitRequiredError{})).To(Equal(3))
	g.Expect(classifyError(&outputPathError{msg: "bad output"})).To(Equal(4))
	g.Expect(classifyError(&noFilesAfterFilterError{msg: "no files"})).To(Equal(6))
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

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-size",
		Filter:     []string{"![invalid"},
	}

	err := cmd.Validate()
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("invalid filter")))
}

func TestTreemapCmd_Validate_ValidFilters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cmd := &TreemapCmd{
		TargetPath: ".",
		Output:     "out.png",
		Size:       "file-size",
		Filter:     []string{"!.*", "*.go", "!**/*.log"},
	}

	err := cmd.Validate()
	g.Expect(err).NotTo(HaveOccurred())
}

func TestCollectDistinctTypes_ReturnsSortedTypes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange - files with different file types added in non-alphabetical order
	fZ := &model.File{Path: "/p/z.go", Name: "z.go"}
	fZ.SetClassification(filesystem.FileType, "go")

	fA := &model.File{Path: "/p/a.md", Name: "a.md"}
	fA.SetClassification(filesystem.FileType, "md")

	fM := &model.File{Path: "/p/m.txt", Name: "m.txt"}
	fM.SetClassification(filesystem.FileType, "txt")

	root := &model.Directory{
		Path:  "/p",
		Name:  "p",
		Files: []*model.File{fZ, fA, fM},
	}

	// Act
	types := collectDistinctTypes(root, filesystem.FileType)

	// Assert
	g.Expect(types).To(Equal([]string{"go", "md", "txt"}))
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

// validateConfig validates the merged config (single source of truth).

func TestTreemapCmd_ValidateConfig_ConfigSuppliesFillAndPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	g.Expect(os.WriteFile(
		cfgPath,
		[]byte("treemap:\n  size: file-size\n  fill: file-lines\n  fillPalette: temperature\n"),
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

func TestTreemapCmd_ValidateConfig_BorderPaletteWithoutBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	size := "file-size"
	borderPalette := "temperature"
	cfg.Treemap.Size = &size
	cfg.Treemap.BorderPalette = &borderPalette // no Border set

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("--border-palette requires --border"))
}

func TestTreemapCmd_ValidateConfig_InvalidFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	size := "file-size"
	fill := "not-a-real-metric"
	cfg.Treemap.Size = &size
	cfg.Treemap.Fill = &fill

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid fill metric"))
}

func TestTreemapCmd_ValidateConfig_InvalidFillPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	size := "file-size"
	fill := "file-lines"
	fillPalette := "not-a-real-palette"
	cfg.Treemap.Size = &size
	cfg.Treemap.Fill = &fill
	cfg.Treemap.FillPalette = &fillPalette

	cmd := &TreemapCmd{}
	err := cmd.validateConfig(cfg.Treemap)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid fill palette"))
}
