package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestComputeBuckets_EvenDistribution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90}
	b := ComputeBuckets(values, 3)
	g.Expect(b.StepCount).To(Equal(3))
	g.Expect(b.Min).To(Equal(10.0))
	g.Expect(b.Max).To(Equal(90.0))
	g.Expect(b.Boundaries).To(HaveLen(2)) // N-1 breakpoints for N steps
}

func TestComputeBuckets_SkewedDistribution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Most values are small, a few are very large
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 1000}
	b := ComputeBuckets(values, 3)
	g.Expect(b.StepCount).To(Equal(3))
	g.Expect(b.Min).To(Equal(1.0))
	g.Expect(b.Max).To(Equal(1000.0))
	g.Expect(b.Boundaries).To(HaveLen(2))
}

func TestComputeBuckets_SingleValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{42}
	b := ComputeBuckets(values, 5)
	g.Expect(b.Min).To(Equal(42.0))
	g.Expect(b.Max).To(Equal(42.0))
	g.Expect(b.StepCount).To(Equal(5))
}

func TestComputeBuckets_AllSameValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{7, 7, 7, 7, 7}
	b := ComputeBuckets(values, 3)
	g.Expect(b.Min).To(Equal(7.0))
	g.Expect(b.Max).To(Equal(7.0))
}

func TestComputeBuckets_BoundaryRounding(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Values that should produce boundaries rounded to 2 sig figs
	values := []float64{123, 456, 789, 1234, 5678, 9012}

	b := ComputeBuckets(values, 3)
	for _, boundary := range b.Boundaries {
		// Each boundary should be rounded to 2 significant figures
		g.Expect(boundary).To(Equal(roundToSigFigs(boundary, 2)))
	}
}

func TestComputeBuckets_DeduplicationAfterRounding(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Values very close together — rounding may produce duplicate boundaries
	values := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108}
	b := ComputeBuckets(values, 9)
	// After deduplication, boundaries should have no duplicates
	seen := map[float64]bool{}
	for _, boundary := range b.Boundaries {
		g.Expect(seen[boundary]).To(BeFalse(), "duplicate boundary: %v", boundary)
		seen[boundary] = true
	}
}

func TestBucketIndex(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := BucketBoundaries{
		Boundaries: []float64{30, 60},
		Min:        10,
		Max:        90,
		StepCount:  3,
	}

	g.Expect(b.BucketIndex(10)).To(Equal(0)) // at min → first bucket
	g.Expect(b.BucketIndex(25)).To(Equal(0)) // below first boundary
	g.Expect(b.BucketIndex(30)).To(Equal(1)) // at first boundary → second bucket
	g.Expect(b.BucketIndex(50)).To(Equal(1)) // between boundaries
	g.Expect(b.BucketIndex(60)).To(Equal(2)) // at second boundary → third bucket
	g.Expect(b.BucketIndex(90)).To(Equal(2)) // at max → last bucket
}
