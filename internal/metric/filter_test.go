package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFilterName_StringConversion(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := FilterName("public")
	g.Expect(string(f)).To(Equal("public"))
}

func TestFilterName_EmptyIsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var f FilterName
	g.Expect(f.IsZero()).To(BeTrue())
	g.Expect(FilterName("public").IsZero()).To(BeFalse())
}
