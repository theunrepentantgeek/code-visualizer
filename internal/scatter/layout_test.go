package scatter

import (
	"fmt"
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func scatterTestFile(name string) *model.File {
	return &model.File{Name: name, Path: name}
}

func TestCollectDataset_SkipsFilesMissingAxisOrSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	keep := scatterTestFile("keep.go")
	keep.SetClassification(filesystem.FileType, "go")
	keep.SetQuantity(filesystem.FileLines, 20)
	keep.SetQuantity(filesystem.FileSize, 100)

	missingX := scatterTestFile("missing-x.go")
	missingX.SetQuantity(filesystem.FileLines, 10)
	missingX.SetQuantity(filesystem.FileSize, 80)

	missingY := scatterTestFile("missing-y.go")
	missingY.SetClassification(filesystem.FileType, "txt")
	missingY.SetQuantity(filesystem.FileSize, 60)

	missingSize := scatterTestFile("missing-size.go")
	missingSize.SetClassification(filesystem.FileType, "md")
	missingSize.SetQuantity(filesystem.FileLines, 40)

	root := &model.Directory{Files: []*model.File{keep, missingX, missingY, missingSize}}

	dataset := CollectDataset(
		root,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
		filesystem.FileSize,
	)

	g.Expect(dataset.Points).To(HaveLen(1))
	g.Expect(dataset.Points[0].File).To(Equal(keep))
	g.Expect(dataset.Skipped.MissingX).To(Equal(1))
	g.Expect(dataset.Skipped.MissingY).To(Equal(1))
	g.Expect(dataset.Skipped.MissingSize).To(Equal(1))
}

func TestDataset_Files_ReturnsFilesInOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f1 := scatterTestFile("a.go")
	f2 := scatterTestFile("b.go")
	dataset := Dataset{
		Points: []PointDatum{
			{File: f1, X: AxisValue{Numeric: 1}, Y: AxisValue{Numeric: 2}, Size: 3},
			{File: f2, X: AxisValue{Numeric: 4}, Y: AxisValue{Numeric: 5}, Size: 6},
		},
	}

	g.Expect(dataset.Files()).To(Equal([]*model.File{f1, f2}))
}

func TestDataset_Files_EmptyDataset_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Dataset{}.Files()).To(BeEmpty())
}

func TestSkipCounts_Total_SumsAllFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := SkipCounts{MissingX: 2, MissingY: 3, MissingSize: 5}
	g.Expect(s.Total()).To(Equal(10))
}

func TestSkipCounts_Total_ZeroWhenNoneSkipped(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(SkipCounts{}.Total()).To(Equal(0))
}

func TestCollectDataset_SkipCountsReflectMissingValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	missingX := scatterTestFile("no-x.go")
	missingX.SetQuantity(filesystem.FileLines, 10)
	missingX.SetQuantity(filesystem.FileSize, 80)

	missingY := scatterTestFile("no-y.go")
	missingY.SetClassification(filesystem.FileType, "txt")
	missingY.SetQuantity(filesystem.FileSize, 60)

	missingSize := scatterTestFile("no-size.go")
	missingSize.SetClassification(filesystem.FileType, "md")
	missingSize.SetQuantity(filesystem.FileLines, 40)

	root := &model.Directory{Files: []*model.File{missingX, missingY, missingSize}}

	dataset := CollectDataset(
		root,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
		filesystem.FileSize,
	)

	g.Expect(dataset.Skipped.MissingX).To(Equal(1))
	g.Expect(dataset.Skipped.MissingY).To(Equal(1))
	g.Expect(dataset.Skipped.MissingSize).To(Equal(1))
	g.Expect(dataset.Skipped.Total()).To(Equal(3))
}

func TestLayout_CategoricalAxesUseAlphabeticalBands(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	alpha := scatterTestFile("alpha.go")
	alpha.SetClassification(filesystem.FileType, "alpha")
	alpha.SetClassification(metric.Name("y-cat"), "beta")
	alpha.SetQuantity(filesystem.FileSize, 80)

	zeta := scatterTestFile("zeta.go")
	zeta.SetClassification(filesystem.FileType, "zeta")
	zeta.SetClassification(metric.Name("y-cat"), "alpha")
	zeta.SetQuantity(filesystem.FileSize, 40)

	root := &model.Directory{Files: []*model.File{alpha, zeta}}
	dataset := CollectDataset(
		root,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: metric.Name("y-cat"), Kind: metric.Classification},
		filesystem.FileSize,
	)

	layout := Layout(
		dataset,
		800,
		600,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: metric.Name("y-cat"), Kind: metric.Classification},
	)

	g.Expect(layout.XAxis.Categorical.Bands).To(HaveLen(2))
	g.Expect(layout.XAxis.Categorical.Bands[0].Label).To(Equal("alpha"))
	g.Expect(layout.XAxis.Categorical.Bands[1].Label).To(Equal("zeta"))
	g.Expect(layout.YAxis.Categorical.Bands).To(HaveLen(2))
	g.Expect(layout.YAxis.Categorical.Bands[0].Label).To(Equal("alpha"))
	g.Expect(layout.YAxis.Categorical.Bands[1].Label).To(Equal("beta"))
	g.Expect(layout.Points[0].Radius).To(BeNumerically(">=", layout.Points[1].Radius))
}

func TestLayout_NumericYAxisPlacesHigherValuesHigherOnCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	low := scatterTestFile("low.go")
	low.SetQuantity(filesystem.FileLines, 10)
	low.SetQuantity(filesystem.FileSize, 20)
	low.SetClassification(filesystem.FileType, "go")

	high := scatterTestFile("high.go")
	high.SetQuantity(filesystem.FileLines, 100)
	high.SetQuantity(filesystem.FileSize, 60)
	high.SetClassification(filesystem.FileType, "go")

	root := &model.Directory{Files: []*model.File{low, high}}
	dataset := CollectDataset(
		root,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
		filesystem.FileSize,
	)

	layout := Layout(
		dataset,
		800,
		600,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
	)

	points := map[string]ScatterPoint{}
	for _, point := range layout.Points {
		points[point.File.Name] = point
	}

	g.Expect(points["high.go"].Y).To(BeNumerically("<", points["low.go"].Y))
	g.Expect(layout.YAxis.Numeric.Ticks).NotTo(BeEmpty())
}

func TestLayout_CrowdedPlotKeepsMinimumDiscRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	files := make([]*model.File, 0, 500)

	for i := range 500 {
		file := scatterTestFile(fmt.Sprintf("file-%03d.go", i))
		file.SetClassification(filesystem.FileType, "go")
		file.SetQuantity(filesystem.FileLines, int64(i+1))
		file.SetQuantity(filesystem.FileSize, int64(i+1))
		files = append(files, file)
	}

	root := &model.Directory{Files: files}
	dataset := CollectDataset(
		root,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
		filesystem.FileSize,
	)

	layout := Layout(
		dataset,
		800,
		600,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
	)

	for _, point := range layout.Points {
		g.Expect(point.Radius).To(BeNumerically(">=", scatterMinRadius))
	}
}

func TestNumericTicks_UsesRegularNiceSteps(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := numericTicks(0.219, 0.875, plot, horizontalAxis)

	g.Expect(ticks).To(HaveLen(8))
	g.Expect(ticks[0].Value).To(BeNumerically("~", 0.2, 1e-9))
	g.Expect(ticks[len(ticks)-1].Value).To(BeNumerically("~", 0.9, 1e-9))

	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Value - ticks[i-1].Value).To(BeNumerically("~", 0.1, 1e-9))
	}
}

func TestNumericTicks_NearZeroRangeIncludesZeroTick(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := numericTicks(9, 842, plot, horizontalAxis)

	g.Expect(ticks).To(HaveLen(10))
	g.Expect(ticks[0].Value).To(Equal(0.0))

	step := ticks[1].Value - ticks[0].Value
	g.Expect(step).To(Equal(100.0))

	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Value - ticks[i-1].Value).To(BeNumerically("~", step, 1e-9))
		g.Expect(math.Mod(ticks[i].Value, step)).To(BeNumerically("~", 0, 1e-9))
	}
}

func TestResolvedAxis_OffsetShiftsNumericTicks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	axis := ResolvedAxis{
		Numeric: &NumericAxis{Ticks: []AxisTick{{Position: 10}, {Position: 25}}},
	}

	axis.Offset(7.5)

	g.Expect(axis.Numeric.Ticks).To(Equal([]AxisTick{{Position: 17.5}, {Position: 32.5}}))
}

func TestResolvedAxis_OffsetShiftsCategoricalBands(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	axis := ResolvedAxis{
		Categorical: &CategoricalAxis{Bands: []AxisBand{{Label: "go", Start: 10, End: 20, Center: 15}}},
	}

	axis.Offset(-5)

	g.Expect(axis.Categorical.Bands).To(Equal([]AxisBand{{Label: "go", Start: 5, End: 15, Center: 10}}))
}

func TestLogNumericTicks_SpansMultipleDecades(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := logNumericTicks(1, 10000, plot, horizontalAxis)

	// Expect ticks at powers of 10: 1, 10, 100, 1000, 10000
	g.Expect(ticks).To(HaveLen(5))
	g.Expect(ticks[0].Value).To(BeNumerically("~", 1, 1e-9))
	g.Expect(ticks[1].Value).To(BeNumerically("~", 10, 1e-9))
	g.Expect(ticks[2].Value).To(BeNumerically("~", 100, 1e-9))
	g.Expect(ticks[3].Value).To(BeNumerically("~", 1000, 1e-9))
	g.Expect(ticks[4].Value).To(BeNumerically("~", 10000, 1e-9))

	// Positions should be logarithmically spaced (equal increments in log space)
	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Position).To(BeNumerically(">", ticks[i-1].Position))
	}

	// Each gap should be the same size (equal decades = equal spacing)
	gap := ticks[1].Position - ticks[0].Position
	for i := 2; i < len(ticks); i++ {
		g.Expect(ticks[i].Position - ticks[i-1].Position).To(BeNumerically("~", gap, 1e-6))
	}
}

func TestLogNumericTicks_NarrowRange_AddsIntermediateTicks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := logNumericTicks(50, 500, plot, horizontalAxis)

	// Range spans ~1 decade, so intermediate ticks (2x, 5x) are added.
	// Expect at least 4 ticks.
	g.Expect(len(ticks)).To(BeNumerically(">=", 4))

	// All tick values should be within [50, 500]
	for _, tick := range ticks {
		g.Expect(tick.Value).To(BeNumerically(">=", 50))
		g.Expect(tick.Value).To(BeNumerically("<=", 500))
	}

	// Positions should be monotonically increasing
	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Position).To(BeNumerically(">", ticks[i-1].Position))
	}
}

func TestLogNumericTicks_SubDecadeRange_UsesFallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	// Range [11, 19] has no power-of-10 or subdivision candidate inside it
	ticks := logNumericTicks(11, 19, plot, horizontalAxis)

	// Fallback generates 5 evenly-spaced ticks in log space
	g.Expect(ticks).To(HaveLen(5))
	g.Expect(ticks[0].Value).To(BeNumerically("~", 11, 0.01))
	g.Expect(ticks[4].Value).To(BeNumerically("~", 19, 0.01))

	// Positions should be monotonically increasing
	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Position).To(BeNumerically(">", ticks[i-1].Position))
	}
}

func TestLogNumericTicks_SingleValue_ReturnsCenterTick(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := logNumericTicks(42, 42, plot, horizontalAxis)

	g.Expect(ticks).To(HaveLen(1))
	g.Expect(ticks[0].Value).To(BeNumerically("~", 42, 1e-9))
	g.Expect(ticks[0].Position).To(BeNumerically("~", 400, 1e-6)) // center of 800-wide plot
}

func TestLayout_LogScalePositionsPointsLogarithmically(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	small := scatterTestFile("small.go")
	small.SetQuantity(filesystem.FileLines, 10)
	small.SetQuantity(filesystem.FileSize, 100)

	medium := scatterTestFile("medium.go")
	medium.SetQuantity(filesystem.FileLines, 100)
	medium.SetQuantity(filesystem.FileSize, 100)

	large := scatterTestFile("large.go")
	large.SetQuantity(filesystem.FileLines, 1000)
	large.SetQuantity(filesystem.FileSize, 100)

	root := &model.Directory{Files: []*model.File{small, medium, large}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Log}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Linear}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)
	layout := Layout(dataset, 800, 600, xAxis, yAxis)

	points := map[string]ScatterPoint{}
	for _, point := range layout.Points {
		points[point.File.Name] = point
	}

	// With log scale, the gap between 10→100 should equal the gap between 100→1000
	// (both are one decade)
	gap1 := points["medium.go"].X - points["small.go"].X
	gap2 := points["large.go"].X - points["medium.go"].X
	g.Expect(gap1).To(BeNumerically("~", gap2, 1.0))

	// All X values should be within the plot area
	g.Expect(points["small.go"].X).To(BeNumerically(">=", scatterPlotLeftMargin))
	g.Expect(points["large.go"].X).To(BeNumerically("<=", 800-scatterPlotRightMargin))
}

func TestValidateLogScale_ErrorsOnZeroValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	zero := scatterTestFile("zero.go")
	zero.SetQuantity(filesystem.FileLines, 0)
	zero.SetQuantity(filesystem.FileSize, 100)

	positive := scatterTestFile("positive.go")
	positive.SetQuantity(filesystem.FileLines, 10)
	positive.SetQuantity(filesystem.FileSize, 50)

	root := &model.Directory{Files: []*model.File{zero, positive}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Log}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Linear}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)

	err := ValidateLogScale(dataset, xAxis, yAxis)
	g.Expect(err).To(HaveOccurred())
	//nolint:nilaway,nolintlint // guarded by HaveOccurred above
	g.Expect(err).To(MatchError(ContainSubstring("x-axis")))
	g.Expect(err).To(MatchError(ContainSubstring("zero.go")))
}

func TestValidateLogScale_PassesWhenAllPositive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := scatterTestFile("a.go")
	a.SetQuantity(filesystem.FileLines, 10)
	a.SetQuantity(filesystem.FileSize, 100)

	b := scatterTestFile("b.go")
	b.SetQuantity(filesystem.FileLines, 200)
	b.SetQuantity(filesystem.FileSize, 50)

	root := &model.Directory{Files: []*model.File{a, b}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Log}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Log}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)

	err := ValidateLogScale(dataset, xAxis, yAxis)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestValidateLogScale_SkipsLinearAxes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	zero := scatterTestFile("zero.go")
	zero.SetQuantity(filesystem.FileLines, 0)
	zero.SetQuantity(filesystem.FileSize, 100)

	root := &model.Directory{Files: []*model.File{zero}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Linear}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Linear}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)

	err := ValidateLogScale(dataset, xAxis, yAxis)
	g.Expect(err).NotTo(HaveOccurred())
}
