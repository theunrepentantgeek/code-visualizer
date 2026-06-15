package golang

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

//nolint:paralleltest // mutates global base registry
func TestRegisterBase_GoMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	RegisterBase()

	types, ok := provider.GetBase(Types)
	g.Expect(ok).To(BeTrue())
	g.Expect(types.Kind).To(Equal(metric.Quantity))
	g.Expect(types.Level).To(Equal(metric.LevelDeclaration))
	g.Expect(types.SupportsFilter("public")).To(BeTrue())
	g.Expect(types.SupportsFilter("private")).To(BeTrue())
	g.Expect(types.SupportsAggregation(metric.AggCount)).To(BeTrue())
	g.Expect(types.SupportsAggregation(metric.AggSum)).To(BeTrue())

	cc, ok := provider.GetBase(CyclomaticComplexity)
	g.Expect(ok).To(BeTrue())
	g.Expect(cc.Kind).To(Equal(metric.Quantity))
	g.Expect(cc.Level).To(Equal(metric.LevelDeclaration))
	g.Expect(cc.SupportsFilter("public")).To(BeTrue())
	g.Expect(cc.SupportsFilter("private")).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMean)).To(BeTrue())

	fl, ok := provider.GetBase(FunctionLength)
	g.Expect(ok).To(BeTrue())
	g.Expect(fl.Kind).To(Equal(metric.Quantity))
	g.Expect(fl.Level).To(Equal(metric.LevelDeclaration))
	g.Expect(fl.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(fl.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(fl.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(fl.SupportsAggregation(metric.AggMean)).To(BeTrue())

	cr, ok := provider.GetBase(CommentRatio)
	g.Expect(ok).To(BeTrue())
	g.Expect(cr.Kind).To(Equal(metric.Measure))
	g.Expect(cr.Level).To(Equal(metric.LevelFile))

	imports, ok := provider.GetBase(Imports)
	g.Expect(ok).To(BeTrue())
	g.Expect(imports.Level).To(Equal(metric.LevelFile))
	g.Expect(imports.SupportsFilter("stdlib")).To(BeTrue())
	g.Expect(imports.SupportsFilter("external")).To(BeTrue())
	g.Expect(imports.SupportsFilter("internal")).To(BeTrue())
	g.Expect(imports.SupportsFilter("public")).To(BeFalse())
}

//nolint:paralleltest // mutates global base registry
func TestRegisterBase_GoProvider_HasFilters(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	RegisterBase()

	pd, ok := provider.GetBaseProvider(Types)
	g.Expect(ok).To(BeTrue())
	g.Expect(pd.Name).To(Equal("go"))
	g.Expect(pd.HasFilter("public")).To(BeTrue())
	g.Expect(pd.HasFilter("private")).To(BeTrue())
}

//nolint:paralleltest // mutates global base registry
func TestRegister_RegistersGoBaseMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	Register()

	baseMetric, ok := provider.GetBase(Types)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseMetric.Level).To(Equal(metric.LevelDeclaration))

	baseProvider, ok := provider.GetBaseProvider(Types)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseProvider).To(Equal(GoProvider))
}
