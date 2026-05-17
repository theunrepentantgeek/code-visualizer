package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"
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
