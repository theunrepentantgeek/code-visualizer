package provider_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

//nolint:paralleltest // mutates global base registry
func TestLoadersFor_ReturnsMatchingLoader(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"test-metric"},
		Load: func(_ *model.Directory) error {
			return nil
		},
	})

	loaders := provider.LoadersFor([]metric.Name{"test-metric"})
	g.Expect(loaders).To(HaveLen(1))
	g.Expect(loaders[0].Metrics).To(ContainElement(metric.Name("test-metric")))
}

//nolint:paralleltest // mutates global base registry
func TestLoadersFor_SkipsUnrelatedLoader(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics: []metric.Name{"other-metric"},
		Load:    func(_ *model.Directory) error { return nil },
	})

	loaders := provider.LoadersFor([]metric.Name{"unrelated"})
	g.Expect(loaders).To(BeEmpty())
}
