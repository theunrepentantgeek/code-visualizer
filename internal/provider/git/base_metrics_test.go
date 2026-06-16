package git

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

//nolint:paralleltest // mutates global base registry
func TestRegisterBase_GitMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	RegisterBase()

	cc, ok := provider.GetBase(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(cc.Kind).To(Equal(metric.Quantity))
	g.Expect(cc.Level).To(Equal(metric.LevelFile))
	g.Expect(cc.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMean)).To(BeTrue())

	cd, ok := provider.GetBase(CommitDensity)
	g.Expect(ok).To(BeTrue())
	g.Expect(cd.Kind).To(Equal(metric.Measure))
	g.Expect(cd.Level).To(Equal(metric.LevelFile))
	g.Expect(cd.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(cd.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(cd.SupportsAggregation(metric.AggSum)).To(BeFalse())
	g.Expect(cd.SupportsAggregation(metric.AggMean)).To(BeFalse())
}

//nolint:paralleltest // mutates global base registry
func TestRegisterBase_MatchesProviderDefsMetadata(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	RegisterBase()

	for _, name := range []metric.Name{
		FileAge,
		FileFreshness,
		AuthorCount,
		CommitCount,
		TotalLinesAdded,
		TotalLinesRemoved,
		CommitDensity,
	} {
		baseMetric, ok := provider.GetBase(name)
		g.Expect(ok).To(BeTrue(), string(name))

		def, ok := providerDefs[name]
		g.Expect(ok).To(BeTrue(), string(name))
		g.Expect(baseMetric.Kind).To(Equal(def.kind), string(name))
		g.Expect(baseMetric.Description).To(Equal(def.description), string(name))
		g.Expect(baseMetric.DefaultPalette).To(Equal(def.defaultPalette), string(name))
	}
}

//nolint:paralleltest // mutates global base registry
func TestRegister_RegistersGitBaseMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	Register()

	baseMetric, ok := provider.GetBase(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseMetric.Level).To(Equal(metric.LevelFile))

	baseProvider, ok := provider.GetBaseProvider(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseProvider).To(Equal(GitProvider))
}

//nolint:paralleltest // mutates global base registry
func TestRegister_RegistersConsolidatedGitLoader(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	Register()

	loaders := provider.LoadersFor([]metric.Name{
		FileAge,
		FileFreshness,
		AuthorCount,
		CommitCount,
		TotalLinesAdded,
		TotalLinesRemoved,
		CommitDensity,
	})

	g.Expect(loaders).To(HaveLen(1))
	g.Expect(loaders[0].Metrics).To(ConsistOf(
		FileAge,
		FileFreshness,
		AuthorCount,
		CommitCount,
		TotalLinesAdded,
		TotalLinesRemoved,
		CommitDensity,
	))
	g.Expect(loaders[0].Load).ToNot(BeNil())
}
