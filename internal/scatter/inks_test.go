package scatter

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

// uniqueCategories

func TestUniqueCategories_ReturnsSortedDistinctValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	goFile := &model.File{Name: "main.go"}
	goFile.SetClassification(filesystem.FileType, "go")

	mdFile := &model.File{Name: "readme.md"}
	mdFile.SetClassification(filesystem.FileType, "md")

	dupFile := &model.File{Name: "util.go"}
	dupFile.SetClassification(filesystem.FileType, "go")

	cats := uniqueCategories([]*model.File{goFile, mdFile, dupFile}, filesystem.FileType)
	g.Expect(cats).To(Equal([]string{"go", "md"}))
}

func TestUniqueCategories_EmptyFiles_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := uniqueCategories([]*model.File{}, filesystem.FileType)
	g.Expect(cats).To(BeEmpty())
}

func TestUniqueCategories_NoMatchingMetric_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "main.go"}
	f.SetClassification(filesystem.FileType, "go")

	cats := uniqueCategories([]*model.File{f}, metric.Name("other-metric"))
	g.Expect(cats).To(BeEmpty())
}

// buildCategoricalInk

func TestBuildCategoricalInk_NoFiles_ReturnsFixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fallback := color.RGBA{R: 0xAA, G: 0xBB, B: 0xCC, A: 0xFF}
	ink := buildCategoricalInk([]*model.File{}, filesystem.FileType, palette.GetPalette(palette.Categorization), fallback)

	g.Expect(ink.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildCategoricalInk_WithCategories_ReturnsCategoricalInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := &model.File{Name: "main.go"}
	f.SetClassification(filesystem.FileType, "go")

	fallback := color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	ink := buildCategoricalInk([]*model.File{f}, filesystem.FileType, palette.GetPalette(palette.Categorization), fallback)

	g.Expect(ink.Info().Kind).To(Equal(inks.KindCategorical))
}

// categoricalPosition

func TestCategoricalPosition_CentersMap_ReturnsMapValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 200, H: 100}
	axis := &CategoricalAxis{
		Centers: map[string]float64{"go": 50, "md": 150},
	}

	pos := categoricalPosition(AxisValue{Category: "go"}, axis, plot, horizontalAxis)
	g.Expect(pos).To(BeNumerically("==", 50))
}

func TestCategoricalPosition_CentersMap_UnknownCategory_ReturnsCenter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 200, H: 100}
	axis := &CategoricalAxis{
		Centers: map[string]float64{"go": 50},
	}

	pos := categoricalPosition(AxisValue{Category: "unknown"}, axis, plot, horizontalAxis)
	g.Expect(pos).To(BeNumerically("==", 100)) // center = X + W/2 = 0 + 100
}

func TestCategoricalPosition_LinearScan_MatchingBand_ReturnsBandCenter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 200, H: 100}
	axis := &CategoricalAxis{
		Bands: []AxisBand{
			{Label: "go", Start: 0, End: 100, Center: 50},
			{Label: "md", Start: 100, End: 200, Center: 150},
		},
		// Centers intentionally nil to exercise linear-scan path.
	}

	pos := categoricalPosition(AxisValue{Category: "md"}, axis, plot, horizontalAxis)
	g.Expect(pos).To(BeNumerically("==", 150))
}

func TestCategoricalPosition_LinearScan_UnknownCategory_ReturnsCenter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 200, H: 100}
	axis := &CategoricalAxis{
		Bands: []AxisBand{
			{Label: "go", Start: 0, End: 100, Center: 50},
		},
	}

	pos := categoricalPosition(AxisValue{Category: "unknown"}, axis, plot, horizontalAxis)
	g.Expect(pos).To(BeNumerically("==", 100)) // center = X + W/2
}

// OffsetLayout

func TestOffsetLayout_ShiftsPlotOrigin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := &ScatterLayout{
		Plot: PlotRect{X: 10, Y: 20, W: 100, H: 50},
	}
	OffsetLayout(layout, 5, 3)
	g.Expect(layout.Plot.X).To(BeNumerically("==", 15))
	g.Expect(layout.Plot.Y).To(BeNumerically("==", 23))
}

func TestOffsetLayout_ShiftsPoints(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := &ScatterLayout{
		Points: []ScatterPoint{
			{X: 10, Y: 20},
			{X: 30, Y: 40},
		},
	}
	OffsetLayout(layout, 5, -10)
	g.Expect(layout.Points[0].X).To(BeNumerically("==", 15))
	g.Expect(layout.Points[0].Y).To(BeNumerically("==", 10))
	g.Expect(layout.Points[1].X).To(BeNumerically("==", 35))
	g.Expect(layout.Points[1].Y).To(BeNumerically("==", 30))
}

func TestOffsetLayout_ShiftsNumericAxisTicks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := &ScatterLayout{
		XAxis: ResolvedAxis{
			Numeric: &NumericAxis{
				Ticks: []AxisTick{
					{Value: 0, Position: 100},
					{Value: 50, Position: 150},
				},
			},
		},
	}
	OffsetLayout(layout, 20, 0)
	g.Expect(layout.XAxis.Numeric.Ticks[0].Position).To(BeNumerically("==", 120))
	g.Expect(layout.XAxis.Numeric.Ticks[1].Position).To(BeNumerically("==", 170))
}
