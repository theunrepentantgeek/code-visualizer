package spiral

import (
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
