package render

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/treemap"
)

func TestDefaultOrientation_CenterPositions_ReturnsHorizontal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(DefaultOrientation(LegendPositionTopCenter)).To(Equal(LegendOrientationHorizontal))
	g.Expect(DefaultOrientation(LegendPositionBottomCenter)).To(Equal(LegendOrientationHorizontal))
}

func TestDefaultOrientation_SidePositions_ReturnsVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sides := []LegendPosition{
		LegendPositionTopLeft,
		LegendPositionTopRight,
		LegendPositionCenterRight,
		LegendPositionBottomRight,
		LegendPositionBottomLeft,
		LegendPositionCenterLeft,
	}

	for _, pos := range sides {
		g.Expect(DefaultOrientation(pos)).To(Equal(LegendOrientationVertical),
			"expected vertical for %s", pos)
	}
}

func TestLegendOrigin_AllPositions_InBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	canvasW, canvasH := 800.0, 600.0
	legendW, legendH := 100.0, 50.0

	positions := []LegendPosition{
		LegendPositionTopLeft,
		LegendPositionTopCenter,
		LegendPositionTopRight,
		LegendPositionCenterRight,
		LegendPositionBottomRight,
		LegendPositionBottomCenter,
		LegendPositionBottomLeft,
		LegendPositionCenterLeft,
	}

	for _, pos := range positions {
		ox, oy := legendOrigin(pos, canvasW, canvasH, legendW, legendH)
		g.Expect(ox).To(BeNumerically(">=", 0), "x out of bounds for %s", pos)
		g.Expect(oy).To(BeNumerically(">=", 0), "y out of bounds for %s", pos)
		g.Expect(ox+legendW).To(BeNumerically("<=", canvasW),
			"right edge out of bounds for %s", pos)
		g.Expect(oy+legendH).To(BeNumerically("<=", canvasH),
			"bottom edge out of bounds for %s", pos)
	}
}

func TestLegendOrigin_TopLeft_IsNearOrigin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ox, oy := legendOrigin(LegendPositionTopLeft, 800, 600, 100, 50)
	g.Expect(ox).To(Equal(legendMargin))
	g.Expect(oy).To(Equal(legendMargin))
}

func TestLegendOrigin_BottomRight_IsNearCorner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ox, oy := legendOrigin(LegendPositionBottomRight, 800, 600, 100, 50)
	g.Expect(ox).To(Equal(800.0 - 100.0 - legendMargin))
	g.Expect(oy).To(Equal(600.0 - 50.0 - legendMargin))
}

func TestFormatBreakpoint_IntegerValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(formatBreakpoint(42)).To(Equal("42"))
	g.Expect(formatBreakpoint(0)).To(Equal("0"))
	g.Expect(formatBreakpoint(1000)).To(Equal("1000"))
}

func TestFormatBreakpoint_FloatValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(formatBreakpoint(3.14)).To(Equal("3.1"))
	g.Expect(formatBreakpoint(0.5)).To(Equal("0.5"))
}

func makeSampleLegendInfo(orient LegendOrientation) *LegendInfo {
	pal := palette.GetPalette("temperature")

	return &LegendInfo{
		Position:    LegendPositionBottomRight,
		Orientation: orient,
		Entries: []LegendEntry{
			{
				Role:       "Fill",
				MetricName: "file-size",
				Kind:       metric.Quantity,
				Palette:    pal,
				Buckets: &metric.BucketBoundaries{
					Boundaries: []float64{100, 500, 1000, 5000},
					Min:        10,
					Max:        10000,
					StepCount:  5,
				},
			},
			{
				Role:       "Border",
				MetricName: "file-type",
				Kind:       metric.Classification,
				Categories: []CategorySwatch{
					{Label: "go", Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}},
					{Label: "rs", Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}},
					{Label: "py", Colour: color.RGBA{R: 53, G: 114, B: 165, A: 255}},
				},
			},
		},
	}
}

func TestDrawLegend_NilInfo_DoesNotPanic(t *testing.T) {
	t.Parallel()

	// Should not panic.
	drawLegendOnTestCanvas(t, nil)
}

func TestDrawLegend_NonePosition_DoesNotPanic(t *testing.T) {
	t.Parallel()

	info := &LegendInfo{
		Position: LegendPositionNone,
	}

	drawLegendOnTestCanvas(t, info)
}

func TestDrawLegend_Vertical_ProducesImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := drawLegendOnTestCanvas(t, makeSampleLegendInfo(LegendOrientationVertical))

	fi, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if fi == nil {
		t.Fatal("os.Stat returned nil FileInfo")
	}

	g.Expect(fi.Size()).To(BeNumerically(">", 0))
}

func TestDrawLegend_Horizontal_ProducesImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := drawLegendOnTestCanvas(t, makeSampleLegendInfo(LegendOrientationHorizontal))

	fi, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if fi == nil {
		t.Fatal("os.Stat returned nil FileInfo")
	}

	g.Expect(fi.Size()).To(BeNumerically(">", 0))
}

func TestWriteSVGLegend_NilInfo_DoesNotPanic(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "legend-nil.svg")

	f, err := os.Create(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	writeSVGLegend(f, nil, 800, 600)
}

func TestWriteSVGLegend_Vertical_ProducesOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	content := writeSVGLegendToString(t, makeSampleLegendInfo(LegendOrientationVertical))
	g.Expect(content).To(ContainSubstring("<g transform="))
	g.Expect(content).To(ContainSubstring("file-size"))
	g.Expect(content).To(ContainSubstring("file-type"))
}

func TestWriteSVGLegend_Horizontal_ProducesOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	content := writeSVGLegendToString(t, makeSampleLegendInfo(LegendOrientationHorizontal))
	g.Expect(content).To(ContainSubstring("<g transform="))
	g.Expect(content).To(ContainSubstring("fill-opacity"))
}

func TestDrawLegend_SizeOnlyEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Position:    LegendPositionTopLeft,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{
				Role:       "Size",
				MetricName: "file-lines",
				Kind:       metric.Quantity,
			},
		},
	}

	out := drawLegendOnTestCanvas(t, info)

	fi, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if fi == nil {
		t.Fatal("os.Stat returned nil FileInfo")
	}

	g.Expect(fi.Size()).To(BeNumerically(">", 0))
}

// drawLegendOnTestCanvas renders a blank canvas with a legend overlay
// and saves it to a temp PNG file.
func drawLegendOnTestCanvas(t *testing.T, info *LegendInfo) string {
	t.Helper()

	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "legend.png")

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
		},
	}

	rects := treemap.Layout(root, 800, 600, "file-size")
	err := Render(rects, 800, 600, out, info)
	g.Expect(err).NotTo(HaveOccurred())

	return out
}

// writeSVGLegendToString writes an SVG legend into a temp file and returns the content.
func writeSVGLegendToString(t *testing.T, info *LegendInfo) string {
	t.Helper()

	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "legend.svg")

	f, err := os.Create(out)
	g.Expect(err).NotTo(HaveOccurred())

	writeSVGLegend(f, info, 800, 600)
	f.Close()

	content, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	return string(content)
}

func TestReserveLegendSpace_NilInfo_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	wReduce, hReduce := ReserveLegendSpace(nil)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveLegendSpace_NonePosition_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{Position: LegendPositionNone}
	wReduce, hReduce := ReserveLegendSpace(info)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveLegendSpace_CenterRight_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationVertical)
	info.Position = LegendPositionCenterRight
	wReduce, hReduce := ReserveLegendSpace(info)
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}

func TestReserveLegendSpace_CenterLeft_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationVertical)
	info.Position = LegendPositionCenterLeft
	wReduce, hReduce := ReserveLegendSpace(info)
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}

func TestReserveLegendSpace_EmptyEntries_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Position:    LegendPositionBottomRight,
		Orientation: LegendOrientationVertical,
		Entries:     []LegendEntry{},
	}

	wReduce, hReduce := ReserveLegendSpace(info)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

// --- Regression tests for issue #89: Horizontal legend layout ---
//
// measureLegendH used to stack entries vertically even in horizontal mode,
// making horizontal legends unnecessarily tall. The fix ensures entries are
// placed side-by-side so the legend is wide and short.

func TestMeasureLegendH_MultipleEntries_HeightSmallerThanVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationHorizontal)
	infoV := makeSampleLegendInfo(LegendOrientationVertical)

	dc := gg.NewContext(1, 1)
	_, hH := measureLegend(dc, info)
	_, hV := measureLegend(dc, infoV)

	// Regression test for #89: horizontal layout should be shorter than
	// vertical when there are multiple entries (side-by-side, not stacked).
	g.Expect(hH).To(BeNumerically("<", hV),
		"horizontal legend height (%.1f) should be less than vertical (%.1f)", hH, hV)
}

func TestMeasureLegendH_MultipleEntries_WiderThanVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationHorizontal)
	infoV := makeSampleLegendInfo(LegendOrientationVertical)

	dc := gg.NewContext(1, 1)
	wH, _ := measureLegend(dc, info)
	wV, _ := measureLegend(dc, infoV)

	// Regression test for #89: horizontal layout with multiple entries
	// should be wider than vertical (entries placed side-by-side).
	g.Expect(wH).To(BeNumerically(">", wV),
		"horizontal legend width (%.1f) should be greater than vertical (%.1f)", wH, wV)
}

// --- Regression tests for issue #90: Legend orientation impacts margin carve out ---
//
// ReserveLegendSpace used to only consider Position, ignoring Orientation.
// A vertical (tall, narrow) legend in a corner should carve out width, not
// height. A horizontal (wide, short) legend in a corner should carve out
// height, not width.

func TestReserveLegendSpace_BottomRight_VerticalOrientation_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationVertical)
	info.Position = LegendPositionBottomRight

	wReduce, hReduce := ReserveLegendSpace(info)

	// Regression test for #90: vertical orientation at bottom-right should
	// carve out width on the right side, not height below.
	g.Expect(wReduce).To(BeNumerically(">", 0),
		"vertical legend at bottom-right should reduce width")
	g.Expect(hReduce).To(BeZero(),
		"vertical legend at bottom-right should NOT reduce height")
}

func TestReserveLegendSpace_BottomRight_HorizontalOrientation_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationHorizontal)
	info.Position = LegendPositionBottomRight

	wReduce, hReduce := ReserveLegendSpace(info)

	// A horizontal (wide, short) legend at bottom-right should carve out
	// height below the visualization.
	g.Expect(hReduce).To(BeNumerically(">", 0),
		"horizontal legend at bottom-right should reduce height")
	g.Expect(wReduce).To(BeZero(),
		"horizontal legend at bottom-right should NOT reduce width")
}

func TestReserveLegendSpace_TopLeft_VerticalOrientation_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationVertical)
	info.Position = LegendPositionTopLeft

	wReduce, hReduce := ReserveLegendSpace(info)

	// Regression test for #90: vertical orientation at top-left should
	// carve out width on the left side.
	g.Expect(wReduce).To(BeNumerically(">", 0),
		"vertical legend at top-left should reduce width")
	g.Expect(hReduce).To(BeZero(),
		"vertical legend at top-left should NOT reduce height")
}

func TestReserveLegendSpace_BottomLeft_HorizontalOrientation_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationHorizontal)
	info.Position = LegendPositionBottomLeft

	wReduce, hReduce := ReserveLegendSpace(info)

	// Regression test for #90: horizontal orientation at bottom-left should
	// carve out height below.
	g.Expect(hReduce).To(BeNumerically(">", 0),
		"horizontal legend at bottom-left should reduce height")
	g.Expect(wReduce).To(BeZero(),
		"horizontal legend at bottom-left should NOT reduce width")
}

func TestReserveLegendSpace_TopRight_VerticalOrientation_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationVertical)
	info.Position = LegendPositionTopRight

	wReduce, hReduce := ReserveLegendSpace(info)

	// Regression test for #90: vertical orientation at top-right should
	// carve out width on the right side.
	g.Expect(wReduce).To(BeNumerically(">", 0),
		"vertical legend at top-right should reduce width")
	g.Expect(hReduce).To(BeZero(),
		"vertical legend at top-right should NOT reduce height")
}

func TestReserveLegendSpace_TopRight_HorizontalOrientation_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := makeSampleLegendInfo(LegendOrientationHorizontal)
	info.Position = LegendPositionTopRight

	wReduce, hReduce := ReserveLegendSpace(info)

	// Regression test for #90: horizontal orientation at top-right should
	// carve out height above.
	g.Expect(hReduce).To(BeNumerically(">", 0),
		"horizontal legend at top-right should reduce height")
	g.Expect(wReduce).To(BeZero(),
		"horizontal legend at top-right should NOT reduce width")
}
