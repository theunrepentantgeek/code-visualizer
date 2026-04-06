package metric

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/palette"
)

func TestDefaultPaletteFor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tests := []struct {
		metric  MetricName
		palette palette.PaletteName
	}{
		{FileSize, palette.Neutral},
		{FileLines, palette.Neutral},
		{FileAge, palette.Temperature},
		{FileFreshness, palette.Temperature},
		{AuthorCount, palette.GoodBad},
		{FileType, palette.Categorization},
	}

	for _, tt := range tests {
		p, ok := DefaultPaletteFor(tt.metric)
		g.Expect(ok).To(BeTrue(), "expected default palette for %q", tt.metric)
		g.Expect(p).To(Equal(tt.palette), "wrong default palette for %q", tt.metric)
	}
}

func TestDefaultPaletteFor_InvalidMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, ok := DefaultPaletteFor(MetricName("unknown"))
	g.Expect(ok).To(BeFalse())
}
