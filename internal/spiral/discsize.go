package spiral

import "math"

// minDiscRadius is the minimum visible disc radius for active time buckets.
const minDiscRadius = 3.0

// ApplyDiscSizes sets disc radii on nodes proportional to their bucket
// SizeValue. Empty buckets get zero radius (not drawn). Active buckets are
// clamped between minDiscRadius and maxDisc.
func ApplyDiscSizes(nodes []SpiralNode, buckets []TimeBucket, maxDisc float64) {
	maxSize := 0.0

	for _, b := range buckets {
		if b.SizeValue > maxSize {
			maxSize = b.SizeValue
		}
	}

	for i := range nodes {
		if buckets[i].SizeValue == 0 && len(buckets[i].Files) == 0 {
			nodes[i].DiscRadius = 0

			continue
		}

		if maxSize == 0 {
			nodes[i].DiscRadius = minDiscRadius

			continue
		}

		ratio := buckets[i].SizeValue / maxSize
		scaled := nodes[i].DiscRadius * math.Sqrt(ratio)
		nodes[i].DiscRadius = max(minDiscRadius, min(scaled, maxDisc))
	}
}
