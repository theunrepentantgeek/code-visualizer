package treemap

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestHeaderFills_ExactDepthPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	want := [5]color.RGBA{
		{R: 0x20, G: 0x26, B: 0x31, A: 0xFF},
		{R: 0x2F, G: 0x3B, B: 0x4D, A: 0xFF},
		{R: 0x3D, G: 0x52, B: 0x68, A: 0xFF},
		{R: 0x51, G: 0x6A, B: 0x7D, A: 0xFF},
		{R: 0x5F, G: 0x78, B: 0x88, A: 0xFF},
	}

	g.Expect(headerFills).To(Equal(want))
}

func TestHeaderFills_MeetMinimumContrastAgainstWhite(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, fill := range headerFills {
		g.Expect(palette.ContrastRatio(fill, palette.White)).To(
			BeNumerically(">=", 4.5),
			"fill %v should contrast at least 4.5 against white", fill,
		)
	}
}
