package goldentest

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestBuildMetricTree_PopulatesEveryFileBaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)

	root := buildMetricTree()
	g.Expect(root.Files).NotTo(BeEmpty())

	// Every file-level base metric must have a value on the first file.
	f := root.Files[0]
	for _, desc := range provider.AllBaseForLevel(metric.LevelFile) {
		switch desc.Kind {
		case metric.Quantity:
			_, ok := f.Quantity(desc.Name)
			g.Expect(ok).To(BeTrue(), "file metric %q (quantity) must be set", desc.Name)
		case metric.Measure:
			_, ok := f.Measure(desc.Name)
			g.Expect(ok).To(BeTrue(), "file metric %q (measure) must be set", desc.Name)
		case metric.Classification:
			_, ok := f.Classification(desc.Name)
			g.Expect(ok).To(BeTrue(), "file metric %q (classification) must be set", desc.Name)
		}
	}

	g.Expect(model.CountFiles(root)).To(BeNumerically(">", 1))
	g.Expect(model.CountDeclarations(root)).To(BeNumerically(">", 0))
	g.Expect(model.CountCommits(root)).To(BeNumerically(">", 0))
}
