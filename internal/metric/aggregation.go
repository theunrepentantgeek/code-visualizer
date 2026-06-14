package metric

import (
	"maps"
	"math"
	"slices"
)

// AggregateSum returns the sum of all values.
func AggregateSum(values []float64) float64 {
	var total float64
	for _, v := range values {
		total += v
	}

	return total
}

// AggregateMin returns the minimum value, or 0 if empty.
func AggregateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	result := math.MaxFloat64
	for _, v := range values {
		if v < result {
			result = v
		}
	}

	return result
}

// AggregateMax returns the maximum value, or 0 if empty.
func AggregateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	result := -math.MaxFloat64
	for _, v := range values {
		if v > result {
			result = v
		}
	}

	return result
}

// AggregateMean returns the arithmetic mean, or 0 if empty.
func AggregateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	return AggregateSum(values) / float64(len(values))
}

// AggregateCount returns the number of values.
func AggregateCount(values []float64) float64 {
	return float64(len(values))
}

// AggregateRange returns max − min, or 0 if fewer than 2 values.
func AggregateRange(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	return AggregateMax(values) - AggregateMin(values)
}

// AggregateMode returns the most common string value.
// On a tie, returns the lexicographically first tied value.
// Returns "" for an empty slice.
func AggregateMode(values []string) string {
	if len(values) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, v := range values {
		counts[v]++
	}

	best := ""
	bestCount := 0

	for _, key := range slices.Sorted(maps.Keys(counts)) {
		if counts[key] > bestCount {
			best = key
			bestCount = counts[key]
		}
	}

	return best
}

// AggregateDistinct returns the number of distinct string values.
func AggregateDistinct(values []string) int {
	if len(values) == 0 {
		return 0
	}

	unique := make(map[string]struct{})
	for _, v := range values {
		unique[v] = struct{}{}
	}

	return len(unique)
}
