package filesystem

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

//nolint:paralleltest // mutates global base registry
func TestRegisterBase_FilesystemMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	RegisterBase()

	fs, ok := provider.GetBase(FileSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(fs.Kind).To(Equal(metric.Quantity))
	g.Expect(fs.Level).To(Equal(metric.LevelFile))
	g.Expect(fs.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(fs.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(fs.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(fs.SupportsAggregation(metric.AggMean)).To(BeTrue())

	fl, ok := provider.GetBase(FileLines)
	g.Expect(ok).To(BeTrue())
	g.Expect(fl.Kind).To(Equal(metric.Quantity))
	g.Expect(fl.Level).To(Equal(metric.LevelFile))
	g.Expect(fl.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(fl.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(fl.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(fl.SupportsAggregation(metric.AggMean)).To(BeTrue())

	ft, ok := provider.GetBase(FileType)
	g.Expect(ok).To(BeTrue())
	g.Expect(ft.Kind).To(Equal(metric.Classification))
	g.Expect(ft.Level).To(Equal(metric.LevelFile))
	g.Expect(ft.SupportsAggregation(metric.AggMode)).To(BeTrue())
	g.Expect(ft.SupportsAggregation(metric.AggDistinct)).To(BeTrue())
	g.Expect(ft.SupportsAggregation(metric.AggSum)).To(BeFalse())
}

//nolint:paralleltest // mutates global base registry
func TestRegister_RegistersFilesystemBaseMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	Register()

	baseMetric, ok := provider.GetBase(FileSize)
	g.Expect(ok).To(BeTrue())
	g.Expect(baseMetric.Level).To(Equal(metric.LevelFile))

	loaders := provider.LoadersFor([]metric.Name{FileSize})
	g.Expect(loaders).To(HaveLen(1))
	g.Expect(loaders[0].Metrics).To(ContainElement(FileSize))
	g.Expect(loaders[0].Load).ToNot(BeNil())
}
