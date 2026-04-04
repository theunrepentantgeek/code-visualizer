package metric

// BucketBoundaries holds quantile-based breakpoints for mapping metric values to palette steps.
type BucketBoundaries struct {
	Boundaries []float64
	Min        float64
	Max        float64
	StepCount  int
}
