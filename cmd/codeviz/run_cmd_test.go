package main

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestRunCmd_Validate_NoArgs_ListMode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{}
	g.Expect(r.Validate()).To(Succeed())
}

func TestRunCmd_Validate_KnownPreset_Valid(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{
		Preset:     "structure-tree-map",
		TargetPath: ".",
		Output:     "out.png",
	}
	g.Expect(r.Validate()).To(Succeed())
}

func TestRunCmd_Validate_UnknownPreset_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{Preset: "does-not-exist", TargetPath: ".", Output: "out.png"}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("unknown preset")))
}

func TestRunCmd_Validate_MissingTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{Preset: "structure-tree-map", Output: "out.png"}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("target path is required")))
}

func TestRunCmd_Validate_MissingOutput_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{Preset: "structure-tree-map", TargetPath: "."}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("output path")))
}

func TestRunCmd_EffectiveTitle_UsesTitleWhenSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	preset := findPreset("structure-tree-map")
	g.Expect(preset).NotTo(BeNil(), "preset should exist")

	if preset != nil {
		r := &RunCmd{Title: "Custom Title"}
		g.Expect(r.effectiveTitle(preset)).To(Equal("Custom Title"))
	}
}

func TestRunCmd_EffectiveTitle_UsesPresetDefaultWhenTitleEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	preset := findPreset("structure-tree-map")
	g.Expect(preset).NotTo(BeNil(), "preset should exist")

	if preset != nil {
		r := &RunCmd{}
		g.Expect(r.effectiveTitle(preset)).To(Equal("Code Structure"))
	}
}

func TestRunCmd_AllPresets_RegisteredAndUnique(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := make(map[string]bool)

	for _, p := range presets {
		g.Expect(p.Name).NotTo(BeEmpty(), "preset must have a name")
		g.Expect(p.Description).NotTo(BeEmpty(), "preset %q must have a description", p.Name)
		g.Expect(p.DefaultTitle).NotTo(BeEmpty(), "preset %q must have a default title", p.Name)
		g.Expect(names[p.Name]).To(BeFalse(), "preset name %q must be unique", p.Name)
		names[p.Name] = true
	}
}

func TestRunCmd_ParsedFromCLI_NoArgs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}

	parser, err := kong.New(
		&cli,
		kong.Name("codeviz"),
		filterMapperOption(),
		kong.Exit(func(int) {}),
	)
	g.Expect(err).ToNot(HaveOccurred())

	ctx, parseErr := parser.Parse([]string{"run"})
	g.Expect(parseErr).ToNot(HaveOccurred())
	g.Expect(ctx).NotTo(BeNil())
}

func TestFindPreset_KnownName_ReturnsPreset(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := findPreset("history-tree-map")
	g.Expect(p).NotTo(BeNil(), "preset should exist")

	if p != nil {
		g.Expect(p.Name).To(Equal("history-tree-map"))
	}
}

func TestFindPreset_UnknownName_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(findPreset("not-a-preset")).To(BeNil())
}

func TestPresetNames_ContainsAllPresets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := presetNames()
	for _, p := range presets {
		g.Expect(names).To(ContainSubstring(p.Name), "presetNames should include %q", p.Name)
	}
}

func TestPresetNames_IsSingleLine(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	names := presetNames()
	g.Expect(strings.Contains(names, "\n")).To(BeFalse(), "presetNames should not contain newlines")
}

func TestRunCmd_StructureTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{TargetPath: "/src", Output: "out.png", Width: 800, Height: 600, HideFooter: true}
	cmd := r.structureTreemap("Structure")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("file-type")}))
	g.Expect(cmd.Title).To(Equal("Structure"))
	g.Expect(cmd.TargetPath).To(Equal("/src"))
	g.Expect(cmd.Output).To(Equal("out.png"))
	g.Expect(cmd.Width).To(Equal(800))
	g.Expect(cmd.Height).To(Equal(600))
	g.Expect(cmd.HideFooter).To(BeTrue())
}

func TestRunCmd_StructureBubbletree_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.structureBubbletree("Structure")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("file-type")}))
	g.Expect(cmd.Title).To(Equal("Structure"))
	g.Expect(cmd.TargetPath).To(Equal("/src"))
	g.Expect(cmd.Output).To(Equal("out.png"))
}

func TestRunCmd_HistoryTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.historyTreemap("Hotspots")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("commit-count")}))
	g.Expect(cmd.Title).To(Equal("Hotspots"))
}

func TestRunCmd_AgeTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.ageTreemap("File Age")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("file-age")}))
	g.Expect(cmd.Title).To(Equal("File Age"))
}

func TestRunCmd_ContributorsTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.contributorsTreemap("Authors")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("author-count")}))
	g.Expect(cmd.Title).To(Equal("Authors"))
}

func TestRunCmd_AllBuilders_PropagateTargetAndOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{TargetPath: "/repo", Output: "/tmp/result.svg", Width: 1280, Height: 720}

	g.Expect(r.structureTreemap("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.structureBubbletree("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.historyTreemap("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.ageTreemap("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.contributorsTreemap("t").TargetPath).To(Equal("/repo"))

	g.Expect(r.structureTreemap("t").Output).To(Equal("/tmp/result.svg"))
	g.Expect(r.historyTreemap("t").Output).To(Equal("/tmp/result.svg"))
}
