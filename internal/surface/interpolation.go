package surface

import (
	"math"
	"sort"
)

const (
	supportRadiusPercentile = 0.90
	supportRadiusMultiplier = 2.0
)

type interpolationModel struct {
	observations []Point
	radius       float64
}

// Interpolate estimates a point's value from observed points with compact support.
func Interpolate(point Point, originals []Point) float64 {
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

func newInterpolationModel(originals []Point) (interpolationModel, bool) {
	observations := observedPoints(originals)
	if len(observations) == 0 {
		return interpolationModel{}, false
	}

	radius, ok := interpolationSupportRadius(observations)
	if !ok {
		return interpolationModel{}, false
	}

	return interpolationModel{observations: observations, radius: radius}, true
}

func interpolationSupportRadius(observations []Point) (float64, bool) {
	nearestPositiveDistances := make([]float64, 0, len(observations))

	for index, observation := range observations {
		if !isFinitePoint(observation) || !isFinite(observation.Value) {
			continue
		}

		nearestDistance := math.Inf(1)
		for otherIndex, other := range observations {
			if otherIndex == index {
				continue
			}

			distance := Distance(observation, other)
			if !isFinite(distance) || distance <= 0 {
				continue
			}

			if distance < nearestDistance {
				nearestDistance = distance
			}
		}

		if isFinite(nearestDistance) && nearestDistance > 0 {
			nearestPositiveDistances = append(nearestPositiveDistances, nearestDistance)
		}
	}

	if len(nearestPositiveDistances) == 0 {
		return 0, false
	}

	sort.Float64s(nearestPositiveDistances)
	index := int(math.Ceil(supportRadiusPercentile*float64(len(nearestPositiveDistances)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(nearestPositiveDistances) {
		index = len(nearestPositiveDistances) - 1
	}

	radius := nearestPositiveDistances[index] * supportRadiusMultiplier
	if !isFinite(radius) || radius <= 0 {
		return 0, false
	}

	return radius, true
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

func (model interpolationModel) interpolate(point Point) (float64, bool) {
	if !isFinitePoint(point) || !isFinite(model.radius) || model.radius <= 0 {
		return 0, false
	}

	for _, observation := range model.observations {
		if point.X == observation.X && point.Y == observation.Y {
			return observation.Value, true
		}
	}

	var (
		weightedValue float64
		totalWeight   float64
	)

	for _, observation := range model.observations {
		distance := Distance(point, observation)
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

	if !isFinite(totalWeight) || totalWeight <= 0 || !isFinite(weightedValue) {
		return 0, false
	}

	value := weightedValue / totalWeight
	if !isFinite(value) {
		return 0, false
	}

	return value, true
}

func (model interpolationModel) assign(point Point) Point {
	value, supported := model.interpolate(point)
	point.Value = value
	point.unsupported = !supported

	return point
}
