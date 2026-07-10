package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

//nolint:paralleltest // captureStdout swaps global os.Stdout, so this test cannot run in parallel
func TestHelpPalettesCmdRun_UsesMetricsStyleLayout(t *testing.T) {
	g := NewGomegaWithT(t)

	output := captureStdout(t, func() {
		err := (HelpPalettesCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	g.Expect(output).To(ContainSubstring("Palettes\n────────"))
	g.Expect(output).NotTo(ContainSubstring("|"), "should not use ASCII table borders")
	g.Expect(output).To(ContainSubstring("For colour swatches, see:"))

	for _, info := range palette.Infos() {
		g.Expect(output).To(ContainSubstring(string(info.Name)))
	}
}

func TestHelpPalettesCmdRun_WrapsOutputToConsoleWidth(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Setenv("COLUMNS", "40")

	output := captureStdout(t, func() {
		err := (HelpPalettesCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "For colour swatches") {
			continue // the footer URL is intentionally not wrapped
		}

		g.Expect(utf8.RuneCountInString(line)).To(BeNumerically("<=", 40), "line exceeds console width: %q", line)
	}
}
