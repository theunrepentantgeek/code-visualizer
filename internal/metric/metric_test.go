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

func TestKindDefaultAggregation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind Kind
		want AggregationName
		ok   bool
	}{
		{name: "quantity", kind: Quantity, want: AggSum, ok: true},
		{name: "measure", kind: Measure, want: AggMean, ok: true},
		{name: "classification", kind: Classification, want: AggMode, ok: true},
		{name: "unknown", kind: Kind(99), want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			got, ok := tt.kind.DefaultAggregation()
			g.Expect(got).To(Equal(tt.want))
			g.Expect(ok).To(Equal(tt.ok))
		})
	}
}
