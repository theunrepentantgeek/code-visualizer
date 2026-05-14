package metric

import (
	"math"
	"slices"
)

// BucketBoundaries holds quantile-based breakpoints for mapping metric values to palette steps.
type BucketBoundaries struct {
	Boundaries []float64
	Min        float64
	Max        float64
	StepCount  int
}

// ComputeBuckets computes quantile-based bucket boundaries for the given values.
// Boundaries are rounded to 2 significant figures and deduplicated.
func ComputeBuckets(values []float64, steps int) BucketBoundaries {
	if len(values) == 0 {
		return BucketBoundaries{StepCount: steps}
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	slices.Sort(sorted)

	minVal := sorted[0]
	maxVal := sorted[len(sorted)-1]

	if steps <= 1 || minVal == maxVal {
		return BucketBoundaries{
			Min:       minVal,
			Max:       maxVal,
			StepCount: steps,
		}
	}

	// Compute N-1 quantile breakpoints for N steps
	raw := make([]float64, 0, steps-1)
	for i := 1; i < steps; i++ {
		idx := i * len(sorted) / steps
		idx = min(idx, len(sorted)-1) // ensure idx is within bounds
		raw = append(raw, roundToSigFigs(sorted[idx], 2))
	}

	// Deduplicate and filter out any boundary at or below minVal.
	// A boundary at minVal would cause the minimum value to map to bucket 1
	// (since value < boundary is false when value == boundary), leaving
	// bucket 0 permanently empty and wasting that palette entry.
	boundaries := slices.Compact(raw)
	if len(boundaries) == 0 || boundaries[0] <= minVal {
		boundaries = boundaries[1:]
	}

	return BucketBoundaries{
		Boundaries: boundaries,
		Min:        minVal,
		Max:        maxVal,
		StepCount:  steps,
	}
}

// NumBuckets returns the total number of buckets (len(Boundaries) + 1).
func (b BucketBoundaries) NumBuckets() int {
	return len(b.Boundaries) + 1
}

// BucketIndex returns the bucket index (0-based) for the given value.
func (b BucketBoundaries) BucketIndex(value float64) int {
	for i, boundary := range b.Boundaries {
		if value < boundary {
			return i
		}
	}

	return len(b.Boundaries)
}

// roundToSigFigs rounds a value to n significant figures.
func roundToSigFigs(v float64, n int) float64 {
	if v == 0 {
		return 0
	}

	d := math.Ceil(math.Log10(math.Abs(v)))
	pow := math.Pow(10, float64(n)-d)

	return math.Round(v*pow) / pow
}
