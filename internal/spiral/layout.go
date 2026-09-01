package spiral

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

const (
	// margin is the padding between the spiral and canvas edges.
	margin = 40.0
	// defaultDiscRadius is a small fixed radius used when disc size is not set.
	defaultDiscRadius = 4.0
	// innerRadiusFraction controls the inner/outer radius ratio (~1:3).
	innerRadiusFraction = 1.0 / 3.0
	// discRadiusSafetyFactor absorbs floating-point error in adjacent-lap distances.
	discRadiusSafetyFactor = 1 - 1e-12
)

// SpiralLayout holds the positioned nodes and the Archimedean spiral parameters
// used to generate them. Exposing these parameters lets callers draw the guide
// track without re-deriving the geometry from node positions.
type SpiralLayout struct {
	Nodes    []SpiralNode
	CX, CY   float64 // canvas centre
	A, B     float64 // Archimedean parameters: r = A + B*theta
	MaxTheta float64 // angle of the last node
}

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
) SpiralLayout {
	return LayoutWithCadence(buckets, width, height, resolution.SpotsPerLap())
}

// LayoutWithCadence positions time buckets along an Archimedean spiral using
// the supplied spots-per-lap cadence.
func LayoutWithCadence(
	buckets []TimeBucket,
	width int,
	height int,
	spotsPerLap int,
) SpiralLayout {
	if len(buckets) == 0 {
		return SpiralLayout{}
	}

	nodes := make([]SpiralNode, len(buckets))
	params := computeSpiralParams(len(buckets), width, height, spotsPerLap)

	for i, b := range buckets {
		nodes[i] = positionNode(i, b, params)
	}

	var maxTheta float64
	if len(nodes) > 0 {
		maxTheta = nodes[len(nodes)-1].Angle
	}

	return SpiralLayout{
		Nodes:    nodes,
		CX:       params.centreX,
		CY:       params.centreY,
		A:        params.a,
		B:        params.b,
		MaxTheta: maxTheta,
	}
}

// spiralParams holds precomputed constants for the Archimedean spiral.
type spiralParams struct {
	centreX     float64 // canvas centre X
	centreY     float64 // canvas centre Y
	a           float64 // innerRadius (starting radius)
	b           float64 // radial growth per radian
	spotsPerLap int
	totalAngle  float64
	maxDisc     float64 // maximum disc radius before overlap
}

// computeSpiralParams derives spiral geometry from canvas dimensions and bucket count.
func computeSpiralParams(n, width, height, spotsPerLap int) spiralParams {
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
		totalAngle:  totalAngle,
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

// MaxDiscRadius returns the maximum disc radius that avoids overlap for the
// given layout parameters. Use this to clamp disc sizes in the CLI layer.
func MaxDiscRadius(
	bucketCount int,
	width int,
	height int,
	resolution Resolution,
) float64 {
	return MaxDiscRadiusWithCadence(bucketCount, width, height, resolution.SpotsPerLap())
}

// MaxDiscRadiusWithCadence returns the maximum disc radius that avoids overlap
// for the given layout parameters and spots-per-lap cadence.
func MaxDiscRadiusWithCadence(
	bucketCount int,
	width int,
	height int,
	spotsPerLap int,
) float64 {
	if bucketCount == 0 {
		return defaultDiscRadius
	}

	params := computeSpiralParams(bucketCount, width, height, spotsPerLap)

	return params.maxDisc
}

// computeMaxDisc calculates the maximum disc radius that avoids overlap.
func computeMaxDisc(innerRadius, outerRadius float64, spotsPerLap int, totalAngle float64) float64 {
	if totalAngle == 0 {
		maxR := min(innerRadius, outerRadius-innerRadius)
		maxR = max(0, maxR-borderWidth(maxR)/2)

		return maxR * discRadiusSafetyFactor
	}

	angularStep := 2 * math.Pi / float64(spotsPerLap)
	gapAngular := innerRadius * angularStep // arc at inner radius (worst case)

	gapRadial := (outerRadius - innerRadius) / (totalAngle / (2 * math.Pi))

	maxR := math.Min(gapAngular, gapRadial) / 2
	maxR = max(0, maxR-borderWidth(maxR)/2)

	return maxR * discRadiusSafetyFactor
}

// positionNode places bucket i on the spiral.
func positionNode(
	i int,
	bucket TimeBucket,
	params spiralParams,
) SpiralNode {
	theta := float64(i) * (2 * math.Pi / float64(params.spotsPerLap))
	r := params.a + params.b*theta

	// Clockwise from north: x = cx + r*sin(θ), y = cy - r*cos(θ)
	center := geometry.Point{X: params.centreX, Y: params.centreY}

	return SpiralNode{
		Position: center.Translate(geometry.Vector{
			X: r * math.Sin(theta),
			Y: -r * math.Cos(theta),
		}),
		DiscRadius:   defaultDiscRadius,
		Angle:        theta,
		SpiralRadius: r,
		TimeStart:    bucket.Start,
		TimeEnd:      bucket.End,
	}
}
