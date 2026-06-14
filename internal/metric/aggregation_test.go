package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAggregateSum_IntValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateSum([]float64{1, 2, 3, 4})).To(Equal(10.0))
}

func TestAggregateSum_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateSum(nil)).To(Equal(0.0))
}

func TestAggregateMin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMin([]float64{5, 2, 8, 1, 7})).To(Equal(1.0))
}

func TestAggregateMin_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMin(nil)).To(Equal(0.0))
}

func TestAggregateMax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMax([]float64{5, 2, 8, 1, 7})).To(Equal(8.0))
}

func TestAggregateMax_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMax(nil)).To(Equal(0.0))
}

func TestAggregateMean(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMean([]float64{2, 4, 6})).To(Equal(4.0))
}

func TestAggregateMean_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMean(nil)).To(Equal(0.0))
}

func TestAggregateCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateCount([]float64{1, 2, 3})).To(Equal(3.0))
}

func TestAggregateCount_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateCount(nil)).To(Equal(0.0))
}

func TestAggregateRange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateRange([]float64{3, 1, 7, 2})).To(Equal(6.0))
}

func TestAggregateRange_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateRange(nil)).To(Equal(0.0))
}

func TestAggregateRange_SingleElement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateRange([]float64{5})).To(Equal(0.0))
}

func TestAggregateMode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMode([]string{"go", "md", "go", "py", "go"})).To(Equal("go"))
}

func TestAggregateMode_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMode(nil)).To(Equal(""))
}

func TestAggregateMode_Tie_ReturnsLexFirst(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "go" and "py" both appear twice; "go" is lexicographically first
	g.Expect(AggregateMode([]string{"go", "py", "go", "py"})).To(Equal("go"))
}

func TestAggregateDistinct(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateDistinct([]string{"go", "md", "go", "py", "md"})).To(Equal(3))
}

func TestAggregateDistinct_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateDistinct(nil)).To(Equal(0))
}
