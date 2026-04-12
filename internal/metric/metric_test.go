package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestKindConstants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Quantity).To(Equal(Kind(0)))
	g.Expect(Measure).To(Equal(Kind(1)))
	g.Expect(Classification).To(Equal(Kind(2)))
}
