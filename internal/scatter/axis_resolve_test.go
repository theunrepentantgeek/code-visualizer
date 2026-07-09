package scatter

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

// TestIncludeNearZero verifies the zero-snapping behaviour of axis bounds.
func TestIncludeNearZero_PositiveRangeFarFromZero_Unchanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	lower, upper := includeNearZero(500, 900)

	// Assert
	g.Expect(lower).To(BeNumerically("~", 500, 1e-9))
	g.Expect(upper).To(BeNumerically("~", 900, 1e-9))
}

func TestIncludeNearZero_PositiveMinNearZero_SnapsToZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	// span=841, snapMargin=841*0.2=168.2; min=9 < 168.2 → snaps to 0
	lower, upper := includeNearZero(9, 842)

	// Assert
	g.Expect(lower).To(BeNumerically("~", 0, 1e-9))
	g.Expect(upper).To(BeNumerically("~", 842, 1e-9))
}

func TestIncludeNearZero_NegativeMaxNearZero_SnapsToZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	// span=841, snapMargin=168.2; max=-9, abs(-9)=9 < 168.2 → snaps to 0
	lower, upper := includeNearZero(-842, -9)

	// Assert
	g.Expect(lower).To(BeNumerically("~", -842, 1e-9))
	g.Expect(upper).To(BeNumerically("~", 0, 1e-9))
}

func TestIncludeNearZero_ZeroSpan_ReturnsUnchanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	lower, upper := includeNearZero(5, 5)

	// Assert
	g.Expect(lower).To(BeNumerically("~", 5, 1e-9))
	g.Expect(upper).To(BeNumerically("~", 5, 1e-9))
}

// TestFormatTick verifies tick label formatting across step sizes.
func TestFormatTick_VariousInputs_ProducesExpectedLabels(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		value    float64
		step     float64
		expected string
	}{
		"integer step formats as integer": {
			value:    100.0,
			step:     100.0,
			expected: "100",
		},
		"step zero uses g format": {
			value:    42.5,
			step:     0,
			expected: "42.5",
		},
		"step 0.1 uses 1 decimal": {
			value:    0.5,
			step:     0.1,
			expected: "0.5",
		},
		"step 0.01 uses 2 decimals": {
			value:    1.25,
			step:     0.01,
			expected: "1.25",
		},
		"step 1000 formats as integer": {
			value:    5000.0,
			step:     1000.0,
			expected: "5000",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			// Act
			result := formatTick(c.value, c.step)

			// Assert
			g.Expect(result).To(Equal(c.expected))
		})
	}
}

// TestNumericExtent verifies min/max extraction from point slices.
func TestNumericExtent_EmptySlice_ReturnsZeros(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	min, max := numericExtent(nil, horizontalAxis)

	// Assert
	g.Expect(min).To(Equal(0.0))
	g.Expect(max).To(Equal(0.0))
}

func TestNumericExtent_SinglePoint_ReturnsThatValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	points := []PointDatum{{X: AxisValue{Numeric: 42.0}}}

	// Act
	min, max := numericExtent(points, horizontalAxis)

	// Assert
	g.Expect(min).To(BeNumerically("~", 42.0, 1e-9))
	g.Expect(max).To(BeNumerically("~", 42.0, 1e-9))
}

func TestNumericExtent_MultiplePoints_ReturnsCorrectExtent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange
	points := []PointDatum{
		{X: AxisValue{Numeric: 10.0}, Y: AxisValue{Numeric: 100.0}},
		{X: AxisValue{Numeric: 50.0}, Y: AxisValue{Numeric: 20.0}},
		{X: AxisValue{Numeric: 30.0}, Y: AxisValue{Numeric: 70.0}},
	}

	// Act
	xMin, xMax := numericExtent(points, horizontalAxis)
	yMin, yMax := numericExtent(points, verticalAxis)

	// Assert
	g.Expect(xMin).To(BeNumerically("~", 10.0, 1e-9))
	g.Expect(xMax).To(BeNumerically("~", 50.0, 1e-9))
	g.Expect(yMin).To(BeNumerically("~", 20.0, 1e-9))
	g.Expect(yMax).To(BeNumerically("~", 100.0, 1e-9))
}

// TestLogTickCandidates verifies power-of-10 tick candidate selection.
func TestLogTickCandidates_MultiDecadeRange_ReturnsPowersOfTen(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	candidates := logTickCandidates(1, 10000)

	// Assert
	g.Expect(candidates).To(Equal([]float64{1, 10, 100, 1000, 10000}))
}

func TestLogTickCandidates_NoPowerOfTenInRange_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	candidates := logTickCandidates(15, 90)

	// Assert: no power of 10 in [15, 90]
	g.Expect(candidates).To(BeEmpty())
}

func TestLogTickCandidates_SingleDecade_ReturnsBothBoundaries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	candidates := logTickCandidates(10, 1000)

	// Assert
	g.Expect(candidates).To(Equal([]float64{10, 100, 1000}))
}

// TestLogTickCandidatesWithSubdivisions verifies that 2x and 5x marks are added.
func TestLogTickCandidatesWithSubdivisions_SingleDecade_IncludesIntermediates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange / Act
	candidates := logTickCandidatesWithSubdivisions(50, 500)

	// Assert: expect 50(=10*5), 100, 200, 500 within [50, 500]
	g.Expect(candidates).To(ContainElements(
		BeNumerically("~", 50, 1e-9),
		BeNumerically("~", 100, 1e-9),
		BeNumerically("~", 200, 1e-9),
		BeNumerically("~", 500, 1e-9),
	))
	g.Expect(len(candidates)).To(BeNumerically(">=", 4))
}

func TestLogTickCandidatesWithSubdivisions_AllValuesInRange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const (
		minVal = 1.0
		maxVal = 100.0
	)

	// Arrange / Act
	candidates := logTickCandidatesWithSubdivisions(minVal, maxVal)

	// Assert: all returned values must be within [min, max]
	for _, v := range candidates {
		g.Expect(v).To(BeNumerically(">=", minVal))
		g.Expect(v).To(BeNumerically("<=", maxVal))
	}
}

// TestLogTickFallback verifies evenly-spaced fallback tick generation.
func TestLogTickFallback_ReturnsExactlyFiveTicks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	candidates := logTickFallback(11, 19)

	// Assert
	g.Expect(candidates).To(HaveLen(logFallbackTicks))
}

func TestLogTickFallback_FirstAndLastMatchEndpoints(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	candidates := logTickFallback(11, 19)

	// Assert
	g.Expect(candidates[0]).To(BeNumerically("~", 11, 0.01))
	g.Expect(candidates[logFallbackTicks-1]).To(BeNumerically("~", 19, 0.01))
}

func TestLogTickFallback_ValuesMonotonicallyIncreasing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act
	candidates := logTickFallback(11, 19)

	// Assert
	for i := 1; i < len(candidates); i++ {
		g.Expect(candidates[i]).To(BeNumerically(">", candidates[i-1]))
	}
}

// TestMakeTickCandidate verifies tick step evaluation.
func TestMakeTickCandidate_ValidStepInRange_ReturnsCandidate(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act: step=100 for range [0, 800] gives 8 gaps — within [4,10]
	cand, ok := makeTickCandidate(0, 800, 100)

	// Assert
	g.Expect(ok).To(BeTrue())
	g.Expect(cand.gaps).To(BeNumerically(">=", scatterMinTickGaps))
	g.Expect(cand.gaps).To(BeNumerically("<=", scatterMaxTickGaps))
	g.Expect(cand.step).To(BeNumerically("~", 100, 1e-9))
}

func TestMakeTickCandidate_TooFewGaps_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act: step=500 for range [0, 1000] gives only 2 gaps < 4
	_, ok := makeTickCandidate(0, 1000, 500)

	// Assert
	g.Expect(ok).To(BeFalse())
}

func TestMakeTickCandidate_TooManyGaps_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Act: step=1 for range [0, 1000] gives 1000 gaps > 10
	_, ok := makeTickCandidate(0, 1000, 1)

	// Assert
	g.Expect(ok).To(BeFalse())
}

// TestAxisSlotSize verifies per-slot size calculation.
func TestAxisSlotSize_CategoricalAxis_ReturnsBandSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: 4 bands over a span of 800
	axis := ResolvedAxis{
		Categorical: &CategoricalAxis{
			Bands: []AxisBand{
				{Label: "a"}, {Label: "b"}, {Label: "c"}, {Label: "d"},
			},
		},
	}

	// Act
	size := axisSlotSize(axis, 800, 100)

	// Assert: 800 / 4 = 200
	g.Expect(size).To(BeNumerically("~", 200, 1e-9))
}

func TestAxisSlotSize_NumericAxis_UsesSquareRootOfPointCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: numeric axis, 100 points, span 800
	axis := ResolvedAxis{Numeric: &NumericAxis{Min: 0, Max: 100}}

	// Act: sqrt(100)=10 > scatterMinNumericSlots(8), so divisor=10
	size := axisSlotSize(axis, 800, 100)

	// Assert: 800 / 10 = 80
	g.Expect(size).To(BeNumerically("~", 80, 1e-9))
}

func TestAxisSlotSize_FewPoints_UsesMinNumericSlots(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: numeric axis, only 2 points → sqrt(2)≈1.41 < scatterMinNumericSlots(8)
	axis := ResolvedAxis{Numeric: &NumericAxis{Min: 0, Max: 100}}

	// Act
	size := axisSlotSize(axis, 800, 2)

	// Assert: 800 / scatterMinNumericSlots = 800/8 = 100
	g.Expect(size).To(BeNumerically("~", 800/scatterMinNumericSlots, 1e-9))
}

// TestPositionForValue verifies that values are correctly mapped to canvas positions.
func TestPositionForValue_NumericHorizontal_MapsToCorrectPosition(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: horizontal axis [0, 100] on a 200-wide plot at x=100
	plot := PlotRect{X: 100, Y: 0, W: 200, H: 600}
	axis := ResolvedAxis{Numeric: &NumericAxis{Min: 0, Max: 100, Scale: Linear}}

	// Act: value 50 should map to midpoint
	pos := positionForValue(AxisValue{Numeric: 50}, axis, plot, horizontalAxis)

	// Assert: x=100 + 0.5*200 = 200
	g.Expect(pos).To(BeNumerically("~", 200, 1e-9))
}

func TestPositionForValue_NumericVertical_InvertsAxis(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: vertical axis [0, 100] on a 600-tall plot at y=0
	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	axis := ResolvedAxis{Numeric: &NumericAxis{Min: 0, Max: 100, Scale: Linear}}

	// Act: higher value should appear higher (lower Y) on canvas
	posLow := positionForValue(AxisValue{Numeric: 0}, axis, plot, verticalAxis)
	posHigh := positionForValue(AxisValue{Numeric: 100}, axis, plot, verticalAxis)

	// Assert: norm=0 → bottom (Y + H); norm=1 → top (Y + 0)
	g.Expect(posLow).To(BeNumerically("~", float64(plot.H), 1e-9))
	g.Expect(posHigh).To(BeNumerically("~", 0, 1e-9))
}

func TestPositionForValue_CategoricalAxis_UsesCenter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: categorical axis with two bands
	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	axis := ResolvedAxis{
		Categorical: &CategoricalAxis{
			Bands: []AxisBand{
				{Label: "go", Start: 0, End: 400, Center: 200},
				{Label: "md", Start: 400, End: 800, Center: 600},
			},
		},
	}

	// Act
	goPos := positionForValue(AxisValue{Category: "go"}, axis, plot, horizontalAxis)
	mdPos := positionForValue(AxisValue{Category: "md"}, axis, plot, horizontalAxis)

	// Assert: uses linear scan over Bands (Centers is nil)
	g.Expect(goPos).To(BeNumerically("~", 200, 1e-9))
	g.Expect(mdPos).To(BeNumerically("~", 600, 1e-9))
}

func TestPositionForValue_LogScale_MapsLogarithmically(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: log-scale horizontal axis [10, 1000] on an 800-wide plot at x=0
	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	axis := ResolvedAxis{Numeric: &NumericAxis{Min: 10, Max: 1000, Scale: Log}}

	// Act
	// norm(100) = (ln100 - ln10) / (ln1000 - ln10) = ln10 / (2*ln10) = 0.5
	pos := positionForValue(AxisValue{Numeric: 100}, axis, plot, horizontalAxis)

	// Assert: should be at 50% of the plot width
	g.Expect(pos).To(BeNumerically("~", 400, 1e-3))
}

func TestPositionForValue_EqualMinMax_ReturnsCenterPosition(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Arrange: degenerate axis where min == max
	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	axis := ResolvedAxis{Numeric: &NumericAxis{Min: 50, Max: 50, Scale: Linear}}

	// Act
	pos := positionForValue(AxisValue{Numeric: 50}, axis, plot, horizontalAxis)

	// Assert: returns the centre of the axis span
	g.Expect(pos).To(BeNumerically("~", 400, 1e-9)) // plot.X + plot.W/2
}

// TestNiceTickStep verifies that nice tick steps stay within the allowed gap range.
func TestNiceTickStep_StandardRange_GapsWithinBounds(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		min, max float64
	}{
		"0 to 100":           {0, 100},
		"0 to 1000":          {0, 1000},
		"219 to 875 (tenth)": {0.219, 0.875},
		"12345 to 98765":     {12345, 98765},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			// Act
			step, start, end := niceTickStep(c.min, c.max)

			// Assert: gap count should be within allowed bounds
			g.Expect(step).To(BeNumerically(">", 0))
			gaps := math.Round((end - start) / step)
			g.Expect(gaps).To(BeNumerically(">=", float64(scatterMinTickGaps)))
			g.Expect(gaps).To(BeNumerically("<=", float64(scatterMaxTickGaps)))

			// Range [start, end] should cover [min, max]
			g.Expect(start).To(BeNumerically("<=", c.min))
			g.Expect(end).To(BeNumerically(">=", c.max))
		})
	}
}
