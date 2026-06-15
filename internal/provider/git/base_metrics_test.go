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
func TestRegisterBase_MatchesLegacyProviderMetadata(t *testing.T) {
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

		legacyProvider := newProvider(name)
		g.Expect(baseMetric.Kind).To(Equal(legacyProvider.Kind()), string(name))
		g.Expect(baseMetric.Description).To(Equal(legacyProvider.Description()), string(name))
		g.Expect(baseMetric.DefaultPalette).To(Equal(legacyProvider.DefaultPalette()), string(name))
	}
}

//nolint:paralleltest // mutates global provider and base registries
func TestRegister_RegistersGitBaseMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetRegistryForTesting()
	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetRegistryForTesting)
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	Register()

	legacyProvider, ok := provider.Get(CommitCount, metric.File)
	if !ok || legacyProvider == nil {
		t.Fatalf("expected git provider %q to be registered", CommitCount)
	}

	g.Expect(legacyProvider.Name()).To(Equal(CommitCount))

	baseMetric, ok := provider.GetBase(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseMetric.Level).To(Equal(metric.LevelFile))

	baseProvider, ok := provider.GetBaseProvider(CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseProvider).To(Equal(GitProvider))
}
