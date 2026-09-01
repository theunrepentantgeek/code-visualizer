package surface

import (
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

const (
	supportRadiusPercentile = 0.90
	supportRadiusMultiplier = 2.0
)

type interpolationModel struct {
	observations []Sample
	radius       float64
}

// Interpolate estimates a point's value from observed samples with compact support.
func Interpolate(point geometry.Point, originals []Sample) float64 {
	model, ok := newInterpolationModel(originals)
	if !ok {
		return 0
	}

	value, ok := model.interpolate(point)
	if !ok {
		return 0
	}

	return value
}

func newInterpolationModel(originals []Sample) (interpolationModel, bool) {
	observations := observedSamples(originals)
	if len(observations) == 0 {
		return interpolationModel{}, false
	}

	radius, ok := interpolationSupportRadius(observations)
	if !ok {
		return interpolationModel{}, false
	}

	return interpolationModel{observations: observations, radius: radius}, true
}

func interpolationSupportRadius(observations []Sample) (float64, bool) {
	nearestPositiveDistances := make([]float64, 0, len(observations))

	for index, observation := range observations {
		if !isFiniteSample(observation) || !isFinite(observation.Value) {
			continue
		}

		if nearestDistance, found := nearestPositiveDistance(observations, index); found {
			nearestPositiveDistances = append(nearestPositiveDistances, nearestDistance)
		}
	}

	if len(nearestPositiveDistances) == 0 {
		return 0, false
	}

	slices.Sort(nearestPositiveDistances)

	index := int(math.Ceil(supportRadiusPercentile*float64(len(nearestPositiveDistances)))) - 1

	radius := nearestPositiveDistances[index] * supportRadiusMultiplier
	if !isFinite(radius) || radius <= 0 {
		return 0, false
	}

	return radius, true
}

func nearestPositiveDistance(observations []Sample, observationIndex int) (float64, bool) {
	nearestDistance := math.Inf(1)

	for otherIndex, other := range observations {
		if otherIndex == observationIndex {
			continue
		}

		distance := observations[observationIndex].Position.DistanceTo(other.Position)
		if isFinite(distance) && distance > 0 {
			nearestDistance = min(nearestDistance, distance)
		}
	}

	return nearestDistance, isFinite(nearestDistance)
}

func smootherstepWeight(t float64) float64 {
	if !isFinite(t) || t >= 1 {
		return 0
	}

	if t <= 0 {
		return 1
	}

	return 1 - t*t*t*(t*(6*t-15)+10)
}

func (model interpolationModel) interpolate(point geometry.Point) (float64, bool) {
	if !point.Valid() || !isFinite(model.radius) || model.radius <= 0 {
		return 0, false
	}

	if value, found := observedValueAt(point, model.observations); found {
		return value, true
	}

	return model.weightedValue(point)
}

func observedValueAt(point geometry.Point, observations []Sample) (float64, bool) {
	for _, observation := range observations {
		if point == observation.Position {
			return observation.Value, true
		}
	}

	return 0, false
}

func (model interpolationModel) weightedValue(point geometry.Point) (float64, bool) {
	var (
		weightedValue float64
		totalWeight   float64
	)

	for _, observation := range model.observations {
		distance := point.DistanceTo(observation.Position)
		if !isFinite(distance) {
			continue
		}

		weight := smootherstepWeight(distance / model.radius)
		if weight <= 0 {
			continue
		}

		weightedValue += observation.Value * weight
		totalWeight += weight
	}

	if totalWeight <= 0 || !isFinite(totalWeight) || !isFinite(weightedValue) {
		return 0, false
	}

	value := weightedValue / totalWeight
	if !isFinite(value) {
		return 0, false
	}

	return value, true
}

func (model interpolationModel) assign(sample Sample) Sample {
	value, supported := model.interpolate(sample.Position)
	sample.Value = value
	sample.unsupported = !supported

	return sample
}
