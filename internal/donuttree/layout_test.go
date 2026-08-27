package donuttree

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func directoryWithLines(name string, lines int64) *model.Directory {
	dir := &model.Directory{Name: name}
	dir.SetQuantity(filesystem.FileLines, lines)

	return dir
}

func TestLayoutAllocatesDirectorySectorsByMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	small := directoryWithLines("small", 10)
	large := directoryWithLines("large", 30)
	large.Dirs = []*model.Directory{directoryWithLines("nested", 20)}
	root := &model.Directory{Name: "root", Dirs: []*model.Directory{small, large}}

	layout := Layout(root, 800, filesystem.FileLines)

	g.Expect(layout.RootName).To(Equal("root"))
	g.Expect(layout.Children).To(HaveLen(2))
	g.Expect(layout.Children[0].Depth).To(Equal(1))
	g.Expect(layout.Children[1].Depth).To(Equal(1))
	g.Expect(layout.Children[1].SweepAngle).To(BeNumerically(">", layout.Children[0].SweepAngle))
	g.Expect(layout.Children[0].InnerRadius).To(BeNumerically("==", layout.Children[1].InnerRadius))
	g.Expect(layout.Children[0].OuterRadius).To(BeNumerically("==", layout.Children[1].OuterRadius))

	nested := layout.Children[1].Children[0]
	g.Expect(nested.StartAngle).To(BeNumerically(">=", layout.Children[1].StartAngle))
	g.Expect(nested.EndAngle()).To(BeNumerically("<=", layout.Children[1].EndAngle()))
	g.Expect(nested.InnerRadius).To(BeNumerically(">", layout.Children[1].InnerRadius))
}

func TestLayoutNilAndEmptyRootsHaveNoSectors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(Layout(nil, 800, filesystem.FileLines).Children).To(BeEmpty())
	g.Expect(Layout(&model.Directory{Name: "empty"}, 800, filesystem.FileLines).Children).To(BeEmpty())
}

func TestLayoutZeroValueSiblingsSplitParentEvenly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{
			{Name: "one"},
			{Name: "two"},
			{Name: "three"},
		},
	}

	layout := Layout(root, 800, filesystem.FileLines)
	g.Expect(layout.Children).To(HaveLen(3))

	for i, child := range layout.Children {
		g.Expect(child.SweepAngle).To(BeNumerically(">", 0))
		g.Expect(child.SweepAngle).To(BeNumerically("~", 2*math.Pi/3, 1e-12))
		if i > 0 {
			g.Expect(child.StartAngle).To(BeNumerically("==", layout.Children[i-1].EndAngle()))
		}
	}

	g.Expect(layout.Children[2].EndAngle()).To(BeNumerically("==", -math.Pi/2+2*math.Pi))
}

func TestLayoutReservesPositiveSectorsForZeroValueSiblings(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{
			directoryWithLines("positive", 10),
			{Name: "zero-one"},
			{Name: "zero-two"},
		},
	}

	layout := Layout(root, 800, filesystem.FileLines)
	g.Expect(layout.Children).To(HaveLen(3))

	for i, child := range layout.Children {
		g.Expect(child.SweepAngle).To(BeNumerically(">", 0))
		if i > 0 {
			g.Expect(child.StartAngle).To(BeNumerically("==", layout.Children[i-1].EndAngle()))
		}
	}

	g.Expect(layout.Children[0].SweepAngle).To(BeNumerically(">", layout.Children[1].SweepAngle))
	g.Expect(layout.Children[1].SweepAngle).To(BeNumerically("~", layout.Children[2].SweepAngle, 1e-12))
	g.Expect(layout.Children[2].EndAngle()).To(BeNumerically("==", -math.Pi/2+2*math.Pi))
}

func TestLayoutDoesNotCreateSectorsForFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{
			{
				Name:  "files-only",
				Files: []*model.File{{Name: "main.go"}},
			},
		},
	}

	layout := Layout(root, 800, filesystem.FileLines)

	g.Expect(layout.Children).To(HaveLen(1))
	g.Expect(layout.Children[0].Children).To(BeEmpty())
}

func TestDirectoryMetricValuePrefersQuantityThenMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const name = metric.Name("test")
	dir := &model.Directory{}
	dir.SetMeasure(name, 0.5)
	g.Expect(directoryMetricValue(dir, name)).To(BeNumerically("==", 0.5))

	dir.SetQuantity(name, 2)
	g.Expect(directoryMetricValue(dir, name)).To(BeNumerically("==", 2))
}
