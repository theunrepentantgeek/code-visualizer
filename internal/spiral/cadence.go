package spiral

import "math"

var dailyCadences = []int{14, 28, 42, 56, 84, 112, 168}

// SpotsPerLap returns the cadence for a layout at the given resolution.
func SpotsPerLap(resolution Resolution, bucketCount, width, height int) int {
	if resolution == Daily {
		return DailySpotsPerLap(bucketCount, width, height)
	}

	return resolution.SpotsPerLap()
}

// DailySpotsPerLap selects the daily cadence with the most even midpoint spacing.
func DailySpotsPerLap(bucketCount, width, height int) int {
	scores := make(map[int]float64, len(dailyCadences))
	for _, spotsPerLap := range dailyCadences {
		scores[spotsPerLap] = dailyCadenceScore(bucketCount, width, height, spotsPerLap)
	}

	return selectDailyCadence(scores)
}

func dailyCadenceScore(bucketCount, width, height, spotsPerLap int) float64 {
	if bucketCount <= 1 {
		return math.Inf(1)
	}

	params := computeSpiralParams(bucketCount, width, height, spotsPerLap)
	if params.b <= 0 {
		return math.Inf(1)
	}

	radialGap := 2 * math.Pi * params.b
	midpointRadius := params.a + params.b*params.totalAngle/2
	arcGap := midpointRadius * (2 * math.Pi / float64(spotsPerLap))

	return math.Abs(radialGap - arcGap)
}

func selectDailyCadence(scores map[int]float64) int {
	selected := 28
	selectedScore := scores[selected]

	for _, candidate := range dailyCadences {
		if scores[candidate] < selectedScore {
			selected = candidate
			selectedScore = scores[candidate]
		}
	}

	return selected
}
