package spiral

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSpiralBorderWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(borderWidth(7.9)).To(Equal(2.0))
	g.Expect(borderWidth(8.0)).To(Equal(3.0))
	g.Expect(borderWidth(10.0)).To(Equal(3.0))
}
