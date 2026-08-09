package spiral

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

func TestDailySpotsPerLapReturnsAllowedCadence(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, bucketCount := range []int{1, 14, 28, 100, 365, 730} {
		g.Expect(DailySpotsPerLap(bucketCount, 1920, 1080)).To(
			BeElementOf(14, 28, 42, 56),
			"bucket count %d", bucketCount,
		)
	}
}

func TestDailySpotsPerLapReturns28ForSingleBucket(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(DailySpotsPerLap(1, 1920, 1080)).To(Equal(28))
}

func TestDailySpotsPerLapMinimizesCandidateScore(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, bucketCount := range []int{28, 100, 365, 730} {
		selected := DailySpotsPerLap(bucketCount, 1920, 1080)
		selectedScore := dailyCadenceScore(bucketCount, 1920, 1080, selected)

		for _, candidate := range dailyCadences {
			g.Expect(selectedScore).To(
				BeNumerically("<=", dailyCadenceScore(bucketCount, 1920, 1080, candidate)),
				"bucket count %d, candidate %d", bucketCount, candidate,
			)
		}
	}
}

func TestDailySpotsPerLapReturns28WhenGeometryHasNoRadialGrowth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	width := int(2 * margin)
	const height = 1080

	for _, candidate := range dailyCadences {
		g.Expect(computeSpiralParams(28, width, height, candidate).b).To(BeZero())
		g.Expect(dailyCadenceScore(28, width, height, candidate)).To(
			Equal(math.Inf(1)),
			"candidate %d", candidate,
		)
	}

	g.Expect(DailySpotsPerLap(28, width, height)).To(Equal(28))
}

func TestDailySpotsPerLapReturns28WhenGeometryHasNegativeRadialGrowth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const (
		bucketCount = 28
		width       = 79
		height      = 1080
	)

	for _, candidate := range dailyCadences {
		g.Expect(dailyCadenceScore(bucketCount, width, height, candidate)).To(
			Equal(math.Inf(1)),
			"candidate %d", candidate,
		)
	}

	g.Expect(DailySpotsPerLap(bucketCount, width, height)).To(Equal(28))
}

func TestSelectDailyCadencePrefers28OnEqualScore(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(selectDailyCadence(map[int]float64{
		14: 1,
		28: 1,
		42: 2,
		56: 3,
	})).To(Equal(28))
}

func TestSpotsPerLapKeepsHourlyCadence(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(SpotsPerLap(Hourly, 730, 1920, 1080)).To(Equal(24))
}
