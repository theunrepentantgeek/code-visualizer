package donuttree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/donuttree"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestBuildDonutInks_OmittedBorderUsesFixedFallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}
	root.SetQuantity(metric.Name("file-lines.sum"), 12)

	requested := stages.CollectRequestedMetrics(metric.Name("file-lines.sum"))

	result := donuttree.BuildInks(root, requested, "file-lines.sum", palette.Neutral, "", "")

	g.Expect(result.HasBorderMetric).To(BeFalse())
	g.Expect(result.Border.Info().Kind).To(Equal(inks.KindFixed))
}

func TestBuildDonutInks_ExplicitBorderBuildsDirectoryMetricInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}
	root.SetQuantity(metric.Name("file-lines.sum"), 12)
	root.SetQuantity(metric.Name("file-freshness.sum"), 3)

	requested := stages.CollectRequestedMetrics(
		metric.Name("file-lines.sum"),
		&config.MetricSpec{Metric: "file-freshness.sum"},
	)

	result := donuttree.BuildInks(
		root,
		requested,
		"file-lines.sum",
		palette.Neutral,
		"file-freshness.sum",
		palette.GoodBad,
	)

	g.Expect(result.HasBorderMetric).To(BeTrue())
	g.Expect(result.Border.Info().Kind).To(Equal(inks.KindNumeric))
}

func TestBuildDonutInks_CategoricalDirectoryFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "root"}
	root.SetClassification(metric.Name("file-type.mode"), "go")

	requested := stages.CollectRequestedMetrics(metric.Name("file-type.mode"))

	result := donuttree.BuildInks(root, requested, "file-type.mode", palette.Categorization, "", "")

	g.Expect(result.Fill.Info().Kind).To(Equal(inks.KindCategorical))
}
