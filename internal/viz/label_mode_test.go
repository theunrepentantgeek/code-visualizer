package viz

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestLabelModeConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(string(LabelAll)).To(Equal("all"))
	g.Expect(string(LabelFoldersOnly)).To(Equal("folders"))
	g.Expect(string(LabelLaps)).To(Equal("laps"))
	g.Expect(string(LabelNone)).To(Equal("none"))
}

func TestLabelModeDistinct(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	modes := []LabelMode{LabelAll, LabelFoldersOnly, LabelLaps, LabelNone}

	seen := make(map[LabelMode]bool)

	for _, m := range modes {
		g.Expect(seen[m]).To(BeFalse(), "duplicate LabelMode value: %q", m)
		seen[m] = true
	}
}

func TestLabelMode_IsStringType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var mode LabelMode = "custom"
	g.Expect(string(mode)).To(Equal("custom"))
}
