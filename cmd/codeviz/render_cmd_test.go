package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"
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
