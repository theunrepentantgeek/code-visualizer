package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"
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
		Preset:     "structure-treemap",
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

	r := &RunCmd{Preset: "structure-treemap", Output: "out.png"}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("target path is required")))
}

func TestRunCmd_Validate_MissingOutput_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	r := &RunCmd{Preset: "structure-treemap", TargetPath: "."}
	g.Expect(r.Validate()).To(MatchError(ContainSubstring("output path")))
}

func TestRunCmd_EffectiveTitle_UsesTitleWhenSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	preset := findPreset("structure-treemap")
	g.Expect(preset).Error()NotTo(BeNil(), "preset should exist")

	if preset != nil {
		r := &RunCmd{Title: "Custom Title"}
		g.Expect(r.effectiveTitle(preset)).To(Equal("Custom Title"))
	}
}

func TestRunCmd_EffectiveTitle_UsesPresetDefaultWhenTitleEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	preset := findPreset("structure-treemap")
	r := &RunCmd{}
	g.Expect(r.effectiveTitle(preset)).To(Equal("Code Structure"))
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
	g.Expect(err).NotTo(HaveOccurred())

	ctx, parseErr := parser.Parse([]string{"run"})
	g.Expect(parseErr).NotTo(HaveOccurred())
	g.Expect(ctx).NotTo(BeNil())
}
