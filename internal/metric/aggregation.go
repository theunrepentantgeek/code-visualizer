package metric

import (
	"math"
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
// The result is computed in a single pass rather than calling AggregateMax
// and AggregateMin separately.
func AggregateRange(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	lo, hi := values[0], values[0]
	for _, v := range values[1:] {
		if v < lo {
			lo = v
		}

		if v > hi {
			hi = v
		}
	}

	return hi - lo
}

// AggregateMode returns the most common string value.
// On a tie, returns the lexicographically first tied value.
// Returns "" for an empty slice.
func AggregateMode(values []string) string {
	if len(values) == 0 {
		return ""
	}

	counts := make(map[string]int, len(values))
	for _, v := range values {
		counts[v]++
	}

	// Find the maximum count in a single pass, then find the
	// lexicographically smallest key that reaches that count.
	// This avoids allocating and sorting a key slice.
	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	best := ""
	for key, c := range counts {
		if c == maxCount && (best == "" || key < best) {
			best = key
		}
	}

	return best
}

// AggregateDistinct returns the number of distinct string values.
func AggregateDistinct(values []string) int {
	if len(values) == 0 {
		return 0
	}

	unique := make(map[string]struct{}, len(values))
	for _, v := range values {
		unique[v] = struct{}{}
	}

	return len(unique)
}
