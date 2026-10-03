package config

import "slices"

// overrideString sets *dst to a copy of v if v is non-empty.
// Used by OverrideX methods to conditionally apply CLI flag values.
func overrideString(dst **string, v string) {
	if v != "" {
		*dst = &v
	}
}

// overrideInt sets *dst to a copy of v if v is non-zero.
func overrideInt(dst **int, v int) {
	if v != 0 {
		*dst = &v
	}
}

// overrideMetricSpec sets *dst to a copy of v if v is non-zero.
func overrideMetricSpec(dst **MetricSpec, v MetricSpec) {
	if !v.IsZero() {
		*dst = &v
	}
}

func overrideSlice[T any](dst *[]T, values []T) {
	if len(values) != 0 {
		*dst = slices.Clone(values)
	}
}
