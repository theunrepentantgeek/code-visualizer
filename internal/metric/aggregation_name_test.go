package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAggregationName_StringConversion(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := AggregationName("sum")
	g.Expect(string(a)).To(Equal("sum"))
}

func TestAggregationName_IsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var a AggregationName
	g.Expect(a.IsZero()).To(BeTrue())
	g.Expect(AggregationName("sum").IsZero()).To(BeFalse())
}

func TestAggregationName_IsKnown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregationName("sum").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("min").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("max").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("mean").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("count").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("mode").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("distinct").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("range").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("bogus").IsKnown()).To(BeFalse())
	g.Expect(AggregationName("").IsKnown()).To(BeFalse())
}
