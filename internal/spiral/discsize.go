package spiral

import "math"

// minDiscRadius is the minimum visible disc radius for active time buckets.
const minDiscRadius = 12.0

// ApplyDiscSizes sets disc radii on nodes proportional to their bucket
// SizeValue. Empty buckets get zero radius (not drawn). Active buckets are
// scaled between minDiscRadius and the readable maximum.
func ApplyDiscSizes(nodes []SpiralNode, buckets []TimeBucket, maxDisc float64) {
	maxSize := 0.0
	cappedMax := max(maxDisc, minDiscRadius)

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
		nodes[i].DiscRadius = minDiscRadius + (cappedMax-minDiscRadius)*math.Sqrt(ratio)
	}
}
