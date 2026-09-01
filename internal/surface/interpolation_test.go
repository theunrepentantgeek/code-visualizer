package surface

import (
	"math"
	"testing"

	"github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestSmootherstepWeight_KnownValues(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	g.Expect(smootherstepWeight(0)).To(gomega.Equal(1.0))
	g.Expect(smootherstepWeight(0.5)).To(gomega.Equal(0.5))
	g.Expect(smootherstepWeight(1)).To(gomega.Equal(0.0))
	g.Expect(smootherstepWeight(math.NaN())).To(gomega.Equal(0.0))
	g.Expect(smootherstepWeight(math.Inf(1))).To(gomega.Equal(0.0))
}

func TestSmootherstepWeight_IsFlatAtEndpoints(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	const epsilon = 1e-4

	g.Expect(math.Abs(1 - smootherstepWeight(epsilon))).To(gomega.BeNumerically("<", 1e-9))
	g.Expect(math.Abs(smootherstepWeight(1 - epsilon))).To(gomega.BeNumerically("<", 1e-9))
}

func TestInterpolationSupportRadius_UsesNinetiethPercentileNearestPositiveDistance(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	observations := make([]Sample, 0, 20)

	for gap := range 10 {
		base := float64((gap + 1) * 100)
		delta := float64(gap + 1)
		observations = append(
			observations,
			Sample{Position: geometry.Point{X: base, Y: 0}, Value: base},
			Sample{Position: geometry.Point{X: base + delta, Y: 0}, Value: base + delta},
		)
	}

	radius, ok := interpolationSupportRadius(observations)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(radius).To(gomega.Equal(18.0))
}

func TestNewInterpolationModel_FiltersNonFiniteAndKeepsCoincidentPoints(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model, ok := newInterpolationModel([]Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 1},
		{Position: geometry.Point{X: 0, Y: 0}, Value: 2},
		{Position: geometry.Point{X: 4, Y: 0}, Value: 3},
		{Position: geometry.Point{X: math.NaN(), Y: 1}, Value: 4},
		{Position: geometry.Point{X: 1, Y: 1}, Value: math.Inf(1)},
	})

	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(model.observations).To(gomega.HaveLen(3))
	g.Expect(model.radius).To(gomega.Equal(8.0))

	for _, observation := range model.observations {
		g.Expect(observation.Original).To(gomega.BeTrue())
	}
}

func TestNewInterpolationModel_RejectsCoincidentOnlyObservations(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	_, ok := newInterpolationModel([]Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 1},
		{Position: geometry.Point{X: 0, Y: 0}, Value: 2},
	})

	g.Expect(ok).To(gomega.BeFalse())
}

func TestInterpolationModelInterpolate_AssignsZeroWeightAtSupportRadiusBoundary(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model := interpolationModel{
		observations: []Sample{{Position: geometry.Point{X: 0, Y: 0}, Value: 10}},
		radius:       4,
	}

	inside, ok := model.interpolate(geometry.Point{X: 3, Y: 0})
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(inside).To(gomega.Equal(10.0))

	boundary, ok := model.interpolate(geometry.Point{X: 4, Y: 0})
	g.Expect(ok).To(gomega.BeFalse())
	g.Expect(boundary).To(gomega.Equal(0.0))

	outside, ok := model.interpolate(geometry.Point{X: 7, Y: 0})
	g.Expect(ok).To(gomega.BeFalse())
	g.Expect(outside).To(gomega.Equal(0.0))
}

func TestInterpolationModelInterpolate_ReturnsUnsupportedWithoutPositiveWeights(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model := interpolationModel{
		observations: []Sample{{Position: geometry.Point{X: 0, Y: 0}, Value: 1}},
		radius:       1,
	}

	value, ok := model.interpolate(geometry.Point{X: 2, Y: 0})
	g.Expect(ok).To(gomega.BeFalse())
	g.Expect(value).To(gomega.Equal(0.0))
}

func TestInterpolationModelInterpolate_ReturnsFirstCoincidentObservationValue(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model := interpolationModel{
		observations: []Sample{
			{Position: geometry.Point{X: 1, Y: 1}, Value: 10},
			{Position: geometry.Point{X: 1, Y: 1}, Value: 20},
			{Position: geometry.Point{X: 3, Y: 1}, Value: 30},
		},
		radius: 4,
	}

	value, ok := model.interpolate(geometry.Point{X: 1, Y: 1})
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(value).To(gomega.Equal(10.0))
}

func TestInterpolate_ReturnsZeroForUnsupportedInterpolation(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	value := Interpolate(geometry.Point{X: 5, Y: 0}, []Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 1},
		{Position: geometry.Point{X: 0, Y: 0}, Value: 2},
	})

	g.Expect(value).To(gomega.Equal(0.0))
}

func TestInterpolationModelAssign_SetsUnsupportedFromSupportState(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model := interpolationModel{
		observations: []Sample{
			{Position: geometry.Point{X: 0, Y: 0}, Value: 0},
			{Position: geometry.Point{X: 2, Y: 0}, Value: 0},
		},
		radius: 2,
	}

	supported := model.assign(Sample{Position: geometry.Point{X: 1, Y: 0}})
	g.Expect(supported.Value).To(gomega.Equal(0.0))
	g.Expect(supported.unsupported).To(gomega.BeFalse())

	unsupported := model.assign(Sample{Position: geometry.Point{X: 4, Y: 0}})
	g.Expect(unsupported.Value).To(gomega.Equal(0.0))
	g.Expect(unsupported.unsupported).To(gomega.BeTrue())
}
