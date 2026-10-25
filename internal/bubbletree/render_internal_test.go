package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestArcFontSize_EmptyLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(bubbleArcFontSize("", 100)).To(Equal(0.0))
}

func TestArcFontSize_TinyRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(bubbleArcFontSize("test", 10)).To(Equal(0.0))
}

func TestArcFontSize_NormalLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fontSize := bubbleArcFontSize("normal", 100)

	g.Expect(fontSize).To(BeNumerically(">", 0))
	g.Expect(fontSize).To(BeNumerically(">=", bubbleMinArcFontSize))
	g.Expect(fontSize).To(BeNumerically("<=", bubbleDefaultFontSize))
}

func TestArcFontSize_LongLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	longLabel := "this_is_a_very_long_label_that_cannot_fit_on_a_small_circle"

	g.Expect(bubbleArcFontSize(longLabel, 30)).To(Equal(0.0))
}

func TestArcFontSize_AllowsFivePixelLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fontSize := bubbleArcFontSize("test", bubbleArcLabelInset+10)

	g.Expect(fontSize).To(BeNumerically(">=", 5))
	g.Expect(fontSize).To(BeNumerically("<", 7))
}

func TestDirectoryLabelFontSize_MinimumFivePixels(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		band float64
		want float64
	}{
		{name: "below-minimum", band: 4.9, want: 0},
		{name: "at-minimum", band: 5, want: 5},
		{name: "above-minimum", band: 6, want: 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			node := BubbleNode{
				Geometry:    geometry.NewCircle(geometry.OriginPoint, 100),
				Label:       "pkg",
				IsDirectory: true,
				ShowLabel:   true,
				labelBand:   tc.band,
			}

			g.Expect(bubbleDirLabelFontSize(node)).To(Equal(tc.want))
		})
	}
}
