package spiral

import (
	"math"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// referenceTime is a fixed anchor for test bucket construction.
var referenceTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func makeBuckets(n int, resolution Resolution) []TimeBucket {
	dur := resolution.bucketDuration()
	buckets := make([]TimeBucket, n)

	for i := range n {
		buckets[i] = TimeBucket{
			Start: referenceTime.Add(time.Duration(i) * dur),
			End:   referenceTime.Add(time.Duration(i+1) * dur),
		}
	}

	return buckets
}

func TestLayoutNodeCountMatchesBuckets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for _, n := range []int{1, 10, 24, 28, 50, 100} {
		buckets := makeBuckets(n, Hourly)
		nodes := Layout(buckets, 1920, 1920, Hourly, LabelAll)
		g.Expect(nodes).To(HaveLen(n), "expected %d nodes", n)
	}
}

func TestLayoutZeroBucketsReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := Layout(nil, 1920, 1920, Hourly, LabelAll)
	g.Expect(nodes).To(BeEmpty())

	nodes = Layout([]TimeBucket{}, 1920, 1920, Daily, LabelNone)
	g.Expect(nodes).To(BeEmpty())
}

func TestLayoutSingleBucket(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(1, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelAll)
	g.Expect(nodes).To(HaveLen(1))

	// Single bucket should be at the centre of the spiral (inner radius, angle 0).
	n := nodes[0]
	g.Expect(n.Angle).To(BeNumerically("==", 0))
	g.Expect(n.DiscRadius).To(BeNumerically(">", 0))
}

func TestLayoutRadiusIncreasesMonotonically(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(72, Hourly) // 3 laps
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(72))

	for i := 1; i < len(nodes); i++ {
		g.Expect(nodes[i].SpiralRadius).To(
			BeNumerically(">=", nodes[i-1].SpiralRadius),
			"radius at index %d should be >= radius at index %d", i, i-1,
		)
	}
}

func TestLayoutInnerOuterRatio(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(72, Hourly) // 3 full laps
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)

	innerR := nodes[0].SpiralRadius
	outerR := nodes[len(nodes)-1].SpiralRadius

	// Ratio should be approximately 1:3.
	ratio := outerR / innerR
	g.Expect(ratio).To(BeNumerically("~", 3.0, 0.1),
		"outer/inner ratio should be ~3.0, got %f", ratio)
}

func TestLayoutHourlySpotsPerLap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Hourly.SpotsPerLap()).To(Equal(24))

	buckets := makeBuckets(48, Hourly) // 2 full laps
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(48))

	// After 24 spots, the angle should wrap past 2π.
	g.Expect(nodes[24].Angle).To(
		BeNumerically("~", 2*math.Pi, 0.01),
		"spot 24 should be at ~2π radians",
	)
}

func TestLayoutDailySpotsPerLap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Daily.SpotsPerLap()).To(Equal(28))

	buckets := makeBuckets(56, Daily) // 2 full laps
	nodes := Layout(buckets, 1920, 1920, Daily, LabelNone)
	g.Expect(nodes).To(HaveLen(56))

	// After 28 spots, the angle should wrap past 2π.
	g.Expect(nodes[28].Angle).To(
		BeNumerically("~", 2*math.Pi, 0.01),
		"spot 28 should be at ~2π radians",
	)
}

func TestLayoutUniformAngularSpacing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(48, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)

	expectedStep := 2 * math.Pi / 24.0

	for i := 1; i < len(nodes); i++ {
		gap := nodes[i].Angle - nodes[i-1].Angle
		g.Expect(gap).To(
			BeNumerically("~", expectedStep, 0.001),
			"angular gap between spots %d and %d should be uniform", i-1, i,
		)
	}
}

func TestLayoutExactlyOneLap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(24, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(24))

	// Last spot should be just under one full revolution.
	lastAngle := nodes[23].Angle
	expectedAngle := 23 * (2 * math.Pi / 24.0)
	g.Expect(lastAngle).To(BeNumerically("~", expectedAngle, 0.001))
}

func TestLayoutPartialLastLap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(30, Hourly) // 1 full lap + 6 extra
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(30))

	// Radius should still increase through the partial lap.
	for i := 25; i < 30; i++ {
		g.Expect(nodes[i].SpiralRadius).To(
			BeNumerically(">", nodes[i-1].SpiralRadius),
			"radius should still increase in partial lap at index %d", i,
		)
	}
}

func TestLayoutClockwiseFromNorth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(4, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)

	cx := float64(1920) / 2
	cy := float64(1920) / 2

	// First node (θ=0, north): should be above centre.
	g.Expect(nodes[0].Y).To(BeNumerically("<", cy), "first node should be above centre")
	g.Expect(nodes[0].X).To(BeNumerically("~", cx, 1.0), "first node should be near centre X")
}

func TestLayoutLabelAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(10, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelAll)

	for i, n := range nodes {
		g.Expect(n.ShowLabel).To(BeTrue(), "node %d should have label visible", i)
		g.Expect(n.Label).NotTo(BeEmpty(), "node %d should have a label", i)
	}
}

func TestLayoutLabelLaps(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(50, Hourly) // 2+ laps
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelLaps)

	for i, n := range nodes {
		if i%24 == 0 {
			g.Expect(n.ShowLabel).To(BeTrue(), "lap boundary at index %d should be labelled", i)
		} else {
			g.Expect(n.ShowLabel).To(BeFalse(), "non-boundary at index %d should not be labelled", i)
		}
	}
}

func TestLayoutLabelNone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(10, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)

	for i, n := range nodes {
		g.Expect(n.ShowLabel).To(BeFalse(), "node %d should not have label visible", i)
	}
}

func TestLayoutPositionsWithinCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(100, Hourly)
	nodes := Layout(buckets, 1920, 1080, Hourly, LabelNone)

	for i, n := range nodes {
		g.Expect(n.X).To(BeNumerically(">=", 0), "node %d X should be >= 0", i)
		g.Expect(n.X).To(BeNumerically("<=", 1920), "node %d X should be <= width", i)
		g.Expect(n.Y).To(BeNumerically(">=", 0), "node %d Y should be >= 0", i)
		g.Expect(n.Y).To(BeNumerically("<=", 1080), "node %d Y should be <= height", i)
	}
}

func TestLayoutTimeFieldsPreserved(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(5, Daily)
	nodes := Layout(buckets, 1920, 1920, Daily, LabelAll)

	for i, n := range nodes {
		g.Expect(n.TimeStart).To(Equal(buckets[i].Start), "node %d TimeStart should match bucket", i)
		g.Expect(n.TimeEnd).To(Equal(buckets[i].End), "node %d TimeEnd should match bucket", i)
	}
}

func TestLayoutRectangularCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(48, Hourly)
	nodes := Layout(buckets, 1920, 1080, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(48))

	// Spiral should fit within the smaller dimension.
	for i := 1; i < len(nodes); i++ {
		g.Expect(nodes[i].SpiralRadius).To(
			BeNumerically(">=", nodes[i-1].SpiralRadius),
			"monotonic radius on rectangular canvas at index %d", i,
		)
	}
}

func TestLayoutDailyLabelsFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(3, Daily)
	nodes := Layout(buckets, 1920, 1920, Daily, LabelAll)

	g.Expect(nodes[0].Label).To(Equal("Jan 1"))
	g.Expect(nodes[1].Label).To(Equal("Jan 2"))
	g.Expect(nodes[2].Label).To(Equal("Jan 3"))
}

func TestLayoutHourlyLabelsFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(3, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelAll)

	g.Expect(nodes[0].Label).To(Equal("12am"))
	g.Expect(nodes[1].Label).To(Equal("1am"))
	g.Expect(nodes[2].Label).To(Equal("2am"))
}

// --- Gap tests added by Lambert (Phase 4, issue #127) ---

func TestLayoutCentreOfSpiral(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(72, Hourly) // 3 laps
	nodes := Layout(buckets, 1920, 1080, Hourly, LabelNone)

	cx := float64(1920) / 2
	cy := float64(1080) / 2

	// First node at θ=0 should be directly above centre (north).
	// Its X should be at cx, and Y should be < cy (above centre).
	g.Expect(nodes[0].X).To(BeNumerically("~", cx, 1.0),
		"first node X should be at canvas centre X")
	g.Expect(nodes[0].Y).To(BeNumerically("<", cy),
		"first node Y should be above canvas centre")

	// Average of all X positions should be roughly centred.
	var sumX, sumY float64
	for _, n := range nodes {
		sumX += n.X
		sumY += n.Y
	}

	avgX := sumX / float64(len(nodes))
	avgY := sumY / float64(len(nodes))

	g.Expect(avgX).To(BeNumerically("~", cx, cx*0.15),
		"average X should be near canvas centre")
	g.Expect(avgY).To(BeNumerically("~", cy, cy*0.15),
		"average Y should be near canvas centre")
}

func TestLayoutScalesWithCanvasSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(48, Hourly) // 2 laps
	smallNodes := Layout(buckets, 800, 800, Hourly, LabelNone)
	largeNodes := Layout(buckets, 1600, 1600, Hourly, LabelNone)

	smallOuter := smallNodes[len(smallNodes)-1].SpiralRadius
	largeOuter := largeNodes[len(largeNodes)-1].SpiralRadius

	// Larger canvas should produce proportionally larger spiral.
	// The fixed margin (40px) means the ratio won't be exactly 2.0.
	ratio := largeOuter / smallOuter
	g.Expect(ratio).To(BeNumerically("~", 2.0, 0.15),
		"doubling canvas should roughly double outer radius, got ratio %f", ratio)
}

func TestLayoutDailyUniformAngularSpacing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(56, Daily) // 2 full laps
	nodes := Layout(buckets, 1920, 1920, Daily, LabelNone)

	expectedStep := 2 * math.Pi / 28.0

	for i := 1; i < len(nodes); i++ {
		gap := nodes[i].Angle - nodes[i-1].Angle
		g.Expect(gap).To(
			BeNumerically("~", expectedStep, 0.001),
			"daily angular gap between spots %d and %d should be uniform", i-1, i,
		)
	}
}

func TestLayoutExactlyOneLapDaily(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(28, Daily)
	nodes := Layout(buckets, 1920, 1920, Daily, LabelNone)
	g.Expect(nodes).To(HaveLen(28))

	// Last spot should be just under one full revolution.
	lastAngle := nodes[27].Angle
	expectedAngle := 27 * (2 * math.Pi / 28.0)
	g.Expect(lastAngle).To(BeNumerically("~", expectedAngle, 0.001))
}

func TestLayoutPartialLapDaily(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(7, Daily) // 1/4 of a lap
	nodes := Layout(buckets, 1920, 1920, Daily, LabelNone)
	g.Expect(nodes).To(HaveLen(7))

	// Angular span should be approximately 7 × 2π/28 = π/2.
	lastAngle := nodes[6].Angle
	expectedAngle := 6 * (2 * math.Pi / 28.0)
	g.Expect(lastAngle).To(BeNumerically("~", expectedAngle, 0.001))

	// Radius should still increase across partial lap.
	for i := 1; i < len(nodes); i++ {
		g.Expect(nodes[i].SpiralRadius).To(
			BeNumerically(">=", nodes[i-1].SpiralRadius),
			"radius should increase in partial daily lap at index %d", i,
		)
	}
}

func TestLayoutArchimedeanProperty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(48, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)

	// Archimedean spiral: r = a + b*θ.
	// All nodes should satisfy this equation with the same a and b.
	// Derive a and b from first two nodes.
	a := nodes[0].SpiralRadius
	if nodes[1].Angle > 0 {
		b := (nodes[1].SpiralRadius - a) / nodes[1].Angle

		for i := 2; i < len(nodes); i++ {
			expected := a + b*nodes[i].Angle
			g.Expect(nodes[i].SpiralRadius).To(
				BeNumerically("~", expected, 0.01),
				"node %d should satisfy r = a + b*θ", i,
			)
		}
	}
}

func TestLayoutManyLaps(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(100, Hourly) // 4+ laps
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(100))

	// Monotonically increasing radius across all spots.
	for i := 1; i < len(nodes); i++ {
		g.Expect(nodes[i].SpiralRadius).To(
			BeNumerically(">=", nodes[i-1].SpiralRadius),
			"radius at index %d should be >= index %d", i, i-1,
		)
	}

	// Last spot should be considerably farther from centre than first.
	g.Expect(nodes[99].SpiralRadius).To(
		BeNumerically(">", nodes[0].SpiralRadius*2),
		"100 spots should span well beyond 2x the inner radius",
	)
}

func TestLayoutManyLapsNoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	spotsPerLap := 24
	buckets := makeBuckets(3*spotsPerLap, Hourly) // 3 full laps
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)

	// For spots at the same angular position on adjacent laps, the radial gap
	// must be greater than the sum of their disc radii (no overlap).
	for i := range len(nodes) - spotsPerLap {
		current := nodes[i]
		nextLap := nodes[i+spotsPerLap]
		radialGap := nextLap.SpiralRadius - current.SpiralRadius
		minGap := current.DiscRadius + nextLap.DiscRadius

		g.Expect(radialGap).To(
			BeNumerically(">=", minGap),
			"spots %d and %d (same angle, adjacent laps) should not overlap: gap %f < minGap %f",
			i, i+spotsPerLap, radialGap, minGap,
		)
	}
}

func TestLayoutFitsWithinCanvasIncludingDisc(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(100, Hourly)
	nodes := Layout(buckets, 1920, 1080, Hourly, LabelNone)

	for i, n := range nodes {
		g.Expect(n.X-n.DiscRadius).To(BeNumerically(">=", 0),
			"node %d left edge should be >= 0", i)
		g.Expect(n.X+n.DiscRadius).To(BeNumerically("<=", 1920),
			"node %d right edge should be <= width", i)
		g.Expect(n.Y-n.DiscRadius).To(BeNumerically(">=", 0),
			"node %d top edge should be >= 0", i)
		g.Expect(n.Y+n.DiscRadius).To(BeNumerically("<=", 1080),
			"node %d bottom edge should be <= height", i)
	}
}

func TestLayoutPartialLapHourly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(6, Hourly) // 1/4 of a lap
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelNone)
	g.Expect(nodes).To(HaveLen(6))

	// Angular span should be approximately 5 × 2π/24 (last index = 5).
	lastAngle := nodes[5].Angle
	expectedAngle := 5 * (2 * math.Pi / 24.0)
	g.Expect(lastAngle).To(BeNumerically("~", expectedAngle, 0.001))
}

func TestLayoutLabelLapsDaily(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(60, Daily) // 2+ laps
	nodes := Layout(buckets, 1920, 1920, Daily, LabelLaps)

	for i, n := range nodes {
		if i%28 == 0 {
			g.Expect(n.ShowLabel).To(BeTrue(),
				"lap boundary at index %d should be labelled", i)
		} else {
			g.Expect(n.ShowLabel).To(BeFalse(),
				"non-boundary at index %d should not be labelled", i)
		}
	}
}

func TestLayoutDiscRadiusPositive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := makeBuckets(10, Hourly)
	nodes := Layout(buckets, 1920, 1920, Hourly, LabelAll)

	for i, n := range nodes {
		g.Expect(n.DiscRadius).To(BeNumerically(">", 0),
			"node %d should have positive disc radius", i)
	}
}
