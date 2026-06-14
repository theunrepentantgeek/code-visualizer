package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestProviderDescriptor_HasFilter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pd := ProviderDescriptor{
		Name: "go",
		Filters: map[metric.FilterName]string{
			"public":  "Exported declarations only",
			"private": "Unexported declarations only",
		},
	}

	g.Expect(pd.HasFilter("public")).To(BeTrue())
	g.Expect(pd.HasFilter("private")).To(BeTrue())
	g.Expect(pd.HasFilter("stdlib")).To(BeFalse())
}

func TestBaseMetricDescriptor_SupportsAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	bmd := BaseMetricDescriptor{
		Name:           "file-size",
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Size of each file in bytes.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}

	g.Expect(bmd.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(bmd.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(bmd.SupportsAggregation(metric.AggMode)).To(BeFalse())
}

func TestBaseMetricDescriptor_SupportsFilter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	bmd := BaseMetricDescriptor{
		Name:    "types",
		Kind:    metric.Quantity,
		Level:   metric.LevelDeclaration,
		Filters: []metric.FilterName{"public", "private"},
	}

	g.Expect(bmd.SupportsFilter("public")).To(BeTrue())
	g.Expect(bmd.SupportsFilter("private")).To(BeTrue())
	g.Expect(bmd.SupportsFilter("stdlib")).To(BeFalse())
}
