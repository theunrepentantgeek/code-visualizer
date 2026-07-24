package surface_test

import (
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestSample_RejectsUnderflowingMinimumDistance(t *testing.T) {
	t.Parallel()

	samples := make(chan []surface.Point, 1)
	go func() {
		samples <- surface.Sample(
			surface.Rect{MinX: 0, MinY: 0, MaxX: 50, MaxY: 50},
			nil,
			1e-200,
			42,
		)
	}()

	select {
	case result := <-samples:
		gomega.NewWithT(t).Expect(result).To(gomega.BeNil())
	case <-time.After(time.Second):
		t.Fatal("Sample did not return for an underflowing minimum distance")
	}
}

func TestSample_RespectsMinimumDistance(t *testing.T) {
	t.Parallel()

	g := gomega.NewGomegaWithT(t)
	originals := []surface.Point{
		{X: 10, Y: 10, Value: 1, Original: true},
		{X: 40, Y: 40, Value: 2, Original: true},
	}

	samples := surface.Sample(
		surface.Rect{MinX: 0, MinY: 0, MaxX: 50, MaxY: 50},
		originals,
		surface.PoissonMinDistance,
		42,
	)

	g.Expect(samples).NotTo(gomega.BeEmpty())

	for _, sample := range samples {
		g.Expect(sample.Original).To(gomega.BeFalse())

		for _, original := range originals {
			g.Expect(surface.Distance(sample, original)).To(
				gomega.BeNumerically(">=", surface.PoissonMinDistance),
			)
		}
	}

	for i, sample := range samples {
		for _, other := range samples[i+1:] {
			g.Expect(surface.Distance(sample, other)).To(
				gomega.BeNumerically(">=", surface.PoissonMinDistance),
			)
		}
	}
}

func TestSample_ReturnsOnlyPointsInsideAnnulus(t *testing.T) {
	t.Parallel()

	g := gomega.NewGomegaWithT(t)
	region := surface.Annulus{
		CX:          25,
		CY:          25,
		InnerRadius: 8,
		OuterRadius: 20,
	}

	samples := surface.Sample(region, nil, surface.PoissonMinDistance, 42)

	g.Expect(samples).NotTo(gomega.BeEmpty())

	for _, sample := range samples {
		g.Expect(region.Contains(sample.X, sample.Y)).To(gomega.BeTrue())
	}
}

func TestSample_IsDeterministicForSeed(t *testing.T) {
	t.Parallel()

	g := gomega.NewGomegaWithT(t)
	region := surface.Annulus{
		CX:          30,
		CY:          30,
		InnerRadius: 6,
		OuterRadius: 25,
	}
	originals := []surface.Point{{X: 30, Y: 10, Value: 3, Original: true}}

	first := surface.Sample(region, originals, surface.PoissonMinDistance, 123)
	second := surface.Sample(region, originals, surface.PoissonMinDistance, 123)

	g.Expect(first).To(gomega.Equal(second))
}
