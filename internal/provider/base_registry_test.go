package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestBaseRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	desc := BaseMetricDescriptor{
		Name:           "file-size",
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Size of each file in bytes.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax},
		DefaultPalette: palette.Neutral,
	}

	reg.register(desc)

	got, ok := reg.get("file-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(got.Name).To(Equal(metric.Name("file-size")))
	g.Expect(got.Level).To(Equal(metric.LevelFile))
}

func TestBaseRegistry_GetUnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()

	_, ok := reg.get("nonexistent")
	g.Expect(ok).To(BeFalse())
}

func TestBaseRegistry_DuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	desc := BaseMetricDescriptor{
		Name: "file-size",
		Kind: metric.Quantity,
	}

	reg.register(desc)
	g.Expect(func() { reg.register(desc) }).To(Panic())
}

func TestBaseRegistry_All(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{Name: "b-metric", Kind: metric.Quantity})
	reg.register(BaseMetricDescriptor{Name: "a-metric", Kind: metric.Measure})

	all := reg.all()
	g.Expect(all).To(HaveLen(2))
	g.Expect(all[0].Name).To(Equal(metric.Name("a-metric")))
	g.Expect(all[1].Name).To(Equal(metric.Name("b-metric")))
}

func TestBaseRegistry_AllForLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{Name: "file-size", Level: metric.LevelFile})
	reg.register(BaseMetricDescriptor{Name: "types", Level: metric.LevelDeclaration})
	reg.register(BaseMetricDescriptor{Name: "file-lines", Level: metric.LevelFile})

	fileMetrics := reg.allForLevel(metric.LevelFile)
	g.Expect(fileMetrics).To(HaveLen(2))

	declMetrics := reg.allForLevel(metric.LevelDeclaration)
	g.Expect(declMetrics).To(HaveLen(1))
	g.Expect(declMetrics[0].Name).To(Equal(metric.Name("types")))
}

func TestBaseRegistry_RegisterWithProvider(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	pd := ProviderDescriptor{
		Name: "go",
		Filters: map[metric.FilterName]string{
			"public": "Exported only",
		},
	}
	desc := BaseMetricDescriptor{Name: "types", Kind: metric.Quantity}

	reg.registerWithProvider(desc, pd)

	got, ok := reg.providerFor("types")
	g.Expect(ok).To(BeTrue())
	g.Expect(got.Name).To(Equal("go"))
	g.Expect(got.HasFilter("public")).To(BeTrue())
}

func TestBaseRegistry_ProviderForUnknown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()

	_, ok := reg.providerFor("nonexistent")
	g.Expect(ok).To(BeFalse())
}

func TestBaseRegistry_Names(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{Name: "z-metric"})
	reg.register(BaseMetricDescriptor{Name: "a-metric"})

	names := reg.names()
	g.Expect(names).To(Equal([]metric.Name{"a-metric", "z-metric"}))
}
