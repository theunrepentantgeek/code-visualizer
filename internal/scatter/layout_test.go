package scatter

import (
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
