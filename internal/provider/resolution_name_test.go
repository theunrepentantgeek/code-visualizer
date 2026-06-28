package provider_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

//nolint:paralleltest // mutates global base registry
func TestResolveName_BareFileMetric(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	filesystem.Register()

	resolved, err := provider.ResolveName("file-size", metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultKind).To(Equal(metric.Quantity))
}

//nolint:paralleltest // mutates global base registry
func TestResolveName_AggregationExpression(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	golang.Register()

	resolved, err := provider.ResolveName("declarations.count", metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultKind).To(Equal(metric.Quantity))
}

//nolint:paralleltest // reads global base registry; serialized with registry mutators
func TestResolveName_UnknownMetric(t *testing.T) {
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	_, err := provider.ResolveName("not-a-real-metric", metric.LevelFile)
	g.Expect(err).To(MatchError(ContainSubstring("unknown base metric")))
}
