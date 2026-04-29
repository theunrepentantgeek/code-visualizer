package spiral

import (
	"math"
)

const (
	// margin is the padding between the spiral and canvas edges.
	margin = 40.0
	// defaultDiscRadius is a small fixed radius used when disc size is not set.
	defaultDiscRadius = 4.0
	// innerRadiusFraction controls the inner/outer radius ratio (~1:3).
	innerRadiusFraction = 1.0 / 3.0
)

// Layout positions time buckets along an Archimedean spiral.
//
// The spiral is clockwise from 12-o'clock (north), starting at the centre and
// expanding outward. The inner radius is approximately 1/3 of the outer radius.
//
// Disc sizes are NOT computed here — the CLI layer sets them from the size metric.
func Layout(
	buckets []TimeBucket,
	width int,
	height int,
	resolution Resolution,
	labels LabelMode,
) []SpiralNode {
	if len(buckets) == 0 {
		return []SpiralNode{}
	}

	nodes := make([]SpiralNode, len(buckets))
	params := computeSpiralParams(len(buckets), width, height, resolution)

	for i, b := range buckets {
		nodes[i] = positionNode(i, b, params, resolution, labels)
	}

	return nodes
}

// spiralParams holds precomputed constants for the Archimedean spiral.
type spiralParams struct {
	centreX     float64 // canvas centre X
	centreY     float64 // canvas centre Y
	a           float64 // innerRadius (starting radius)
	b           float64 // radial growth per radian
	spotsPerLap int
	maxDisc     float64 // maximum disc radius before overlap
}

// computeSpiralParams derives spiral geometry from canvas dimensions and bucket count.
func computeSpiralParams(n, width, height int, resolution Resolution) spiralParams {
	spotsPerLap := resolution.SpotsPerLap()
	canvasRadius := math.Min(float64(width), float64(height))/2 - margin
	outerRadius := canvasRadius
	innerRadius := outerRadius * innerRadiusFraction

	totalAngle := computeTotalAngle(n, spotsPerLap)

	var b float64
	if totalAngle > 0 {
		b = (outerRadius - innerRadius) / totalAngle
	}

	maxDisc := computeMaxDisc(innerRadius, outerRadius, spotsPerLap, totalAngle)

	return spiralParams{
		centreX:     float64(width) / 2,
		centreY:     float64(height) / 2,
		a:           innerRadius,
		b:           b,
		spotsPerLap: spotsPerLap,
		maxDisc:     maxDisc,
	}
}

// computeTotalAngle returns the total angle swept by all buckets.
func computeTotalAngle(n, spotsPerLap int) float64 {
	if n <= 1 {
		return 0
	}

	return float64(n-1) * (2 * math.Pi / float64(spotsPerLap))
}

// computeMaxDisc calculates the maximum disc radius that avoids overlap.
func computeMaxDisc(innerRadius, outerRadius float64, spotsPerLap int, totalAngle float64) float64 {
	angularStep := 2 * math.Pi / float64(spotsPerLap)
	gapAngular := innerRadius * angularStep // arc at inner radius (worst case)

	var gapRadial float64
	if totalAngle > 0 {
		gapRadial = (outerRadius - innerRadius) / (totalAngle / (2 * math.Pi))
	} else {
		gapRadial = outerRadius - innerRadius
	}

	maxR := math.Min(gapAngular, gapRadial) / 2
	if maxR < defaultDiscRadius {
		maxR = defaultDiscRadius
	}

	return maxR
}

// positionNode places bucket i on the spiral and assigns label visibility.
func positionNode(
	i int,
	bucket TimeBucket,
	params spiralParams,
	resolution Resolution,
	labels LabelMode,
) SpiralNode {
	theta := float64(i) * (2 * math.Pi / float64(params.spotsPerLap))
	r := params.a + params.b*theta

	// Clockwise from north: x = cx + r*sin(θ), y = cy - r*cos(θ)
	x := params.centreX + r*math.Sin(theta)
	y := params.centreY - r*math.Cos(theta)

	showLabel := computeLabelVisibility(i, params.spotsPerLap, labels)
	label := formatBucketLabel(bucket, resolution)

	return SpiralNode{
		X:            x,
		Y:            y,
		DiscRadius:   defaultDiscRadius,
		Angle:        theta,
		SpiralRadius: r,
		TimeStart:    bucket.Start,
		TimeEnd:      bucket.End,
		Label:        label,
		ShowLabel:    showLabel,
	}
}

// computeLabelVisibility determines whether a node at index i should show its label.
func computeLabelVisibility(i, spotsPerLap int, labels LabelMode) bool {
	switch labels {
	case LabelAll:
		return true
	case LabelLaps:
		return i%spotsPerLap == 0
	default:
		return false
	}
}

// formatBucketLabel generates a human-readable label for a time bucket.
func formatBucketLabel(bucket TimeBucket, resolution Resolution) string {
	switch resolution {
	case Hourly:
		return bucket.Start.Format("3pm")
	default:
		return bucket.Start.Format("Jan 2")
	}
}
