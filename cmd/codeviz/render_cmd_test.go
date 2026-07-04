package main

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestRenderCmd_Validate_NoArgs_ListMode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{}
	g.Expect(r.Validate()).To(Succeed())
}

func TestRenderCmd_Validate_KnownPreset_Valid(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{
		Preset:     "structure-tree-map",
		TargetPath: ".",
		Output:     "out.png",
	}
	g.Expect(r.Validate()).To(Succeed())
}

func TestRenderCmd_Validate_UnknownPreset_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{Preset: "does-not-exist", TargetPath: ".", Output: "out.png"}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("unknown preset")))
}

func TestRenderCmd_Validate_MissingTarget_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{Preset: "structure-tree-map", Output: "out.png"}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("target path is required")))
}

func TestRenderCmd_Validate_MissingOutput_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{Preset: "structure-tree-map", TargetPath: "."}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("output path")))
}

func TestRenderCmd_EffectiveTitle_UsesTitleWhenSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	preset := findPreset("structure-tree-map")
	g.Expect(preset).NotTo(BeNil(), "preset should exist")

	if preset != nil {
		r := &RenderCmd{Title: "Custom Title"}
		g.Expect(r.effectiveTitle(preset)).To(Equal("Custom Title"))
	}
}

func TestRenderCmd_EffectiveTitle_UsesPresetDefaultWhenTitleEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	preset := findPreset("structure-tree-map")
	g.Expect(preset).NotTo(BeNil(), "preset should exist")

	if preset != nil {
		r := &RenderCmd{}
		g.Expect(r.effectiveTitle(preset)).To(Equal("Code Structure"))
	}
}

func TestRenderCmd_AllPresets_RegisteredAndUnique(t *testing.T) {
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

func TestRenderCmd_ListPresets_UsesMetricsStyleLayout(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	output := captureStdout(t, func() {
		err := (&RenderCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	g.Expect(output).To(ContainSubstring("Presets\n───────"))
	g.Expect(output).NotTo(ContainSubstring("|"), "should not use ASCII table borders")

	for _, p := range presets {
		g.Expect(output).To(ContainSubstring(p.Name))
	}
}

func TestRenderCmd_ParsedFromCLI_NoArgs(t *testing.T) {
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

	ctx, parseErr := parser.Parse([]string{"render"})
	g.Expect(parseErr).NotTo(HaveOccurred())
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

func TestRenderCmd_StructureTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{TargetPath: "/src", Output: "out.png", Width: 800, Height: 600, HideFooter: true}
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

func TestRenderCmd_StructureBubbletree_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.structureBubbletree("Structure")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("file-type")}))
	g.Expect(cmd.Title).To(Equal("Structure"))
	g.Expect(cmd.TargetPath).To(Equal("/src"))
	g.Expect(cmd.Output).To(Equal("out.png"))
}

func TestRenderCmd_HistoryTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.historyTreemap("Hotspots")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("commit-count")}))
	g.Expect(cmd.Title).To(Equal("Hotspots"))
}

func TestRenderCmd_AgeTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.ageTreemap("File Age")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("file-age")}))
	g.Expect(cmd.Title).To(Equal("File Age"))
}

func TestRenderCmd_ContributorsTreemap_SetsCorrectMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{TargetPath: "/src", Output: "out.png", Width: 1920, Height: 1080}
	cmd := r.contributorsTreemap("Authors")

	g.Expect(cmd.Size).To(Equal(metric.Name("file-lines")))
	g.Expect(cmd.Fill).To(Equal(config.MetricSpec{Metric: metric.Name("author-count")}))
	g.Expect(cmd.Title).To(Equal("Authors"))
}

func TestRenderCmd_AllBuilders_PropagateTargetAndOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RenderCmd{TargetPath: "/repo", Output: "/tmp/result.svg", Width: 1280, Height: 720}

	g.Expect(r.structureTreemap("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.structureBubbletree("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.historyTreemap("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.ageTreemap("t").TargetPath).To(Equal("/repo"))
	g.Expect(r.contributorsTreemap("t").TargetPath).To(Equal("/repo"))

	g.Expect(r.structureTreemap("t").Output).To(Equal("/tmp/result.svg"))
	g.Expect(r.historyTreemap("t").Output).To(Equal("/tmp/result.svg"))
}
