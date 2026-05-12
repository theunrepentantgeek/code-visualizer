package legendlayout

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewBasicMeasurer_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	m := NewBasicMeasurer()
	g.Expect(m).NotTo(BeNil())
}

func TestBasicMeasurer_MeasureString_NonEmptyReturnsPositiveWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	m := NewBasicMeasurer()
	w, h := m.MeasureString("hello")
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(h).To(BeNumerically(">", 0))
}

func TestBasicMeasurer_MeasureString_EmptyReturnsZeroWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	m := NewBasicMeasurer()
	w, _ := m.MeasureString("")
	g.Expect(w).To(BeZero())
}

func TestBasicMeasurer_MeasureString_LongerStringIsWider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	m := NewBasicMeasurer()
	wShort, _ := m.MeasureString("ab")
	wLong, _ := m.MeasureString("abcdef")
	g.Expect(wLong).To(BeNumerically(">", wShort))
}
