package spiral

import "math"

// minDiscRadius is the minimum visible disc radius for active time buckets.
const minDiscRadius = 12.0

// ApplyDiscSizes sets disc radii on nodes proportional to their bucket
// SizeValue. Empty buckets get zero radius (not drawn). Active buckets are
// scaled between the readable floor (when geometry permits) and maxDisc.
func ApplyDiscSizes(nodes []SpiralNode, buckets []TimeBucket, maxDisc float64) {
	maxSize := 0.0
	effectiveMin := min(minDiscRadius, maxDisc)

	for _, b := range buckets {
		if b.SizeValue > maxSize {
			maxSize = b.SizeValue
		}
	}

	for i := range nodes {
		if buckets[i].SizeValue == 0 && len(buckets[i].Files) == 0 {
			nodes[i].Geometry.Radius = 0

			continue
		}

		if maxSize == 0 {
			nodes[i].Geometry.Radius = effectiveMin

			continue
		}

		ratio := buckets[i].SizeValue / maxSize
		nodes[i].Geometry.Radius = min(
			maxDisc,
			max(effectiveMin, effectiveMin+(maxDisc-effectiveMin)*math.Sqrt(ratio)),
		)
	}
}
